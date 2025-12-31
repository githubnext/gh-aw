---
"gh-aw": patch
---

Add compile step with security tools to pre-download Docker images and
store compile output for inspection. This ensures security scanning Docker
images (zizmor, poutine) are cached before the workflow analysis phase,
reducing runtime delays during scans.

