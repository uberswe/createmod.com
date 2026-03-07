#!/usr/bin/env bash
set -euo pipefail

# Seed script for local/dev and CI
# Currently performs a robust wait-for-health on PocketBase, then exits.
# Actual data seeding (users, schematics, collections, guides) to be added later
# as the data model stabilizes.
#
# Usage examples:
#   docker compose exec pocketbase pb migrate --apply
#   PB_URL=${PB_URL:-http://localhost:8090} ./dev/scripts/seed.sh
#
# Expected seed data (to be implemented):
# - Admin user and a regular user (user@example.com/password123)
# - A couple of schematics (one paid), one collection, one guide
# - Any counters/initial settings needed for tests
#
# Notes:
# - Prefer idempotent operations so running multiple times is safe.
# - Consider a golden pb_data snapshot for faster cold starts (see TESTING.md).

PB_URL=${PB_URL:-http://localhost:8090}
echo "[seed] PocketBase URL: ${PB_URL}"

# Wait for PB /api/health to respond 200 (up to ~30s)
ATTEMPTS=30
SLEEP=1
for i in $(seq 1 ${ATTEMPTS}); do
  if curl -fsS "${PB_URL}/api/health" >/dev/null; then
    echo "[seed] PocketBase is healthy."
    break
  fi
  echo "[seed] Waiting for PocketBase... (${i}/${ATTEMPTS})"
  sleep ${SLEEP}
  if [[ ${i} -eq ${ATTEMPTS} ]]; then
    echo "[seed] PocketBase health check failed after ${ATTEMPTS} attempts" >&2
    exit 1
  fi
done

# Placeholder: real seeding logic goes here (create users, collections, schematics, etc.)
echo "[seed] No-op seeding for now (to be implemented)."

exit 0
