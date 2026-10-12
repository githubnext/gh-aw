---
"gh-aw": major
---

Default `gh aw logs` to a 1024 MB (1 GB) storage budget to prevent bulk downloads
from exhausting runner disk space. The budget is soft: concurrent in-flight
downloads can temporarily exceed it.

**Breaking change:** Downloads previously used unlimited storage by default.
Use `--max-storage 0` for unlimited storage or pass a larger budget explicitly.
MCP logs calls inherit the new default; explicit `max_storage: 0` remains unlimited.

Prefer compact `info` artifacts when usage data lacks `aw_info.json`, retaining
activation fallback for legacy runs. Shared daily log prefetchers request
`info,usage` explicitly and use a 1 GB storage budget.
