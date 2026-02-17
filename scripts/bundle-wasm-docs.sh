#!/usr/bin/env bash
#
# bundle-wasm-docs.sh -- Build the WebAssembly compiler and copy
# artifacts into the Astro docs site's public directory.
#
# Usage:
#   ./scripts/bundle-wasm-docs.sh
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST_DIR="${REPO_ROOT}/docs/public/wasm"

echo "==> Building gh-aw.wasm..."
cd "${REPO_ROOT}"
make build-wasm

echo "==> Copying artifacts to ${DEST_DIR}..."
mkdir -p "${DEST_DIR}"

cp "${REPO_ROOT}/gh-aw.wasm" "${DEST_DIR}/gh-aw.wasm"

# wasm_exec.js ships with the Go toolchain; fall back to the repo-root copy.
WASM_EXEC_SRC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
if [ ! -f "${WASM_EXEC_SRC}" ]; then
  WASM_EXEC_SRC="${REPO_ROOT}/wasm_exec.js"
fi
cp "${WASM_EXEC_SRC}" "${DEST_DIR}/wasm_exec.js"

echo ""
echo "Done. Files in ${DEST_DIR}:"
ls -lh "${DEST_DIR}/gh-aw.wasm" "${DEST_DIR}/wasm_exec.js"
