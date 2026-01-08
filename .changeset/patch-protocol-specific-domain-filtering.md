---
"gh-aw": patch
---

Support protocol-specific domain filtering for `network.allowed` entries.

This change adds validation and compiler integration so `http://` and
`https://` prefixes (including wildcards) are accepted for protocol-specific
domain restrictions. It also preserves protocol prefixes through compilation,
adds unit and integration tests, and updates the documentation.

Fixes githubnext/gh-aw#9040

