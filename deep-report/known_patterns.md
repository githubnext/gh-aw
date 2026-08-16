## DeepReport Memory (2026-08-16T18:20:00Z)

### Standing practice (reconfirmed, load-bearing again this cycle)
"Closed status on a code-quality issue is not evidence the fix landed — spot-check source directly before assuming 'already fixed'." Caught 2 more stale closures this cycle: Design Decision Gate's `pr_number`/workflow_dispatch hard-fail (#52987 closed, bug still live in source), Sentrux firewall `api.sentrux.dev` block (2 prior closed fixes, still missing from network.allowed). Also applied in reverse this cycle: verified a *reported* gap (Anthropic WIF "undocumented") was actually a false positive by reading source directly — same discipline, both directions.

### Design Decision Gate: two distinct root causes, don't conflate
(1) LLM invocation-cap — genuinely fixed by merged #52836. (2) `pr_number` codegen hard-fail on bare workflow_dispatch — separate bug, still broken despite #52987's closure. Re-filed cleanly scoped to (2) only, with an explicit note distinguishing it from the already-fixed (1). Lesson: a workflow having "a fix merged" doesn't mean all its failure modes are resolved.

### Fleet reliability jumped sharply (96.93%/97.26% vs 79.5-82.1%)
Partly explained by audit-workflows resuming after a 40-day gap (catch-up sampling), but corroborated by multiple independent signals. Treat as probably-real but keep sampling next cycle before fully trusting the new baseline.

### Resolved watch-items this cycle (retire from active tracking)
- #52518 shared PR-review infra flakiness — closed as predicted.
- Monitoring-staleness meta-theme (3 agents flagged 08-14) — resolved via catch-up runs, did not recur a 4th time.
- Sentrux `no_god_files` enforcement gap (flagged 08-13) — now actively firing (3 god files found).

### New cross-engine signal: 0-turn Execute CLI crash spreading
Previously Copilot-only signature (`copilot-sdk-driver-failures`, improving trend). This cycle: first-ever instances on Aider and Crush. Small sample, filed as an investigation task rather than assumed-confirmed.

### Chronic pattern, deliberately not re-filed (16th would-be duplicate)
Docs "jargon before first use" complaint (frontmatter/WIF/lock.yml terms used before definition) — filed and closed 15+ times since February across recurring Documentation Noob Test discussions, never durably fixed. Same treatment as the `agenticworkflows logs` timeout bug: flag in report body, don't spend an issue slot on a 16th duplicate. Consider recommending a different escalation path (direct maintainer ping) if this persists past another cycle or two.

### `agenticworkflows logs` timeout bug: not re-tested this cycle
Only used `--count 15` (succeeded, 33.6s) — did not attempt to reconfirm the ~40-run ceiling this cycle. Re-verify next cycle.

### Dedup process that keeps working
`gh api "search/issues?q=repo:github/gh-aw+is:issue+<keywords>"` (never `gh search issues`/`gh issue list -S`, both documented broken in this env). For any CLOSED match, read/grep current source before skipping — don't trust the closed label alone.

### Full findings-file cross-reference
See `extracted-tasks.md` for the 7 issues filed + not-filed rationale this cycle; `flagged_items.md` for next-cycle watch items; `trend_data.md` for the quantitative deltas; `last_analysis_timestamp.md` for full narrative detail per finding.
