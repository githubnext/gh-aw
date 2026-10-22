---
"gh-aw": major
---

Move live outcome evaluation to `gh aw logs --outcomes` and remove `gh aw outcomes`.
For one run, pipe its ID to `gh aw logs --stdin --repo owner/repo --outcomes --json`.
Read per-run observations from `.outcomes[]` and episode summaries from `.episode_outcomes`.
Use `gh aw logs outcomes-history` for historical scoring and
`gh aw logs outcomes-episodes` for offline joins. JSONL export uses `--outcomes-dir`.
