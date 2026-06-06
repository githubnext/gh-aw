---
"gh-aw": patch
---

Extracted Copilot SDK permission helpers into `copilot_sdk_permissions.cjs` and session runner into `copilot_sdk_session.cjs` so other drivers can reuse them. `copilot_sdk_driver.cjs` is now a minimal entry point (~139 lines) that re-exports both modules for backward compatibility.
