---
"gh-aw": patch
---

Refresh the `gh-aw-node` image's Alpine and npm packages and publish it for both amd64 and arm64. Bump npm to 11.19.0 and patch bundled `tar` to ≥7.5.22 and `brace-expansion` to ≥5.0.8 to address container security findings (CVE-2026-58055, CVE-2025-60876, and related npm dependency vulnerabilities).
