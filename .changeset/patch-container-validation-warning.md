---
"gh-aw": patch
---

Treat container image validation as warning instead of error

Container image validation failures during `compile --validate` are now treated as warnings instead of errors. This prevents compilation from failing due to local Docker authentication issues or private registry access problems, while still informing users about potential container validation issues.
