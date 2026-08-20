---
"gh-aw": patch
---

Threat detection runs now default to zero harness retries (`GH_AW_HARNESS_MAX_RETRIES: 0`) instead of inheriting the harness's default of 3 retries with backoff. Detection is a bounded scan of already-completed agent output, so a single failed attempt (for example a sandboxed cleanup command failing inside the read-only `/tmp/gh-aw` mount) no longer silently retries the whole detection run multiple times, burning extra runtime and model spend. Explicit `engine.harness.max-retries` (or `threat-detection.engine.harness.max-retries`) configuration is still honored and overrides this default.
