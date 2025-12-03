---
"gh-aw": patch
---

Add `assign-to-user` safe output type and supporting files (schemas, Go structs, JS implementation, tests, and docs).

This change adds a new safe output `assign-to-user` analogous to `assign-to-agent`, including parser schema, job builder, JavaScript runner script, and tests. It is an internal addition and does not change public CLI APIs.

