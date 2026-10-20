---
"gh-aw": major
---

Harden executable bootstrap against compromised certificate authorities with trusted release-manifest digests, HTTPS-only redirects, pinned Compose and apt-key bytes, patched Go release toolchains, and bounded runtime TLS inventory.

**Breaking change:** Unreviewed AWF/Copilot downloads without a supplied pin and `latest` downloads fail closed. Use a reviewed built-in release or provide an explicit version with `sandbox.agent.checksums-sha256` (AWF `checksums.txt`) or `engine.checksums-sha256` (Copilot `SHA256SUMS.txt`). These literal frontmatter pins propagate to matching detection/eval installations and must come from an independently trusted channel. Explicit Copilot pins bypass the toolcache; cache behavior without a supplied pin is unchanged. Updating the trusted setup action's built-in digest table remains supported. See security architecture §6.8 for emergency distrust and per-runtime ownership.
