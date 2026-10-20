#!/usr/bin/env bash
set +o histexpand
set -euo pipefail

# Version evidence only: do not dump environment variables, proxy URLs or headers.
echo "::group::Runtime TLS inventory (not a trust-store attestation)"
if command -v curl >/dev/null 2>&1; then
  if ! curl --version | sed -n '1p'; then
    echo "WARNING: curl version inspection failed; trust status unknown" >&2
  fi
else
  echo "curl: unavailable"
fi
if command -v node >/dev/null 2>&1; then
  if ! node -p '"Node " + process.version + "; OpenSSL " + process.versions.openssl'; then
    echo "WARNING: Node/OpenSSL version inspection failed; trust status unknown" >&2
  fi
else
  echo "Node: unavailable"
fi
if command -v go >/dev/null 2>&1; then
  if ! GOTOOLCHAIN=local go env GOVERSION; then
    echo "WARNING: Go version inspection failed; trust status unknown" >&2
  fi
  for binary in gh gh-aw; do
    if command -v "$binary" >/dev/null 2>&1; then
      if ! GOTOOLCHAIN=local go version -m "$(command -v "$binary")" | sed -n '1p'; then
        echo "WARNING: ${binary} build-toolchain inspection failed; trust status unknown" >&2
      fi
    else
      echo "${binary}: unavailable for build-toolchain inspection"
    fi
  done
else
  echo "Go: unavailable; binary build-toolchain inspection unavailable"
fi
echo "Python/certifi: not inspected by shell setup; record each client's interpreter, TLS backend and selected CA bundle separately."
echo "Container, proxy and MCP trust stores require separate image/vendor evidence."
echo "Browser updates and successful HTTPS connections do not attest CA distrust."
echo "::endgroup::"
