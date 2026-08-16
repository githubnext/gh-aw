## Extracted code-quality tasks (2026-08-16 cycle)

1. Fix Design Decision Gate hard-fail on bare workflow_dispatch (pr_number codegen mismatch; distinct from already-fixed #52836 invocation-cap bug; prior #52987 closed without landing this fix) — filed.
2. Add missing top-level `ambient-folders`/`github-app` fields to `FrontmatterConfig` struct (pkg/workflow/frontmatter_types.go) — filed.
3. Reconcile contradictory max-turns engine-support tables in docs/.../engines.md (line 51 vs line 534) — filed.
4. Replace tabloid-style run notifications in smoke-copilot-arm.md — filed.
5. Add `api.sentrux.dev` to daily-sentrux-report.md network.allowed (regressed after 2 prior fixes #43655/#40546) — filed.
6. Add driver-level timeout/log signal for stuck Execute CLI steps — filed.
7. Investigate 0-turn Execute CLI crash spreading to Aider/Crush (previously Copilot-only) — filed.

Not filed:
- Anthropic WIF "undocumented" claim (#53114) — FALSE POSITIVE, verified fully documented at auth.mdx:220, not filed.
- Docs "jargon before first use" complaint — chronic, 15+ closures since Feb without durable fix, not re-filed (16th would be pure noise); flagged in report body instead.
- File-length backlog (compiler_safe_outputs_job.go etc.) — already under LintMonster's umbrella tracker #52814, which explicitly declines new issues while it's open.
- MCPFailureSummary dup (#52517), PolicyCompiler seed-rule gap, httpnoctx gap (#52627), eslint-refiner #52643/#52645 — all confirmed still open from prior cycles, not re-checked in depth this cycle, not re-filed.

## Prior cycles (condensed)

- **2026-08-14**: 7 filed (Design Decision Gate hotspot investigation [superseded — see above, real root cause was different], getParsedSchemaDoc any-type, dead SkipInstructions field, AI Moderator token usage, RunSummary/DownloadResult dup, RunsOn any→RunsOnValue, dead pr-code-quality-reviewer cache read). Plus comment on #52518.
- **2026-08-13**: 7 filed (Sentrux god_files_ceiling gap [now resolved], PolicyCompiler seed-rule gap, MCPFailureSummary dup, Test Quality Sentinel pipefail fallback, PR-review infra flakiness investigation [now resolved/#52518 closed], Matt Pocock fallback, Ponytail Reviewer criteria — this 7th one deferred to 08-14 due to a shell-quoting mishap).
- **2026-08-12**: 7 filed (coverage.findProfile path bug #52309, misdirected hostname [VERIFIED FIXED via b2ef1f3/#52377], gh-aw-detection labels, schema-consistency stale target, GitHubToken shadowing #52313, agenticworkflows logs timeout re-file, label pre-creation docs).
- **2026-08-11**: 7 filed (inverted strict: docs #52086 [VERIFIED FIXED], repository_dispatch schema enum, README Copilot-default gap, JobStep/JobStepData dup, 4 log-entry structs dup, compiler_types.go split, 49% failure-rate investigation [resolved]).
