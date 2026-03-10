---
"gh-aw": patch
---

Fix `__GH_AW_WIKI_NOTE__` placeholder not being substituted when repo-memory is configured without wiki mode. Previously, when `wiki: false`, the variable used a static empty string that could be missing from the substitution step in older compiled workflows. Now it uses `${{ '' }}` (a GitHub expression evaluating to empty string), ensuring expression interpolation always produces an empty value for the placeholder.
