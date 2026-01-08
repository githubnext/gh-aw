---
"gh-aw": patch
---

Refactor agentic engine error patterns from Go to JavaScript and load patterns at runtime.

The change centralizes pattern definitions in `actions/setup/js/error_patterns.cjs`, enables runtime
loading based on `GH_AW_ENGINE_ID`, and supports custom patterns via `GH_AW_CUSTOM_ERROR_PATTERNS`.
This reduces compiled workflow size and keeps Go fallbacks for compatibility.
