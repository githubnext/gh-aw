## DeepReport Memory (2026-08-17T18:23:00Z)

### RESOLVED (root-caused this cycle): the "3 consecutive quiet cycles" were a caching artifact, not reality
`.github/workflows/shared/discussions-data-fetch.md` / `shared/weekly-issues-data-fetch.md` cache their fetch results keyed only by calendar day, but `deep-report` runs every 6h (`cron: "9 */6 * * *"`). Only the day's first run gets live data; the next 3 runs reuse that stale snapshot. This explains why 2026-08-17T06:26Z and 2026-08-17T12:22Z both reported zero new discussions/issues — filed as a fix this cycle. **Until the fix lands, always cross-check the pre-fetched file's newest `updatedAt` against a live spot-check before trusting "no new activity."**

**Live workaround used this cycle** (re-fetch bypassing the stale cache):
```bash
# Discussions (single-line query — multi-line -f query strings and -F query=@file both broke the gh wrapper here)
Q='query { repository(owner: "github", name: "gh-aw") { discussions(first: 100, orderBy: {field: UPDATED_AT, direction: DESC}) { nodes { number title updatedAt createdAt url category { name slug } author { login } labels(first: 5) { nodes { name } } body } } } }'
gh api graphql -f query="$Q" > raw.json
# then jq-extract into the same shape as discussions.json

# Issues: gh issue list --search consistently failed with "malformed version:" (gh 2.97.0, this sandbox's proxy) — use gh api search/issues instead:
gh api -X GET search/issues -f q="repo:github/gh-aw is:issue updated:>=<date>" -f per_page=100 --paginate --jq '.items[] | {...}'
```
Both `gh issue list` (any invocation, even without --search) and `gh api graphql` intermittently threw transient `HTTP 503`/`malformed version:`/`error connecting to api.github.com` — all resolved on retry after a few seconds. Treat as transient proxy flakiness, not a real outage; retry 2-4 times with short sleeps before concluding the API is down.

### New pattern: auto-expiring failure-notification issues can mask a genuinely recurring bug
The `github-remote-mcp-auth-test` workflow's failure issues (#51703, #52245) carry an expiry tag and auto-close ~weekly regardless of whether the root cause was fixed — so a real, recurring MCP-toolset-unavailable bug (3rd+ occurrence as of today's discussion #53314/#53058) never accumulated into a persistent tracking issue. **Lesson: when a "chronic, no durable fix" pattern involves auto-expiring issues specifically, check whether the expiry mechanism itself is why no fix ever lands (nobody is actually looking at a live, growing issue) — this is a different remediation than the standard "duplicate, decline to re-file" case.** Filed a new non-expiring tracking issue this cycle to break the cycle.

### Confirmed pattern (carried forward, not re-verified this cycle): "label the unlabeled issues" is a non-productive loop
As of the 2026-08-17T12:22Z cycle: same 6 unlabeled open issues (#53204, #53136, #52723, #52608, #52575, #52547), 7+ near-identical deep-report-filed issues over the project's history, all closed without a durable fix. Issues-analyst sub-agent pass was skipped this cycle (see last_analysis_timestamp.md) so this wasn't re-checked live — still declining to re-file absent a fresh full pass. See [[flagged_items]].

### Reconfirmed practice: verify merge status directly, don't trust search-API fields
`gh api search/issues` unreliably populates `merged_at` for PRs — always confirm via the direct `gh api repos/.../pulls/{n}` endpoint before concluding a linked fix didn't land, especially before deciding whether to file a duplicate.

### `agenticworkflows logs` throughput guidance
Not used this cycle (workflow-log sampling deferred in favor of live discussion/issue re-fetch + CI-regression root-cause work, since diagnosing the stale-cache meta-bug consumed the analysis budget). Prior guidance (`count<=50`, ~46s for count=30) still stands for next cycle.
