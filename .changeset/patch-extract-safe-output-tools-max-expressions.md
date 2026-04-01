---
"gh-aw": patch
---

Extract `${{ }}` expressions from `max:` values in the `<safe-output-tools>` prompt block to `GH_AW_*` env vars (matching the existing prompt extraction patterns). This prevents `${{ }}` from appearing inline in the `run:` heredoc, which was subject to the GitHub Actions 21 KB expression-size limit and caused compilation failures for workflows with large prompts that used `${{ inputs.* }}` in safe-output `max:` config values.
