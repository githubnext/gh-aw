---
"gh-aw": patch
---

Add support for workflow imports using `uses`/`with` syntax with `import-schema` validation, including typed input validation and `github.aw.import-inputs.*` expression support in imported content.

Deprecate `tools.serena` in favor of `mcp-servers.serena` via shared Serena workflows, and migrate bundled workflows to `shared/mcp/serena.md` and `shared/mcp/serena-go.md`.
