---
"gh-aw": patch
---

Add domain blocklist support via `--block-domains` flag; updates frontmatter parsing, engines, tests, and documentation.

This change adds the `blocked` network frontmatter field and ensures the engines pass `--block-domains` to the AWF binary when blocked domains are configured.

