---
"gh-aw": patch
---

Rename the `command` trigger to `slash_command` with a deprecation path.

The `slash_command` frontmatter field was added (same validation as the old `command`), and the `command` field remains supported but is marked as deprecated and emits a compile-time warning. Schema, compiler, docs, and workflows were updated to prefer `slash_command` while keeping backward compatibility.

