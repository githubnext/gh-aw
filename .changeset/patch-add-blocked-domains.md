---
"gh-aw": patch
---

Add domain blocklist support via the `--block-domains` flag and the
`blocked` frontmatter field. This enables specifying domains or ecosystem
identifiers to block in workflows and ensures the flag is only added when
blocked domains are present.

Supported engines: Copilot, Claude, Codex.

Ref: githubnext/gh-aw#9063

