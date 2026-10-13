---
"gh-aw": patch
---

Preserve secret-safe operation stages and allowlisted filesystem error codes when artifact redaction fails. Keep fail-closed removal and permission handling unchanged; these diagnostics improve investigation without claiming recovery from the underlying failure.
