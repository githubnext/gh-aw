---
"gh-aw": patch
---

Remove the `shared/mcp/open-ontologies.md` MCP wrapper and its usage in `glossary-maintainer.md`. The `ghcr.io/fabio-rovai/open-ontologies` container image had Critical/High severity vulnerabilities in its Debian base layer and multiple license policy violations flagged by the daily container image security scan.
