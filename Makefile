.PHONY: up down go-test e2e e2e-up seed pw-install pb-reset

APP_BASE_URL ?= http://localhost:8080
PB_URL ?= http://localhost:8090

up:
	docker compose up --build -d

# Run Playwright against already running services (does not start Docker)
e2e: pw-install
	@echo "[e2e] Checking services at $(APP_BASE_URL) and $(PB_URL) ..."
	@curl -fsS $(PB_URL)/api/health >/dev/null || (echo "[e2e] PocketBase not reachable at $(PB_URL). Start services with 'make e2e-up' or ensure PB is running." && exit 1)
	@curl -fsS $(APP_BASE_URL)/robots.txt >/dev/null || (echo "[e2e] App not reachable at $(APP_BASE_URL). Start services with 'make e2e-up' or ensure the app is running." && exit 1)
	APP_BASE_URL=$(APP_BASE_URL) PB_URL=$(PB_URL) npx playwright test

# Start docker-compose, wait for services, seed, then run tests
e2e-up: up pw-install
	@echo "[e2e-up] Waiting for PocketBase at $(PB_URL) ..."
	@for i in `seq 1 60`; do curl -fsS $(PB_URL)/api/health >/dev/null && break || (sleep 1 && echo "[e2e-up] waiting PB ($$i/60)"); done
	@echo "[e2e-up] Waiting for app at $(APP_BASE_URL) ..."
	@for i in `seq 1 60`; do curl -fsS $(APP_BASE_URL)/robots.txt >/dev/null && break || (sleep 1 && echo "[e2e-up] waiting app ($$i/60)"); done
	PB_URL=$(PB_URL) bash ./dev/scripts/seed.sh
	APP_BASE_URL=$(APP_BASE_URL) PB_URL=$(PB_URL) npx playwright test

pw-install:
	npm ci || npm i
	npx playwright install --with-deps

down:
	docker compose down -v

go-test:
	go test ./...

seed:
	bash ./dev/scripts/seed.sh

pb-reset:
	bash ./dev/scripts/reset_pb_data.sh
