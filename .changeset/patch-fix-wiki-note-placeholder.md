---
"gh-aw": patch
---
Fix `__GH_AW_WIKI_NOTE__` placeholder not being substituted when repo has no GitHub Wiki enabled. The activation job now adds `GH_AW_WIKI_NOTE` to the placeholder substitution step even when wiki mode is disabled (empty string). The `validate_prompt_placeholders.sh` script also applies a fallback substitution for this placeholder with a warning, ensuring backwards compatibility with workflows compiled before this fix.
