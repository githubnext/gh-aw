## Extracted code-quality tasks (2026-08-17 12:22Z cycle)

1 filed from workflow-log evidence (zero discussions changed this cycle, third quiet cycle running):
1. Move Go toolchain provisioning (`actions/setup-go` + preflight guard) from `linter-miner.md`-only into the shared `shared/mcp/serena-go.md` import — 15 other consumers of that import, including Typist (which just exceeded its entire AI credit budget as a direct result), remain exposed to the same root cause PR #53194 already fixed for Linter Miner alone. Filed.

Comment added (not counted against quota, avoided a duplicate issue):
- #52745 (Typist AIC optimization, filed 2026-08-14) — added this cycle's credit-exhaustion recurrence (run 32025029881) and the confirmed root-cause link to the Serena Go provisioning gap above.

Not filed: the standing 6-issue unlabeled-issues backlog (unchanged for 3rd cycle) — see known_patterns.md for why this is a declared do-not-repeat pattern. All other carried-forward watch items (Design Decision Gate, Sentrux, MCP type-field, Cache Strategy Analyzer, Avenger chronic driver_exit, Copilot PR prompt guidance, audit-workflows gap) had no fresh source report this cycle to re-check against.

## Extracted code-quality tasks (2026-08-17 06:26Z cycle)

None extracted — zero discussions changed in this ~6h window. The only real signal was a workflow-log-level recurrence of the already-tracked #53180 (commented, not filed) and a resolved investigation into `agenticworkflows logs` timeout behavior (see known_patterns.md / flagged_items.md). Explicitly declined to re-file the chronic "label unlabeled issues" pattern (7+ prior closures, see known_patterns.md).

## Extracted code-quality tasks (2026-08-17 cycle, ~6h focused window)

1. Fix Cache Strategy Analyzer's cache-memory detection (checks nonexistent `.tools.cache_memory` field in `aw_info.json`; verified via `generate_aw_info.cjs` + `pkg/cli/logs_models.go`) — filed.
2. Root-cause Avenger's chronic `err-config-no-structured-logs` driver_exit (19 recurrences, 4 prior closures didn't hold) — filed, distinct from the Bun-segfault issue (#51984).
3. Investigate why `audit-workflows` silently missed its own schedule for 41 days; add a heartbeat check for schedule-triggered monitors generally — filed.
4. Add prompt-writing guidance for Copilot-assigned issues (6-week trend: concise/concrete prompts merge more) — filed.
5. Investigate whether real `rpc-messages.jsonl` telemetry lacks the top-level `type` field our parser expects (0/202 sampled) — filed as an explicit investigation, not a confirmed bug.

Comments added (not counted against the 7-issue quota, used instead of filing duplicates):
- #51984 (Avenger Bun segfault) — cross-linked new recurrence #53238 (3rd distinct crash address).
- #53180 (0-turn Execute CLI crash spreading) — added this cycle's rotated-workflow evidence (recurrence #25, 5 new workflow names).

Not filed:
- smoke-copilot Google-domain firewall blocks (#53226) — verified as Chromium/Playwright's own background telemetry, not a real workflow need. Benign.
- `safeoutputs.jsonl` absent in 4 runs (observability report) — verified in source this file is only written when a safe output fires; expected absence, not a gap.
- Firewall deny-path visibility weak (18/20 runs, 0 blocked) — likely just means no blocks occurred; not filed without stronger evidence.
- Code metrics "increase inline comments fleet-wide" recommendation — real signal but too broad/unscoped (2,832 Go files) for a 1-3 day task; noted in report only.
- Docs "jargon before first use" chronic pattern — did not resurface in this cycle's 14-discussion window; nothing new, standing note carried forward only.
- Design Decision Gate pr_number bug, Sentrux `api.sentrux.dev` regression — both filed in the 2026-08-16 cycle, not re-checked this cycle since neither report appeared in this cycle's narrow 14-discussion window (no new evidence either way).

## Prior cycles (condensed)

- **2026-08-16**: 7 filed (Design Decision Gate pr_number hard-fail [distinct 2nd root cause], FrontmatterConfig ambient-folders/github-app gap, engines.md max-turns table contradiction, smoke-copilot-arm tabloid notifications, Sentrux api.sentrux.dev regression [3rd fix attempt], Execute CLI stuck-step timeout signal, 0-turn crash spreading to Aider/Crush investigation). Not filed: Anthropic WIF false-positive claim, docs jargon chronic pattern (16th dup avoided).
- **2026-08-14**: 7 filed (Design Decision Gate hotspot investigation [superseded], getParsedSchemaDoc any-type, dead SkipInstructions field, AI Moderator token usage, RunSummary/DownloadResult dup, RunsOn any→RunsOnValue, dead pr-code-quality-reviewer cache read). Plus comment on #52518.
- **2026-08-13**: 7 filed (Sentrux god_files_ceiling gap [now resolved], PolicyCompiler seed-rule gap, MCPFailureSummary dup, Test Quality Sentinel pipefail fallback, PR-review infra flakiness investigation [now resolved/#52518 closed], Matt Pocock fallback, Ponytail Reviewer criteria).
- **2026-08-12**: 7 filed (coverage.findProfile path bug #52309, misdirected hostname [VERIFIED FIXED via b2ef1f3/#52377], gh-aw-detection labels, schema-consistency stale target, GitHubToken shadowing #52313, agenticworkflows logs timeout re-file, label pre-creation docs).
- **2026-08-11**: 7 filed (inverted strict: docs #52086 [VERIFIED FIXED], repository_dispatch schema enum, README Copilot-default gap, JobStep/JobStepData dup, 4 log-entry structs dup, compiler_types.go split, 49% failure-rate investigation [resolved]).
