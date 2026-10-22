---
"gh-aw": major
---

Move live outcome evaluation to `gh aw logs --outcomes` and remove `gh aw outcomes`.
For one run, pipe its ID to `gh aw logs --stdin --repo owner/repo --outcomes --json`.
Read per-run observations from `.outcomes[]` and episode summaries from `.episode_outcomes`.
Combine `--episodes --outcomes` to include recorded episode membership and hops
with the automatically joined outcome summaries, without a separate offline join.
Remove historical repository scoring.
Use `gh aw logs outcomes-episodes` for offline joins. JSONL export uses `--outcomes-dir`.

Reuse successful GitHub API reads across selected runs within each collection,
including GraphQL queries and labels already returned on issue or PR objects.
Request 100 items per page for paginated REST reads.
Batch impact-report outcome collection into one CLI invocation. Later collections
still check current state, and failed or partial responses are not retained.
