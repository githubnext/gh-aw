---
"gh-aw": patch
---

Add structured types for frontmatter configuration parsing and fix integer rendering in YAML outputs.

This change introduces typed frontmatter parsing (`FrontmatterConfig`) to reduce runtime type
assertions and improve error messages. It also fixes integer marshaling so integer fields
(for example `retention-days` and `fetch-depth`) are preserved as integers in compiled YAML.

