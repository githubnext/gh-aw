---
"gh-aw": major
---

Harden executable bootstrap against compromised certificate authorities with trusted release-manifest digests, HTTPS-only redirects, pinned Compose and apt-key bytes, patched Go release toolchains, and bounded runtime TLS inventory.

**Breaking change:** Unreviewed AWF/Copilot versions and `latest` downloads fail closed. Use a reviewed pinned release or add independently authenticated, release-bound manifest digests to the trusted setup action's `release_integrity.sh` and publish that action through the normal release process. Trusted runner toolcache selection remains unchanged. See security architecture §6.8 for emergency distrust and per-runtime ownership.
