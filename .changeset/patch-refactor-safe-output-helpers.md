---
"gh-aw": patch
---

Refactor: Extract duplicate safe-output environment setup logic into helper functions

Extracted duplicated safe-output environment setup code from multiple workflow engines and job builders into reusable helper functions in `pkg/workflow/safe_output_helpers.go`. This eliminates ~123 lines of duplicated code across 4 engine implementations and 5 safe-output job builders, improving maintainability and consistency while maintaining 100% backward compatibility.
