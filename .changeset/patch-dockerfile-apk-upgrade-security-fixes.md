---
"gh-aw": patch
---

Add `apk upgrade --no-cache` to the Dockerfile to pick up patched Alpine packages.

This addresses vulnerabilities found in the `ghcr.io/github/gh-aw-mcpg` container image scan:
- CVE-2025-60876 (busybox/ssl_client, Medium)
- CVE-2022-3219 (gnupg, Low)
- GHSA-f5mr-q85p-6hh6 (sigstore/fulcio in github-cli binary, High) — resolved via updated alpine github-cli package
- GHSA-xjvp-4fhw-gc47 (opencontainers/runc in github-cli binary, Medium) — resolved via updated alpine github-cli package
