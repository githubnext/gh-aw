---
"gh-aw": patch
---

Fix YAML boolean keyword quoting to prevent workflow validation failures

Fixed the compiler to prevent unquoting the "on" key in generated workflow YAML files. This prevents YAML parsers from misinterpreting "on" as the boolean value `True` instead of a string key, which was causing GitHub Actions workflow validation failures. The fix ensures all compiled workflows generate valid YAML that passes GitHub Actions validation.
