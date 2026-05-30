---
"gh-aw": patch
---

Allow safe output `max`/`expires` fields to accept templated integers so expressions like `${{ inputs.max-issues }}` continue to work in addition to literal numbers.
