---
"gh-aw": patch
---

Fix shell argument escaping for `--allow-domains`/`--block-domains` when values contain GitHub Actions `${{ }}` expressions by using double quotes so expressions with single-quoted strings remain valid.
