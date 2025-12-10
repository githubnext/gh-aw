---
"gh-aw": patch
---

Added support for passing workflow inputs to `gh aw run` via the new `--raw-field` (`-f`) flag. This accepts `key=value` pairs and forwards them to `gh workflow run` as `-f key=value` arguments. The implementation validates input formatting and provides clear error messages for malformed inputs.

