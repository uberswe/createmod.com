# Testing Plan (Local + CI)

Purpose: Provide a complete, actionable plan to implement reliable local and CI testing for the CreateMod.com web app, validating the frontend against the backend (PocketBase) with realistic data and browser-driven flows.

This plan is incremental. You can implement it in phases and still gain value at each step.


## 1) Prerequisites
- Go 1.22+
- Docker and Docker Compose v2
- Node.js 18+ and pnpm or npm (Playwright requires Node)
- Make (optional, for convenience targets)


## 2) Test Layers Overview
We will use a layered approach:

- Fast HTTP/component tests in Go
  - Location: internal/... with _test.go files
  - Use internal/testutil for lightweight server, auth behaviors, and regression coverage for HTMX headers, redirects, etc.

- End-to-End (E2E) browser tests with Playwright
  - Location: tests/e2e
  - Drive the real app in a browser against a real PocketBase + MailHog stack.
  - Assert UI flows, downloads, emails, counters, i18n, collections, etc.

- Contract tests (optional)
  - Ensure response shapes for public API endpoints remain stable.

- Accessibility and Visual Regression
  - Axe-core via @axe-core/playwright.
  - Playwright snapshot testing for key pages.


## 3) Local Stack with docker-compose
Create docker-compose.yml at repo root to orchestrate the stack.

Example (adjust volumes/paths to your repo structure):

```yaml
version: "3.8"

services:
  pocketbase:
    image: ghcr.io/pocketbase/pocketbase:latest
    command: ["serve", "--http=0.0.0.0:8090"]
    ports: ["8090:8090"]
    volumes:
      - ./dev/pb_data:/pb_data
      - ./dev/pb_migrations:/pb_migrations
      - ./dev/pb_hooks:/pb_hooks
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8090/api/health"]
      interval: 2s
      timeout: 2s
      retries: 30

  mailhog:
    image: mailhog/mailhog:latest
    ports:
      - "8025:8025" # UI
      - "1025:1025" # SMTP

  app:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      - POCKETBASE_URL=http://pocketbase:8090
      - SMTP_HOST=mailhog
      - SMTP_PORT=1025
      - APP_BASE_URL=http://localhost:8080
      - ENV=local
    ports: ["8080:8080"]
    depends_on:
      pocketbase:
        condition: service_healthy
      mailhog:
        condition: service_started
    volumes:
      - .:/app
```

Notes:
- The app must use http://pocketbase:8090 inside Docker.
- If the app makes outbound calls (e.g., YouTube thumbnails), consider allowing network in tests or stubbing.


## 4) Data Seeding and Reset Strategy
Deterministic data makes tests reliable and fast.

Two approaches:

- Golden snapshot (recommended for speed):
  - Prepare a fully-migrated pb_data directory with seed data in dev/golden_pb_data.
  - For each local run or Playwright worker, copy the golden snapshot to a temp dir (e.g., dev/pb_data_run_<ts>), mount it to PocketBase.
  - This avoids re-running migrations on every run.

- Migration/seed scripts (simpler to reason about):
  - Maintain PocketBase migrations in dev/pb_migrations and seed scripts that create the test users, collections, schematics, guides, reports, and counters.
  - Run seeds at stack startup or as a dedicated step before tests.

Seed contents to include:
- Accounts: at least one admin and one regular user.
- Schematics: several examples with variations; ensure at least one paid schematic.
- NBT fixtures: store small .nbt sample files under tests/fixtures.
- Collections: one featured, a few regular; set up ordering and images.
- Guides: at least one example with markdown; include a video link.
- Reports: none initially; tests will create them.
- API Keys: at least one pre-generated key for contract tests, or generate in test setup.

Reset mechanism:
- For golden snapshot approach, copying a fresh dir per run is the reset.
- For scripts, provide a script that wipes/repaves PocketBase collections.


## 5) Repository Structure for Tests
- Go HTTP/component tests (already present):
  - internal/pages/*_test.go
  - internal/testutil/*

- Playwright tests:
  - tests/e2e/
    - fixtures/
      - auth.ts (sets auth cookie or performs login via PB REST)
    - specs/*.spec.ts
    - fixtures files: tests/fixtures/*.nbt, images, zips
  - playwright.config.ts


## 6) Playwright Setup
Install Playwright dependencies:

- Initialize:
  - npm init -y (if not already present)
  - npm i -D @playwright/test @axe-core/playwright pocketbase
  - npx playwright install --with-deps

- playwright.config.ts example:
```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60_000,
  retries: 1,
  use: {
    baseURL: process.env.APP_BASE_URL || 'http://localhost:8080',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  workers: 4,
});
```

- Auth fixture (adjust cookie name to match the app; current tests indicate `create-mod-auth`):
```ts
// tests/e2e/fixtures/auth.ts
import { test as base } from '@playwright/test';
import PocketBase from 'pocketbase';

type Fixtures = {
  userPage: any;
  adminPage: any;
};

export const test = base.extend<Fixtures>({
  userPage: async ({ browser }, use) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    // Fast-path: authenticate via PB REST, then set cookie app expects
    const pb = new PocketBase(process.env.PB_URL ?? 'http://localhost:8090');
    const auth = await pb.collection('users').authWithPassword('user@example.com', 'password123');

    await context.addCookies([
      {
        name: 'create-mod-auth', // set to real cookie name if different
        value: auth.token,
        domain: 'localhost',
        path: '/',
        httpOnly: true,
      },
    ]);

    await use(page);
    await context.close();
  },
});
```

- Example specs to implement first:
  1) logout works for normal and HTMX flows
  2) upload → private link → make public → interstitial token single-use
  3) collections DnD reorder persists
  4) schematic page a11y has no serious violations

Use @axe-core/playwright for accessibility scans.


## 7) E2E Flows to Cover (Acceptance Checklist)
Map 1:1 with features in TODO.md. Each item must have at least one green E2E test.

1) Authentication & HTMX
- Header/sidebar parity after login and HTMX navigation
- GET /logout clears cookie; HTMX gets HX-Redirect

2) Upload & NBT pipeline
- Upload valid NBT shows stats; private preview link works
- Duplicate detection shows friendly message
- Make public redirects to pending moderation
- Multiple NBTs per schematic → zip download contains all files, names correct
- Version history and diff visible

3) Download interstitial & tokens
- Interstitial waits ~5s then downloads
- Manual link fallback works
- Token is single-use
- Paid schematics: no site download, external link interstitial only

4) Collections
- CRUD flows including banner image
- Add schematic to collection, appears in listing
- Sorting persists; DnD reorder updates positions
- Featured collections visible; downloads increment collection and schematic counters

5) Moderation & reporting
- Report modal submits; DB entry created; MailHog receives email
- Blacklisted schematics hidden from public; downloads blocked

6) Content: Guides & Videos
- Guide editor saves markdown; rendered view; search/filter; view counter increments
- Videos list unique YouTube videos; thumbnail present (or stub); actions navigate

7) API keys & docs
- Generate/revoke API keys; call an endpoint with key; counters/limits update
- API docs page renders examples

8) Internationalization
- Language dropdown sets cookie; UI strings switch
- Browser-detected default language without cookie
- Upload translation path: unsupported language converts to English

9) UX/UI & accessibility
- Home shows trending then recent; spacing consistent
- Axe-core no serious violations on critical pages
- Visual snapshots for critical pages

10) Pagination
- All listings paginate correctly, including HTMX variants

11) Regression for prior template errors
- Ensure all referenced partials exist and render (e.g., schematic_card_full.html)

For each of the above, add database-level assertions via PocketBase REST where applicable (views, downloads, keys usage).


## 8) Go Tests (Component/HTTP) Strategy
- Continue to expand internal/pages/*_test.go using internal/testutil.
- Use these for:
  - Logout HTMX header behaviors
  - Upload form validations and duplicate checks
  - Basic counters and redirects
- Keep tests hermetic and fast by using the in-memory test server; only use E2E when browser/UI or PocketBase state is essential.

Commands:
- go test ./...
- go test ./internal/pages -count=1 -v


## 9) Running Locally
- Start stack: `docker compose up --build -d`
- Wait for pocketbase healthcheck.
- Seed data if using scripts: `docker compose exec pocketbase pb migrate --apply && ./dev/scripts/seed.sh`
- Run E2E tests:
  - `npx playwright test`
  - To view: `npx playwright show-report`
- Run Go tests: `go test ./...`

Makefile (optional):
```
.PHONY: up down test go-test e2e seed
up:
	docker compose up --build -d

down:
	docker compose down -v

go-test:
	go test ./...

e2e:
	npx playwright test

seed:
	./dev/scripts/seed.sh
```


## 10) CI Integration
Use GitHub Actions as an example (adapt if using another CI):

.github/workflows/tests.yml
```yaml
name: tests
on:
  push:
  pull_request:

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    services:
      pocketbase:
        image: ghcr.io/pocketbase/pocketbase:latest
        ports:
          - 8090:8090
        options: >-
          --health-cmd="wget -qO- http://localhost:8090/api/health || exit 1" \
          --health-interval=2s --health-timeout=2s --health-retries=30
      mailhog:
        image: mailhog/mailhog:latest
        ports:
          - 8025:8025
          - 1025:1025
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Go tests
        run: go test ./...
      - uses: actions/setup-node@v4
        with:
          node-version: 18
      - name: Install Playwright
        run: |
          npm ci || npm i -D @playwright/test @axe-core/playwright pocketbase
          npx playwright install --with-deps
      - name: Start app
        run: |
          docker build -t app:local .
          docker run -d --network host -e POCKETBASE_URL=http://localhost:8090 -e SMTP_HOST=localhost -e SMTP_PORT=1025 -e APP_BASE_URL=http://localhost:8080 -e ENV=ci --name app app:local
      - name: Wait for app
        run: |
          for i in `seq 1 60`; do curl -fsS http://localhost:8080/ && break || sleep 1; done
      - name: Playwright tests
        env:
          APP_BASE_URL: http://localhost:8080
          PB_URL: http://localhost:8090
        run: npx playwright test
      - name: Upload Playwright report on failure
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: playwright-report
```

Artifacts: Collect Playwright traces, screenshots, server and PB logs on failure.


## 11) Observability, Isolation, Performance
- Enable structured logs in the app; include request IDs for E2E correlation.
- For Playwright, enable tracing/video on failure and store artifacts.
- Data isolation:
  - Prefer golden pb_data copy per test run or per worker (PB_DATA_DIR suffixed with worker index).
- Speed:
  - Run Playwright with workers (3–6) based on machine capacity.
  - Keep fixtures small (tiny NBT files).


## 12) Implementation Tasks (Checklist)
Short-term (Phase 1):
- [ ] Add docker-compose.yml with pocketbase, app, mailhog.
- [ ] Create dev/golden_pb_data or seed scripts; document reset.
- [ ] Add Playwright to repo (package.json, config, install step).
- [ ] Implement auth fixture; confirm cookie name aligns with app (create-mod-auth or actual).
- [ ] Write initial E2E specs:
  - [ ] Auth parity & logout (normal + HTMX)
  - [ ] Upload → preview → make public (happy path)
  - [ ] Interstitial token single-use
  - [ ] Collections add + DnD reorder
  - [ ] A11y scan for schematic page
- [ ] Add tests/fixtures with sample .nbt file(s).
- [ ] Document commands in README/TESTING.md.

Phase 2:
- [ ] Expand E2E coverage for moderation/reporting with MailHog assertions.
- [ ] Cover paid schematics path + external interstitial.
- [ ] API keys flows + usage counters assertions via PB REST.
- [ ] i18n switching and defaults.
- [ ] Pagination on all listings including HTMX.
- [ ] Visual snapshot tests on critical screens.

Phase 3 (Optional/Polish):
- [ ] Contract tests for public API shapes.
- [ ] Lighthouse CI performance checks.
- [ ] Parallel PB datasets per worker for full isolation.

Acceptance Criteria:
- Green Go test suite via `go test ./...`.
- Green Playwright suite locally and in CI with recorded artifacts on failure.
- Deterministic seeded data; repeatable runs without manual cleanup.


## 13) Roles and Ownership
- Backend/Infra: docker-compose, seeds, PB migrations, app configs.
- QA/Frontend: Playwright specs, a11y and visual checks, fixtures.
- Shared: Contract tests, CI pipelines, and test data design.


## 14) Troubleshooting
- App cannot reach PocketBase in Docker: ensure POCKETBASE_URL uses the service name (pocketbase) inside the compose network.
- HTMX redirect mismatches: verify HX-Redirect headers in server responses.
- Downloads in headless browsers: ensure proper wait for download events and increase timeout on slow CI.
- Flaky tests: add retries at test level; avoid time-dependent logic where possible; prefer polling for counters.


## 15) Glossary
- PB: PocketBase
- HTMX: Hypermedia-driven interactions via headers like HX-Request and HX-Redirect
- E2E: End-to-end browser automation tests
