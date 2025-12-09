---
"gh-aw": patch
---

Convert the safe outputs MCP server to run as a Node process (follow safe inputs pattern). Refactor bootstrap, write modules as individual `.cjs` files, add tests, fix log directory and environment variables, improve ingestion logging, and remove premature config cleanup so ingestion can validate outputs correctly.
