# CreateMod.com Improvements

## Priorities / Next Up
- [x] Fix inconsistent auth state (cookie vs session) so header/sidebar agree
- [x] Split upload flow and implement NBT validation + private link preview
- [x] Add basic e2e/component tests for critical flows (auth, upload, view, download)

---

## Engineering Quality & Infrastructure
- [x] Tests
  - [x] Establish test harness (lightweight internal/testutil HTTP server for regression tests)
  - [x] Seed minimal fixtures (user, schematic, comment)
  - [x] Auth flow tests (logout normal vs HTMX; header/sidebar parity)
  - [x] Upload flow tests (nbt upload, validation, duplicate detection, private preview, make-public redirect)
  - [x] Schematic page tests (view counter, download counter)
  - [x] Reports flow tests (POST /reports normal vs HTMX redirects)
- [x] Authentication consistency
  - [x] Audit current cookie + session usage and single source of truth
  - [x] Standardize middleware/handlers for auth detection (server‑side + HTMX logout behavior)
  - [x] Ensure header, sidebar, and HTMX fragments consume the same auth state (parity tests)
  - [x] Add regression test covering mixed/edge cases (HTMX search parity + logout)

## Upload & NBT Pipeline
- [x] Multi‑step flow
  - [x] Page 1: Upload NBT file
  - [x] Backend validation using mcnbt (https://github.com/uberswe/mcnbt)
  - [x] Extract stats: size, block count, materials list (item id + count)
  - [x] Page 2: Show private link + stats; anyone with link can view/download
  - [x] Page 2: “Make Public” form with usual fields
  - [x] Page 2: Sidebar shows schematic stats
  - [x] Page 3: After submit, show moderation pending screen
- [x] Duplicate detection
  - [x] Hash each NBT file on upload and check for existing
  - [x] User‑facing message with moderation/blacklist guidance and contact link
- [x] Multiple NBTs per schematic
  - [x] Data model to associate many NBT files (variations) to a schematic
  - [x] Will download a zip file with all schematic files
- [x] Versioning
  - [x] Preserve historical versions and expose public history on schematic page
  - [x] Diff/notes between versions (minimal version metadata)
- [x] Scheduling
  - [x] Allow scheduling publish time (respect user timezone or explicit UTC)

## Moderation & Reporting
- [x] Reports
  - [x] Report button on schematics and comments (future: collections, guides)
  - [x] Modal with reason; submit stores in DB
  - [x] Email to superadmin on report submit
  - [x] Admin UI to review/resolve reports
- [x] Blacklisting
  - [x] API hides blacklisted schematics in public API endpoints
- [x] Profile page: “Request schematic blacklisting”
- [x] Bulk upload many NBTs; store hashes in separate table
- [x] Ensure blacklisted files cannot be downloaded

## Collections
- [x] Collections CRUD
  - [x] List all collections page + search filters
  - [x] Create collection (name, description, banner image)
  - [x] Add schematic to collection from schematic page
  - [x] Sorting options
  - [x] Drag-and-drop reorder
  - [x] Featured collections (superadmin)
  - [x] View counters on collections
  - [x] Download collection: zip of all schematics
  - [x] Increment both collection and individual schematic download counts

## Downloads & Monetization
- [x] Download interstitial page
  - [x] 5s wait after page load then auto‑start download
  - [x] Manual download link fallback
  - [x] Track downloads as separate metric from views
  - [x] Ad slot(s) on this page
  - [x] Generate a token when the interstitial download page is shown which is valid for 1 download
- [x] Paid schematics
  - [x] Flag schematics as paid
  - [x] Protect download URL (no site download)
  - [x] “Paid” icon in search and schematic page
  - [x] “Get Schematic” external link (no download counter, only views)
  - [x] Interstitial warning page before external links (also used site‑wide)
- [x] External link interstitial (site‑wide)
  - [x] Generic page to warn users and optionally show ads

## Content: Guides & Videos
- [x] Guides
  - [x] Markdown (or WYSIWYG like TinyMCE) editor for guides
  - [x] Guides listing page + search filters
  - [x] Example guides (uploading, getting started with Create)
  - [x] Optional video link in guide
  - [x] View counters for guides
- [x] Videos
  - [x] Videos page listing unique YouTube videos referenced by schematics
  - [x] Fetch and show thumbnails
  - [x] Actions: “View on YouTube”, “View schematic”
  - [x] Search integration and filters

## API & Developer Experience
- [x] API keys
  - [x] Generate/revoke keys in user settings
  - [x] Usage tracking per key: requests count
  - [x] Usage tracking per key: errors and rate limits
- [x] Endpoints
  - [x] Search schematics
  - [x] Schematic detail: images, size, blocks, materials list (no full NBT)
- [x] API Docs page
  - [x] How to generate keys; example requests; limits; code samples
  - [x] Show user’s usage stats on profile

## Internationalization (i18n)
- [x] Language support
  - [x] Languages: en, pt-BR, pt-PT, es, de, pl, ru, zh-Hans
  - [x] Header language dropdown; cookie to persist selection
  - [x] Use browser‑detected language as default
- [x] Content translation
  - [x] Detect upload language; if unsupported, convert to English
  - [x] Use ChatGPT for creating translations during upload
  - [x] Backfill process to translate past schematics
  - [x] Language files for UI pages/strings

## UX/UI & Frontend
- [x] Site design refresh (keep colors similar)
  - [x] Home shows trending first, then recent
  - [x] Remove tags on main page
  - [x] Subtle visual polish pass (variables + spacing)
- [x] HTMX consistency
  - [x] Ensure partials reflect auth state and pagination filters
- [x] Accessibility checks for major flows

## User Accounts & Gamification
- [x] Contributor status
  - [x] Users with uploaded schematics get contributor badge
  - [x] Hide ads for contributors site‑wide
- [x] Achievements
  - [x] Default Minecraft face avatar for all users
  - [x] Earn achievements for uploads (first upload)
  - [x] Earn achievements for comments (first comment)
  - [x] Earn achievements for guides
  - [x] Earn achievements for collections
  - [x] Earn achievements for views
  - [x] Points unlock skins/accessories (goggles, wrench, mustache, etc.)
  - [x] Profile customization UI for avatar

## Pagination
- [x] Add pagination for all pages and search results
- [x] Schematics listing
- [x] Search results
- [x] Collections
- [x] Guides
- [x] Videos
- [x] Users

## Errors that need to be fixed
- [x] when viewing a schematic page: ERROR GET /schematics/silverfish-xp-farm-with-killing-and-collection-system - html/template:schematic.html:379:44: no such template "schematic_card_full.html"

### Decisions & Open Questions
- Decisions
  - Canonical auth source: PocketBase token via cookie (auth.CookieName) -> e.Auth set in cookieAuth middleware. Templates must derive state from DefaultData.Populate(e). 
  - Implement GET /logout that clears the auth cookie (Path=/, Max-Age=0) and redirects to "/"; also support HX-Redirect for HTMX requests.
  - Use "/profile" as the canonical self-profile link in header and sidebar. Keep /author/{username} for public profiles navigated from content.
  - Update all Logout UI elements to actual href="/logout" links (remove JS-only placeholders).
- Open questions
- Risks / Notes
  - HTMX navigation after logout: ensure client receives HX-Redirect header to avoid partial update mismatches.
  - Confirm that all server-rendered endpoints include commonTemplates so auth state is consistently present in header and sidebar.
