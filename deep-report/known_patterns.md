## DeepReport Memory (2026-08-17T12:22:00Z)

### Third consecutive quiet cycle for discussions/issues (not for workflow logs)
Zero discussions and zero issues changed in the ~6h since the prior cycle (2026-08-17T06:26Z) — same dataset snapshot as last cycle (newest discussion still #53243). This is now a 3-cycle pattern (00:26Z, 06:26Z, 12:22Z all essentially empty on discussions/issues). Workflow logs remain the reliable source of genuinely new data every cycle regardless of discussion/issue cadence — treat discussions/issues as "check but expect empty" and logs as the primary feed until the discussion/issue fetch cadence changes.

### New pattern: a workflow-scoped fix can leave a shared-import-level gap unfixed
#53190/#53194 fixed Serena's missing-Go-toolchain LSP crash for `linter-miner.md` specifically (added `pre-agent-steps` with `actions/setup-go`), but the same `shared/mcp/serena-go.md` import is used by 15 other workflows that never got the fix. Typist hit the identical crash-and-burn (this time burning the entire AI credit budget, not just failing outright) days later. **Lesson: when a fix is filed against a symptom in one workflow but the root cause lives in a shared import/template, check how many other consumers of that import are equally exposed before treating the issue as closed.** This is a new category distinct from the "chronic re-file with no fix" pattern below — here the fix DID work, just too narrowly scoped.

### Confirmed pattern still holding: "label the unlabeled issues" is a non-productive loop
Same 6 unlabeled open issues (#53204, #53136, #52723, #52608, #52575, #52547) for the third cycle running. 7+ near-identical deep-report-filed issues over the project's history, all closed without a durable fix. **Still declining to re-file** — no new root-cause angle has appeared. See [[flagged_items]].

### Reconfirmed practice: verify merge status directly, don't trust search-API fields
`gh api search/issues` returned `merged_at: null` for PR #53194, which looked like "closed without merging" — but `gh api repos/.../pulls/53194` (the direct PR endpoint) showed `merged: true`. The search API's issue-shaped PR objects don't reliably populate `merged_at`. **Always confirm merge status via the direct pulls endpoint before concluding a linked fix didn't land**, especially before deciding whether to file a duplicate.

### `agenticworkflows logs` throughput guidance (from 06:26Z cycle) reconfirmed
Used `count=30` this cycle per the `count<=50` recommendation; completed in ~46s, no timeout issues. No new throughput data needed.

### All other tracked items unchanged this cycle
See `flagged_items.md` for the full carry-forward list — none had a fresh source report appear in this cycle's empty discussion window to re-verify against.
