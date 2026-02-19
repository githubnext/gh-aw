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

# wasm_exec.js ships with the Go toolchain; try both known locations then fall back to repo-root copy.
WASM_EXEC_SRC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
if [ ! -f "${WASM_EXEC_SRC}" ]; then
  # Go 1.24+ moved wasm_exec.js to lib/wasm/
  WASM_EXEC_SRC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
fi
if [ ! -f "${WASM_EXEC_SRC}" ]; then
  # Search the Go module cache as a fallback
  WASM_EXEC_SRC="$(find "$(go env GOPATH)/pkg/mod" -name wasm_exec.js -path "*/go1.*/lib/wasm/*" 2>/dev/null | sort -V | tail -1)"
fi
if [ ! -f "${WASM_EXEC_SRC}" ]; then
  WASM_EXEC_SRC="${REPO_ROOT}/wasm_exec.js"
fi
cp "${WASM_EXEC_SRC}" "${DEST_DIR}/wasm_exec.js"

# Generate brotli-compressed version for smaller transfers
# GitHub Pages serves .br files automatically when Accept-Encoding: br is present
if command -v brotli &> /dev/null; then
  echo "==> Compressing WASM with brotli..."
  brotli -k -q 11 "${DEST_DIR}/gh-aw.wasm"
  echo "Compressed: $(du -h "${DEST_DIR}/gh-aw.wasm.br" | cut -f1) (from $(du -h "${DEST_DIR}/gh-aw.wasm" | cut -f1))"
else
  echo "Warning: brotli not found. Install with: apt-get install brotli (or brew install brotli)"
  echo "Falling back to gzip compression..."
  gzip -k -9 "${DEST_DIR}/gh-aw.wasm"
  echo "Compressed: $(du -h "${DEST_DIR}/gh-aw.wasm.gz" | cut -f1) (from $(du -h "${DEST_DIR}/gh-aw.wasm" | cut -f1))"
fi

echo ""
echo "Done. Files in ${DEST_DIR}:"
ls -lh "${DEST_DIR}/gh-aw.wasm" "${DEST_DIR}/wasm_exec.js"
