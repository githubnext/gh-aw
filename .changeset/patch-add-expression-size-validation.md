---
"gh-aw": patch
---

Add compiler validation for GitHub Actions 21KB expression size limit

The compiler now validates that expressions in generated YAML files don't exceed GitHub Actions' 21KB limit. This prevents silent failures at runtime by catching oversized environment variables and expressions during compilation. When violations are detected, compilation fails with a descriptive error message and saves the invalid YAML to `*.invalid.yml` for debugging.
