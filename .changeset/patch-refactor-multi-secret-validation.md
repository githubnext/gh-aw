---
"gh-aw": patch
---

Refactor multi-secret validation into a shared shell script and simplify generator.

Replaced duplicated inline validation logic in compiled workflows with
`actions/setup/sh/validate_multi_secret.sh`, updated `pkg/workflow/agentic_engine.go`
to invoke the script, and adjusted tests and documentation accordingly.

This reduces repeated validation code across compiled workflows and centralizes
validation logic for easier maintenance and testing.

