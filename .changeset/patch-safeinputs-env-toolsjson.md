---
"gh-aw": patch
---

Document required environment variable names for safe-inputs `tools.json` and
delete the file immediately after loading to avoid leaving secrets on disk.

The `tools.json` file now contains only environment variable names (e.g.
`"GH_TOKEN": "GH_TOKEN"`) and the server removes the file after reading it.

