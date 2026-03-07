#!/usr/bin/env bash
set -euo pipefail

# Reset PocketBase data directory from golden snapshot
# This speeds up local/dev testing by restoring a known-good pb_data state.
#
# Golden snapshot source: dev/golden_pb_data
# Target (mounted by docker-compose): dev/pb_data
#
# Usage:
#   ./dev/scripts/reset_pb_data.sh
#   make pb-reset
#
# Notes:
# - This will delete dev/pb_data contents.
# - If golden snapshot is empty, this effectively empties pb_data.
# - You can create/update the golden snapshot by stopping PocketBase and
#   copying a prepared pb_data into dev/golden_pb_data.

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
SRC="${ROOT_DIR}/dev/golden_pb_data"
DST="${ROOT_DIR}/dev/pb_data"

if [[ ! -d "${SRC}" ]]; then
  echo "[pb-reset] Golden snapshot directory not found: ${SRC}" >&2
  exit 1
fi

mkdir -p "${DST}"

echo "[pb-reset] Clearing ${DST}"
# Remove contents but keep directory itself
rm -rf "${DST:?}/"*
rm -rf "${DST}/."??* || true

echo "[pb-reset] Copying from ${SRC} to ${DST}"
# Use rsync if available for speed; otherwise fallback to cp -a
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete "${SRC}/" "${DST}/"
else
  cp -a "${SRC}/." "${DST}/"
fi

echo "[pb-reset] Done. You can now start docker-compose and PocketBase will use the reset data."
