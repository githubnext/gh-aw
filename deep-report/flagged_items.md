## Flagged Items (2026-09-05, ~18:00Z cycle, baseline #58812, window since 12:45:58Z, 6 new discussions: 58814,58818,58823,58824,58836,58840)

- **[new, filed]** Daily Go Test Parallelizer — 75% of fleet-wide firewall blocks (235/314 `api.github.com`) this week, likely bypassing GitHub MCP proxy for direct API calls; also chronic low success rate — Daily Security Observability #58836 + Agent Performance Report #58814.
- **[new, filed]** `metrics-collector` 4+ day stale snapshot (0 executed runs in its own data) bundled with `cgo`/`content-moderation` AR-vs-executed count aggregation bug — Agent Performance Report #58814; blocks 3 downstream meta-orchestrators.
- **[new, filed]** `docs/src/content/docs/reference/editors.mdx` missing contextual intro paragraph — Delight #58824 Task 1, live-verified.
- **[new, filed]** `pkg/workflow/mcp_scripts_dependencies_validation_wasm.go` missing doc comment explaining WASM no-op rationale — Delight #58824 Task 2, live-verified.
- **[declined, chronic]** WIP-auto-label-at-creation suggestion (Daily Issues Report #58823) — already closed `not_planned` as #56107.
- **[declined, chronic, no new evidence]** `ab.chatgpt.com` firewall blocks + DIFC `unknown`-author (1,071 events) gap (#58836) — same standing watch items, no degradation evidence yet.
- **[declined, insufficient specificity]** 2 DIFC `secrecy_violation` events (#58836) — no run ID provided to investigate.
- **[declined, watch-only per source report]** `lint-monster`/`daily-firewall-report` single-sample 0/1 failures (#58814) — report itself recommends waiting for a 2nd occurrence.
- **[process note]** repo-memory `last_analysis_timestamp.md` had a one-cycle gap — briefing #58812 (12:45:58Z) was never logged by its own cycle; this cycle re-derived the correct baseline via a live discussions.json timestamp query. See [[known_patterns]].

## Flagged Items (2026-09-02, ~18:34Z cycle, baseline #57810→#57883, window since 06:46Z, 21 new discussions read in full)

- **[new, filed]** Copilot PR Conversation NLP Analysis — PR comment/review pre-fetch returned empty for all 169 PRs, total pipeline failure (not partial) — #57912.
- **[new, filed]** `LinearTargetConfig` (`pkg/workflow/linear_safe_outputs.go:16-19`) doesn't embed `SafeOutputTargetConfig`, unlike the other 14 safe-output integrations — Typist #57923, live-verified via grep.
- **[new, filed]** `docker-sbx` KVM-availability check failing during agent-prep, blocking agent execution across 3+ workflows (Daily Fact, Daily Go Test Parallelizer, Sub-Issue Closer) in one 24h window — Storify #57895.
- **[new, filed]** Code Scanning Fixer blocked by missing Sentry ingest domain (`o205451.ingest.us.sentry.io`) in network allowlist, 5 blocked requests — Storify #57895, concrete new cause for this workflow's chronic 0% pattern.
- **[new, filed]** Firewall/audit tooling: domain-parsing bug producing `(unknown)`/`and` bogus blocked-domain entries (~11% of 90 blocks) + `policy_analysis.rule_hits` empty across all 92 sampled runs despite active 11-rule policy — Daily Security Observability Report #57991.
- **[new, filed]** Delight bundle: `cache-memory` scope validator's raw `%v` error message (`pkg/workflow/cache_config.go:266`) inconsistent with sibling validators + blog post #2026-01-13 "carnival-barker" tone mismatch — #57971.
- **[declined, chronic]** `CLAUDE_CODE_OAUTH_TOKEN` silently ignored, 11th consecutive daily occurrence via #57940 — closed 4+ times previously (#54943, #54584, #46613, #39601, #48005) without a landed fix; see [[known_patterns]].
- **[declined, dup]** Squad Implement Worker 0% auto-resolve finding — overlaps already-open #57488.
- **[declined, self-filed]** UK AI Operational Resilience #57980 findings 1-2 (untracked README.md CodeQL alert, alert-dismissal hygiene gap) — already self-filed same-run as #57981/#57982.
- **[declined, out-of-scope]** UK AI Operational Resilience #57980 finding 3 (add CODEOWNERS file, 30-day SLA) — governance/policy decision, not a quick-win code fix; see [[known_patterns]].
- Weekly issues-analyst sub-agent: chronic `[WIP] * : work in progress` auto-stub pattern continuing at usual rate, 0 issues stale >7 days.
- Fleet workflow-log sampling this cycle: per-workflow-filtered queries hit `context deadline exceeded`; unfiltered 50-run sample (partial, ~1h of intended 7-day window) showed no new systemic regression beyond already-tracked chronic clusters (PR-review bot cluster #57438, AI Moderator #57437).

## Flagged Items (2026-09-02, ~06:46Z cycle, baseline #57810, window since 01:15:13Z, 9 new discussions: 57824,57827,57842,57845,57850,57853,57862,57866,57870)

- **[new, filed]** `approve_workflow_run`/`push_to_pull_request_branch` protected-files policy decline surfaced as hard failure instead of soft skip — Safe Output Health Monitor #57850, now the dominant safe_outputs failure mode (7 failed items in one day, 66.7% of today's 3 failures, 11 occurrences over 7+ days). Re-files expired #56576 (proposed 2026-08-28, auto-closed NOT_PLANNED unfixed).
- **[new, filed]** `secret-masking` still missing from `docs/src/content/docs/reference/frontmatter.md` — Schema Consistency Checker #57866 finding #3, live-reconfirmed. Re-files expired #54458.
- **[new, filed]** Quick Start docs use "stable-engine path" before "engine" concept is introduced, and "frontmatter" term isn't linked back to its definition from nav/CLI docs — Documentation Noob Test Report #57842, distinct from the standing declined "5-engine wall of options" complaint (that one was about auth-tab count, this is about term-ordering/linking).
- **[new, filed]** `max-tool-denials`/`max-daily-ai-credits`/`max-turn-cache-misses` guardrail-field family has inconsistent expression-support docs/implementation — Schema Consistency Checker #57866 finding #4.
- **[declined, already open, about to re-expire]** Schema Consistency #57866 findings #1/#2 (max-daily-ai-credits default-disabled/per-user-aggregation contradiction) — already open as #57377 (itself a re-file of closed #56301), expires ~07:06Z today; this would be its 3rd unfixed expiry if it lapses again. Watch next cycle — see [[known_patterns]].
- **[declined, already self-filed, expired unfixed]** safe-output-health Cluster 2 (`create_discussion` TypeError, "Cannot read properties of undefined (reading 'createDiscussion')", Auto-Triage Issues run 33508317422) — already self-filed as #57696, auto-closed NOT_PLANNED before a fix landed. Same orphaned-WIP-lifecycle gap as noted in the 2026-09-01T01:01Z cycle's steer-issue finding.
- **[declined, chronic, diminishing value]** Daily Compiler Quality #57827 `compiler_orchestrator_engine.go` refactor suggestions (split `setupEngineAndImports`, option structs for multi-param functions) — report explicitly frames these as "recommendations, not defects"; file already has 25+ closed refactor/lint-monster/code-quality issues with no lasting size reduction, treated as chronic saturated territory, not re-filed.
- **[declined, already tracked, scope note]** Daily Firewall Report #57824 `proxy.golang.org` blocks (138x) now also name Design Decision Gate and Matt Pocock Skills Reviewer alongside CI Optimization Coach — already covered by open #57752 (CI Optimization Coach only, filed last cycle); not re-filed, but flag for a follow-up if #57752's fix doesn't also cover the other two workflows.
- **[declined, healthy/informational]** Issue Arborist #57862 (housekeeping, chronic-failure clusters correctly left unlinked pending a real tracker), Firewall Escape #57853 (SECURE, 14/14 novel techniques failed), Sergo #57845 (self-filed `aw_sg61a1`), ESLint Refiner #57870 (self-filed 2 issues, also flagged its own repo-memory branch went stale ~2 months without anyone noticing — worth a spot-check next time that workflow's memory is reviewed) — no DeepReport action.
- **[declined, healthy]** Fleet spot-check (40-run/~4.1h sample, 2026-09-02T01:00-05:17Z): 90% raw success (36/40); all non-success entries are the already-tracked chronic PR-review 6-bot cluster (#57438) and AI Moderator (#57437), plus Avenger/CLI Version Checker (chronic, no shared root cause) and 2 already-self-filed isolated failures (Sub-Issue Closer #57877 open, Daily VulnHunter Scan). No new systemic regression.

## Flagged Items (2026-09-01, ~18:00Z cycle, baseline #57702, window since 12:47:24Z, 10 new discussions: 57703,57705,57706,57716,57719,57722,57725,57727,57736,57744)

- **[new, filed]** CI Optimization Coach missing `go` ecosystem preset in `network.allowed` (`.github/workflows/ci-coach.md`, currently `[defaults, node]`) — Daily Security Observability Report #57736 showed `proxy.golang.org:443` blocked 136x this week (61% of all firewall blocks, 100% from this workflow); live-verified frontmatter and that `pkg/workflow/domains.go` already defines a `go` preset. Recurring-pattern class, see [[known_patterns]].
- **[new, filed]** `pkg/workflow/glob_validation.go:280` error message lacks a corrective example for the `'.'`/`'..'`/`'./'`/`'../'` glob-path restriction — User Experience Analysis Report #57719, live-verified exact message text.
- **[new, filed]** `.github/workflows/test-quality-sentinel.md:210` `run-started` message uses 🔬 while the workflow's own declared `emoji`/`footer` (lines 3, 209) use 🧪 — same report #57719, live-verified all 3 lines.
- **[new, filed]** 29+ duplicated `RequireGit`/`RequireDocker`-style skip blocks across `pkg/cli/init_command_test.go` (16) and `pkg/cli/git_test.go` (13), not yet consolidated into existing `pkg/testutil` — Repository Quality Improvement Report #57706 Task 3, live-verified site counts and package existence.
- **[new, filed]** No non-blocking lint target flags unconditional `t.Skip("pending...")` (14 sites across `otel_reliability_formal_test.go`/`otel_observability_formal_test.go`) — same report #57706 Task 4, live-verified skip count and `Makefile:1041-1042` `lint-error-messages-report` precedent.
- **[declined, chronic]** `CLAUDE_CODE_OAUTH_TOKEN` silent-ignore when unset (docs/config review #57703) — closed 4-6x previously as standing policy, not re-filed.
- **[declined, chronic]** `ab.chatgpt.com:443` firewall blocks (Daily Security Observability #57736) — 2nd+ occurrence, previously root-caused as likely analytics-subdomain noise from engine-auto-injected default domains, not a missing-preset bug; not a code fix, standing decline. Distinct mechanism from the `go`-preset finding above — see [[known_patterns]].
- **[declined, chronic]** Unlabeled-issue auto-labeling at creation time / WIP-label attribution gap (Daily Issues report #57716) — no single attributable source location found after live grep, standing chronic decline (recurs most cycles).
- **[declined, already self-filed]** CodeQL alert #653 stale-dismissal flag (UK AI Resilience Assessment #57727) — that report already filed its own issue (#57728) for this same finding; not duplicated.
- **[declined, same family as chronic]** `gateway.jsonl` / DIFC `author_login`-unknown telemetry gap (Daily Security Observability #57736) — same recurring gateway.jsonl gap family previously declined multiple cycles pending verified-merged evidence; standing policy.
- **[declined, live-verification contradicted report]** Agent Performance Report #57705's claim of "6 open Dependabot-style PRs ready to batch-merge" — live `gh pr list --state open` showed zero open Dependabot PRs at time of check; stale snapshot in the source report, correctly not filed. Reconfirms the live-verify-before-filing-dependency-claims lesson (see [[known_patterns]]).
- **[declined, judgment call left to source workflow]** Repository Quality Improvement Report #57706 Tasks 1-2 (triaging/resolving the 14 otel `t.Skip`s themselves, choosing Go-vs-JS reimplementation) — too judgment-heavy for a scoped 1-3 day agentic task; the non-blocking lint-target issue (filed above) surfaces the same problem without pre-empting the implementation decision.
- **[declined, chronic]** quick-start.mdx 5-engine Copilot-framing rebalance complaint — previously NOT_PLANNED twice (#46478, #53927); recurs periodically, standing decline.

## Flagged Items (2026-09-01, ~06:40Z cycle, baseline #57574, window since 01:21:07Z, 8 new discussions: 57588,57590,57605,57607,57612,57617,57625,57626)

- **[new, filed]** `GitHubActionsPermissionsConfig` (pkg/workflow/frontmatter_types.go:54-71) missing 5 typed fields present in schema/docs (`attestations`, `copilot-requests`, `models`, `metadata`, `secret-scanning-alerts`) — Schema Consistency Check #57626, verified live via grep; distinct from already-closed docs-only #54753 (which only fixed permissions.md wording, not the Go struct).
- **[new, comment]** Added quick-start.mdx:101 grammar typo ("run one of this command" → "run this command") as a comment on already-open #57375 (3-item quick-start friction tracker filed last cycle) rather than opening a new issue — verified live, distinct line from #57375's 3 listed items.
- **[declined, chronic]** compiler.go bare `return err` / missing `%w` wrapping (Daily Compiler Quality #57590, 6 new line numbers cited) — 6+ prior issues on this exact file/theme (#45673, #45806, #47440, #49094, #50516, #53928), all closed, keeps recurring; not re-filed again, standing chronic pattern.
- **[declined, chronic]** Quick Start 5-engine auth-tab "wall of options" complaint (Documentation Noob Tester #57605) — same complaint already explicitly declined as NOT_PLANNED at #53927 (2026-08-21); recurs nearly every docs-noob-tester run, standing policy not to re-file.
- **[declined, already investigated, recurring]** Firewall Escape basic-test anomaly (api.github.com/github.com blocked + DNS SERVFAIL alongside deliberately-blocked test domain) — 3rd occurrence (prior: runs 33150215669, 33234472980); already filed and closed COMPLETED as #56577 on 2026-08-28, but recurring again suggests that closure didn't actually root-cause/fix it. Not re-filed this cycle (no new root-cause info beyond "still recurs"); flagged for a future cycle to consider reopening #56577 if a 4th occurrence appears. See [[known_patterns]].
- **[declined, self-filed]** LintMonster consolidated function-length issue (#57588), Sergo's `generatedyamlheredoc` quote bug (#57606, from #57607), ESLint Refiner's `isChildProcessObjectBinding` reassignment gap (#57625) — all self-filed by their source workflow same run.
- **[declined, healthy/informational]** Issue Arborist housekeeping (#57617, Morrow Mosaic sub-issue links, no new parents), Firewall Escape Test (#57612, SECURE, 8/8 novel techniques failed) — no action beyond the anomaly note above.
- **[declined, chronic, confirmed reverted]** Weekly issues data (500 issues, 158 open/342 closed): 91 unlabeled issues (chronic `[WIP] * : work in progress` auto-stub pattern) — confirms last cycle's "3 unlabeled, sharp positive break" was a sampling artifact, not a genuine fix; reverted to the normal 90-140 range. 0 issues open >7 days — healthy.
- **[declined, healthy]** Fleet spot-check (25-run/~3.3h sample, 2026-09-01T01:12-04:33Z): 22/25 success (88%), 3 failures (Daily Sub-Agent Model Resolution Audit, Daily AgentRx Trace Optimizer, CLI Version Checker) — latter two already logged as isolated non-systemic flakiness in prior cycles; no shared root cause, no new action.

## Flagged Items (2026-08-31, ~01:06Z cycle, baseline #57214, window since 18:35:32Z, 9 new discussions: 57217,57223,57235,57236,57241,57260,57261,57293,57299)

- **[new, filed]** Re-enable `gh-aw-detection` on `smoke-codex.md:98`, `smoke-copilot.md:168`, `daily-github-docs-seo-optimizer.md:37` — Detection Analysis Report #57261, first-time flag (3 workflows disable detection despite 8-21 runs/week); verified live at each frontmatter line.
- **[declined, self-filed]** Daily Cache Strategy Analyzer (#57223) opened 5 own issues (#57218-#57222) for cache-miss patterns in GitHub MCP Tools Report, Daily Firewall Logs Reporter, Daily Issues Report, Smoke Copilot, Agent Job Health Monitor — no DeepReport action.
- **[declined, likely non-bug, monitor]** Lockfile Statistics (#57236) engine-mix reshuffle (copilot 151→120, codex 46→75, pi 22→29 in 1 day) — no bulk-migration PR found; likely the existing experiment/engine-selection system (ADR-29985 precedent), not a misconfig. Monitor only.
- **[declined, too diffuse]** Copilot PR Prompt Analysis (#57241) CVE-remediation 65% vs 84% baseline merge rate — traced toward daily-squid-image-scan.md/#52657 but that workflow explicitly doesn't create per-image issues or assign to Copilot; no single attributable fix file found. See [[known_patterns]].
- **[declined, chronic]** Daily Observability Report (#57260) — gateway.jsonl still absent in all sampled runs (5+ prior closed-without-fix attempts), standing policy not to re-file without verified-merged evidence.
- **[declined, healthy/informational]** Daily Code Metrics (#57217, first baseline 77.7/100), Daily Team Evolution (#57235, narrative only), Smoke Copilot/ARM64 (#57293/#57299, routine passes, chronic Google-domain blocks already declined) — no action.
- **[declined, isolated, not a fleet regression]** Fleet health spot-check (20-run/~2.1h sample): 4 failures, all single-run "baseline" smoke variants (Cursor, Codex/Service-Ports, Copilot-CLI/Cross-Repo-PR, Gemini) with no shared root cause across different engines.
- **[declined, chronic, healthy]** Weekly issues data (500 issues, 139 open/361 closed): 0 open >7 days; unlabeled issues are the standing `[WIP] ... work in progress` auto-stub pattern (Daily Trajectory Grader Implementer, Daily Caveman Optimizer, etc.), not a fresh gap.

## Flagged Items (2026-08-30, ~11:40Z cycle, baseline #57099, window since 06:49:52Z, 5 new discussions: 57104,57112,57117,57129,57139)

- **[new, filed]** Re-implement `.github/skills/task-preflight` skill for container-vuln/CVE tasks — Prompt Clustering Analysis #57129, Cluster 6 doubled 60→120 PRs (2026-08-29→08-30) at flat ~66% merge, 78% of unmerged PRs near-zero-diff. Verified PR #53674 (the claimed prior fix) was closed UNMERGED and the skill file is 404 in the repo — the source discussion's "already merged" claim was factually wrong. Traced full supersession chain #53448→#55464/#55465, neither of which covers this problem. See [[known_patterns]].
- **[declined, chronic]** Copilot Session Insights (#57104) conversation-transcript fetch gap, 46+ consecutive empty-log days — already tracked open #56493, standing policy, not re-filed. One-off `CJS` failure on trajectory-grader branch noted, single occurrence, not independently actionable.
- **[declined, already tracked]** Daily Storify (#57112) — Avenger repeat engine-exit failures (chronic series #56694/#56728/#56737/#56361), Code Scanning Fixer tool-denial pattern (already self-filed #56857/#56798), Windows Runner recurrence (already tracked #56848) — no new action.
- **[declined, standing policy]** arXiv Research (#57117) — 7 architectural/security feature proposals (instruction privilege tagging, IFC for safe-outputs, persona-execution separation, skill compilation from traces, etc.), all multi-day+ scope — excluded per standing feature-request/architectural-decision policy, consistent with every prior arXiv cycle (#54196/#54469/#54767/#55048/#56867).
- **[declined, non-actionable]** Constraint Solving POTD (#57139) — puzzle content.
- **[declined, standing/chronic]** Weekly issues data (500 issues, 149 open/351 closed): 94 unlabeled issues are entirely auto-generated `[WIP] * : work in progress` stubs from known daily agent workflows (Daily Trajectory Grader Implementer, Package Specification Enforcer/Extractor, etc.) — same chronic auto-stub pattern declined in every prior cycle, not a fresh gap. 0 issues open >7 days (149/149 created within last 2 days) — healthy churn, no backlog-age concern this cycle.

## Flagged Items (2026-08-30, ~06:37Z cycle, baseline #56945, window since 18:32:48Z, 15 new discussions: 56947,56950,56958,56959,56963,56964,56997,57002,57031,57036,57053,57070,57077,57086,57090)

- **[new, filed]** `organization-projects`/`organization-custom-org-roles` permission schema wrongly allows `write` (App-only scopes must be read-only per docs and per `secret-scanning-alerts`'s existing enum) — Schema Consistency Check #57077, verified live including the runtime App-token-minting forwarding path. Root cause: fresh regression from #56851/#56982 (schema-completeness fix used the wrong enum).
- **[new, filed]** Generated `github-app` schema examples still feature deprecated `app-id` instead of preferred `client-id` — Schema Consistency Check #57077, verified live.
- **[declined, false-positive avoided]** ChatGPT-domain firewall blocks (168 total, #57031) — hypothesized missing `codex` network preset across 23 codex-engine workflows; verified live via `pkg/workflow/domains.go` that codex engine defaults are auto-injected regardless of `network.allowed` contents, so no preset gap exists. Not filed. See [[known_patterns]].
- **[declined, already resolved]** "Cannot find module 'undici'" 9-issue failure cluster (#57021-#57065, flagged unparented by Issue Arborist #57070) — verified root-cause PR #57057 merged 05:21:28Z, all 9 issues predate the merge. No action; expect self-clear.
- **[declined, chronic]** `gateway.jsonl` MCP telemetry format absent in all 20 sampled runs (Observability Coverage #57002) — 5+ prior closed-without-fix attempts per #51807 ("closed 25+ times, all still broken"); standing policy requires verified-merged evidence before any future closure, not re-filed.
- **[declined, healthy/informational]** Daily Code Metrics baseline (#56947, 83.8/100 quality, first run), Cache Strategy (#56950, no new threshold breaches), Team Evolution (#56958), Lockfile Statistics (#56959, 297 workflows stable), Copilot PR Prompt Analysis (#56963, conciseness/WIP-framing trends), Daily Performance Summary (#56964, informational backlog watch), Detection Analysis (#56997, 0 misconfigured/319 runs), "copilot was here" smoke test (#57036, chronic Google-domain blocks already declined), Artifacts Usage Report (#57090, CI/CGO sizes modest, prior retention fixes #32389/#32451 already closed) — no action.
- **[declined, self-filed]** Sergo's `manualpathconcat` 2-op coverage gap (#57053) and ESLint Refiner's 2 rule-quality findings (#57086) — both self-filed by their source workflow same run.

## Flagged Items (2026-08-28, ~08:xx cycle, true baseline #56580 (memory had gone stale to #56555 again — 2nd+ occurrence of the write-race, see [[known_patterns]]), window #56581-56696, 16 new discussions)

- **[new, filed]** Re-embed `SafeOutputTargetConfig` in remaining 11 safe-output configs — verified live still unfixed after 2 prior "closed" issues; closure was not backed by a merged fix either time. Root cause of re-filing: dedup gate must check linked commit/PR on closed issues, not just state=closed.
- **[new, filed]** Fix `graderManifestEntry` write/read schema drift (compiler writes 8 fields the CLI-side reader silently drops) — Typist #56632 Cluster 8, verified live at compiler_yaml_graders.go:67 / audit_report_graders.go:67.
- **[new, filed]** Consolidate `AuditData`/`RunAnalysis` + fix `FirewallTokenUsage`/`TokenUsage` naming drift — Typist #56632 Cluster 2, verified live at audit_report.go:52 / logs_models.go:263.
- **[new, filed]** Remove duplicate `WorkflowRunInfo` + embed `ToolUsageStatsBase` in `ToolUsageInfo` — Typist #56632 Clusters 3+7, bundled as one issue (both small, same package).
- **[new, filed]** Extract `MemoryEntryBase` for `CacheMemoryEntry`/`RepoMemoryEntry`/`DriveMemoryEntry` — Typist #56632 Cluster 4.
- **[new, filed]** Add `SuggestedFixes` to 4 simplest linters lacking auto-fix (`sortslice`, `trimleftright`, `stringbytesroundtrip`, `regexpcompileinfunction`) — discussion #56657, verified live (no matches via grep for all 4).
- **[new, filed]** Tone down cloclo.md `run-failure` "Intermission..." wording — discussion #56672, obscures actual failures.
- **[declined, chronic]** GitHubMCPDockerOptions/RemoteOptions consolidation — re-filed once already with "didn't stick" framing, reclosed; not re-filed again this cycle, monitor only.
- **[declined, stale claim corrected]** `CLAUDE_CODE_OAUTH_TOKEN` "critical blocker" framing (discussion #56650) — verified live that this is already thoroughly documented as unsupported/silently-ignored across 4 doc files (cli.md:284, claude.md:22, faq.md:494-496, auth.mdx:218-220,428). Source report's framing is stale; not filed.
- **[declined, chronic]** `gh aw init --engine claude` onboarding parity — previously declined multiple times as "too large"; not re-filed.
- **[declined, already tracked]** Metrics Collector memory cap (#56537, already open), Avenger `driver_exit` (chronic, #56361), Copilot Agent PR Analysis lock-file failure (self-filed as #56698 same run), CodeQL alerts #664/#665/#656 (already self-filed as #56680/#56681).
- **[declined, deprioritized]** Typist Clusters 5 and 6 — lower impact/space this cycle, candidates for a future cycle if still unaddressed.
- **[declined, healthy/informational]** Remaining discussions in the window were narrative/status-only with no actionable gap found.

## Flagged Items (2026-08-27, ~09:23Z cycle, window since #56215's own 02:10:40Z creation — memory timestamp was stale, see [[known_patterns]] process note; 9 new discussions: 56231,56234,56249,56254,56277,56285,56290,56291,56292)

- **[new, filed]** Fix `max-daily-ai-credits` default-behavior contradiction across docs/schema/compiler (disabled-by-default vs 5000-AIC-default vs "by the triggering user" stale scoping) — Schema Consistency Checker #56291 findings 1-2, verified live.
- **[new, filed]** Fix `user-rate-limit` legacy-alias schema gap (`max`/`max-runs` accepted by parser, rejected by schema) + `events` fallback semantics drift — Schema Consistency Checker #56291 findings 3-4, verified live.
- **[declined, self-filed]** LintMonster 650-finding largefunc backlog (#56231) — self-filed 2 execution-topic issues as usual.
- **[declined, self-filed]** Workflow Skill Extractor's 3 shared-component proposals (#56249) — all self-filed same run (cross-ref #56245).
- **[declined, self-filed]** ESLint Refiner's 2 rule-quality findings (#56290) — self-filed as usual.
- **[declined, self-filed]** Sergo's `notYetEnforced` stale-reason-string finding (#56254, aw_sg62a1) — self-filed.
- **[declined, chronic]** `CompileWorkflowData` 179-line split (#56234 Daily Compiler Quality) — 3+ prior closed attempts (#50814/#49094/#46178) never stuck, now subsumed by LintMonster's own open trackers #56228/#56229.
- **[declined, chronic]** GitHub Remote MCP Auth Test toolset unavailability (#56292) — 19th+ occurrence, standing policy.
- **[declined, healthy]** Firewall Escape Test (#56277, SECURE, 10/10 novel techniques failed), Issue Arborist (#56285, 2 new parent umbrellas created, healthy housekeeping) — no action.

## Flagged Items (2026-08-26, ~18:56Z cycle, window since 12:37Z baseline #56035, 9 new discussions: 56036,56041,56045,56065,56070,56073,56074,56095,56096)

- **[new, filed]** CI ratchet for full-repo error-message audit (2,631 violations, currently unenforced) — Repository Quality #56045 Task 1.
- **[new, filed]** Migrate 5 high-traffic `pkg/workflow/*_validation.go` files to `NewValidationError` — Repository Quality #56045 Task 2.
- **[new, filed]** Sweep top 5 `pkg/cli` files by error-message violation count (~236 violations, 9% of total) — Repository Quality #56045 Task 3.
- **[new, filed]** Remove duplicate H2 heading in Agent of the Day blog post — Delight #56070, exact before/after supplied.
- **[new, filed]** Auto-label WIP/tracker issues at creation (54/1000 unlabeled) — Daily Issues Report #56065.
- **[new, filed]** Make `gh aw init --engine claude/codex` equally prominent in quick-start.mdx — Claude Code User Docs Review #56036.
- **[new, filed]** Document Pi model-prefix fallback + Codex web-search enabled-state — Claude Code User Docs Review #56036 Priority 3.
- **[escalated, root-caused]** Avenger `ERR_CONFIG` startup failures — #55860 updated this window with root cause: sandbox refuses to bind-mount `/usr/bin/go` when it's a symlink. Agent Performance Report #56041's "67% failure, new regression" flag is this same issue, already tracked; not re-filed.
- **[declined, chronic]** `CLAUDE_CODE_OAUTH_TOKEN` doc gap (#56036), 5th consecutive daily flag — 15+ prior closed attempts per 2026-08-25 log below; not re-filed.
- **[declined, already tracked]** #56000 E007 mention-limit false-positive, corroborated by Daily Issues #56065 — already open with suggested fixes; cited not duplicated.
- **[declined, already tracked]** AI Moderator Codex 404/engine-switch ask (#56041) — covered by #56092 + prior P0s #54242/#55412/#55682.
- **[declined, already tracked]** UK AI Governance remediation queue (#56074) — redirect-check/#54037, GraphQL sprintf/#52749, untrusted write/#53737, prompt-injection sanitization/#28775+#19967+#5437, circuit breaker/#28776 all pre-existing; container-image backlog SLA too process-level to scope, monitor only.
- **[declined, healthy]** Daily Secrets Analysis (#56095, 100% redaction coverage), Copilot PR Merged Report (#56073, 38 PRs), Smoke Copilot (#56096) — no action.

## Flagged Items (2026-08-25, ~12:23Z cycle, window since 06:25Z baseline #55692, 11 new discussions: 55689,55712,55715,55717,55721,55722,55734,55749,55752,55753,55764)

- **[new, filed]** Rename `pkg/cli.Finding` → `AuditFinding` (collides with `pkg/scanfindings.Finding`) — Typist #55753, verified live.
- **[new, filed]** Add `ArtifactName`/`Filename`/`FilePath` semantic types to `pkg/constants`, retype ~90 untyped constants — Typist #55753, extends existing convention.
- **[new, filed]** Type `GitHubReposScope` (currently bare `any`) with custom YAML unmarshaling — Typist #55753, verified live at tools_types.go:306.
- **[new, filed]** Consolidate `BoundedQueriesConfig`/`AWFBoundedQueriesConfig` duplicate structs — Typist #55753 Cluster 4, verified live, code comments already flag manual-sync burden.
- **[new, filed]** Investigate Squad Implement Worker's 0/4 `action_required` stall — Copilot Session Insights #55712.
- **[declined, chronic]** Copilot Session Insights conversation-transcript fetch gap (#55712), 47th+ consecutive empty-log day — standing policy, not re-filed.
- **[declined, inconclusive/monitor]** Prompt Clustering container/MCP-CVE cluster 63.4% merge rate (#55734) — matching fix #53687 closed completed 2026-08-18, but 18-day sample window straddles that date; can't tell if the fix is holding. Monitoring.
- **[declined, accepted design tradeoff]** Daily Experiment Report's recurring >2-variant decision-engine ask (#55715) — confirmed deliberate per ADR-54978 (Accepted, 2026-08-23): "Automatic adjudication is limited to exactly two variants." Not a bug.
- **[declined, needs design call]** Design Decision Gate PR #55641 allowed-files mismatch (#55717 Daily Storify) — gate is scoped to `docs/adr/**` only; widening allowlist vs. tightening the agent's own write scope is a judgment call, not filed this cycle.
- **[declined, healthy]** Terminal Stylist (#55722, console system already strong), Daily News (#55721), Auto-Triage (#55689, 100% success), API Consumption (#55749, partial-data run), MCP Structural Analysis (#55764, day-1 baseline) — no action.

## Flagged Items (2026-08-25, ~00:34Z cycle, window since 18:25Z baseline #55473, 8 new discussions: 55467,55477,55503,55505,55514,55519,55526,55538)

- **[new, filed, HIGH]** Codex `driver_exit`/context-rebuild-runaway collapse, 49.4% success (41/83 runs) — #55526 Finding 1; confirmed #54393 (closed) covers an unrelated root cause, so this was genuinely untracked.
- **[new, filed]** Daily Documentation Diagram npm firewall block (2.1M requests) + retry storm — #55526 Finding 2; verified live, only `defaults`/`github` in `network.allowed`.
- **[new, filed]** Daily Code Metrics implausible LOC/test-ratio swing (-29% LOC, +75% ratio in 1 day) — #55519 Critical Issue 1.
- **[new, filed]** Daily Performance Summary 90-day window instability (45-69% swings) — #55519 Critical Issue 2.
- **[new, filed]** Daily Team Evolution title-date bug (uses run date, not window_start) — #55519 Warning 1; verified live at workflow lines 65-68/125.
- **[new, filed]** github-discussion-query mcp-script "Argument list too long" above ~12 items — #55519's own tooling note, caps Regulatory Report's review sample.
- **[declined, already tracked]** Code Scanning Fixer chronic 0-tok failures — #55526 Finding 3, already open #55498.
- **[declined, already tracked, chronic]** Ponytail Reviewer 58.3% success, 182K avg tokens — #55538, already open #55397/#54402; 5+ prior closed attempts never stuck.
- **[declined, self-corrected]** Lockfile Statistics discussion_category regex — #55505 fixed its own extraction methodology within the same run, no code gap remains.
- **[declined, environmental/monitor]** audit-workflows repo-memory mount reported read-only this run (#55526) — source workflow explicitly called this a sandbox quirk not to work around; single occurrence, watch for recurrence.
- **[declined, healthy]** Detection Analysis Report (#55538, 0 misconfigured workflows) — no action.

## Flagged Items (2026-08-24, ~18:25Z cycle, window since 06:29:00Z baseline #55312, 20 new discussions: 55323,55334,55337,55339,55340,55343,55354,55364,55368,55369,55381,55391,55401,55405,55409,55423,55432,55448,55453,55460)

- **[escalated, comment]** Cross-engine agent-CLI segfault (P0 #54186) — added fresh evidence from Weekly Workflow Analysis #55354: per-window failure-rate table (0%→100% between 03:26-08:59 UTC), newly-affected workflows (AI Moderator 5/5 codex, Daily Go Test Parallelizer 3/3 copilot, Smoke Claude 2/2 claude). Same incident as already-open #54186, not a new issue.
- **[new, filed]** Typist #55369: dedupe agentUsageEntry/TokenCoreMetrics (pkg/cli/token_usage_types.go) + retype DisapprovalIntegrity/EndorserMinIntegrity string fields to GitHubIntegrityLevel enum (pkg/workflow/tools_types.go) — both verified live. A third Typist claim (narrow checkStepGHToken's parameter to map[string]any) was checked and found INVALID — verified live that its caller passes `any` from a `[]any` list (step_shell_validator.go:75,84-85), not a guaranteed map — excluded from the filed issue.
- **[new, filed]** Repository Quality Improvement Report #55409: split pkg/workflow/safe_outputs_handler_registry.go (1,091 lines, largest non-test file in repo, 49-entry closure map).
- **[new, filed]** same report: split pkg/workflow/awf_config.go (1,090 lines, mixes schema/build/policy in one file).
- **[declined, already tracked]** ai-moderator engine-switch (#54941), cgo 29.3% success (#54940), q workflow 0.8% success — all already open/tracked chronic issues.
- **[declined, chronic, no re-file]** "instrument Copilot CLI stderr", CLAUDE_CODE_OAUTH_TOKEN docs warning — 4-6+ prior closed attempts each.
- **[declined, verified not a gh-aw fix]** list_label MCP pagination (mcp-analysis #55381, self-flagged as possibly regressed) — closed 3x already (#50689, #48942, #33819); actual implementation lives in external github-mcp-server, not this repo.
- **[declined, already tracked]** UK AI Resilience "critical untrusted checkout in q.lock.yml" (#55432) — already filed as open #55433.
- **[noted, not re-filed]** #55328 (threat-detection issues:write regression, v0.87.4, defeats ADR-54630) — genuine unlabeled open issue, already exists (not discussion-sourced this cycle); flagged for triage/labeling only.
- **[declined, healthy/informational]** Org health 64/69 unassigned issues (#55339, partial ~14-repo sample), api-consumption null core_consumed outage (#55364), copilot-session-insights 46-day empty-transcript gap (#55323, chronic), agent-performance AI Moderator/PR Description Updater recommendations (#55405, chronic 3-6x closed/reopened).

## Flagged Items (2026-08-24, ~06:29Z cycle, window since 00:34:57Z baseline #55204, 11 new discussions: 55233,55240,55244,55245,55268,55270,55280,55282,55292,55296,55299)

- **[new, filed]** CI enforcement gap for 4 confirmed-clean linters (`generatedyamlheredoc`, `manualpathconcat`, `packagelevelmutableslicemap`, `walkfuncerrshadow`) not in `cgo.yml`'s `LINTER_FLAGS` — Sergo R61 (#55268).
- **[declined, chronic, no re-file]** GitHub Remote MCP Auth Test toolset unavailability (#55296) — 18th+ occurrence, see prior entries below.
- **[declined, already tracked]** Code Scanning Fixer 100% failure, 2/2 runs (#55233, #55270) — #54544/#54237/#54063.
- **[declined, already tracked]** Design Decision Gate 25% failure, standing issue confirmed independent of infra events (#55233, #55270) — #54238.
- **[declined, informational, re-affirmed]** Sentry telemetry allowlist gap, >50% of blocked traffic fleet-wide (#55240) — blocking 3rd-party telemetry is arguably correct posture, not a bug; consistent with prior #54693 read.
- **[declined, self-consolidated]** LintMonster 734-finding function-length backlog (#55244) + Compiler Quality's `CompileWorkflowData` 180-line flag (#55245) — same tracker.
- **[declined, too early, n=1]** Safe-output-health's 2 new single-occurrence failure signatures (#55280): GitHub App token gen failure (Daily Harness Experiment Proposer), Process Safe Outputs failure (Designer Drift Audit) — report itself recommends monitoring, not filing yet.
- **[declined, duplicate]** Safe-output-health's raw step-log-capture recommendation (#55280, 3rd consecutive audit to raise it) — duplicate of open #54756.
- **[declined, healthy/no action]** Firewall Escape Test SECURE (#55282), Issue Arborist housekeeping (#55292), Sergo/ESLint Refiner self-filed findings (#55268, #55299).
- **[noted, not actioned]** Single spam-like unlabeled issue #55307 ("asap") from a real non-bot user — too trivial to file a code-quality task around; left for normal issue triage.

## Flagged Items (2026-08-23, ~18:28Z cycle, window since 12:32:09Z baseline #55074, 8 new discussions: 55075,55076,55078,55100,55104,55114,55117,55123)

- **[new, filed]** model-alias fallback/failure behavior undocumented in model-tables.md — Claude docs review (#55075); verified live, page has no such section.
- **[new, filed]** smoke-agent-public-none.md run-failure message doesn't name the tested guard policy — Delight (#55104); verified live at lines 45/47.
- **[new, filed, corrected]** github-mcp-tools-report.md's own instructions (lines 392/442/522) point "Default toolsets" doc updates at `.github/aw/github-agentic-workflows.md`, which exists but lacks that content — real content is at `.github/aw/syntax-tools-imports.md:90`. Source report (#55076) claimed the target file "doesn't exist" — verified false via `ls`; refiled with the accurate root cause instead of repeating the false claim.
- **[new, filed]** Delight's CLI-quality section silently skipped this run — `storage.googleapis.com` not in `network.allowed`, corroborated independently by #55117's firewall-block log (2x blocks attributed to Delight) and #55104's own self-report of a skipped section.
- **[new, filed]** engine-example-counter (claude-code-user-docs-review.md) undercounts nested `engine: {id: ...}` form — root-caused #55075's own self-flagged "sharp shift... worth confirming" trend note; verified live via grep (37/34 literal vs 24/101 additional nested-form files).
- **[new, filed]** Daily Secrets Analysis (#55123, first-ever run) recommends replacing its own ad hoc line-proximity secrets-in-output grep (self-flagged as producing false positives) with a deterministic CI check.
- **[declined, chronic, no re-file]** CLAUDE_CODE_OAUTH_TOKEN quick-start warning (#55075) — closed 4x already (#46613, #54584, #54590, #54951).
- **[declined, chronic, no re-file]** `q` workflow 0.8% success re-diagnosis (#55078) — 2 prior re-diagnose issues closed without fixing it since PR #43527 merged 7 weeks ago.
- **[declined, already tracked]** AI Moderator 3.6% success (#55078, already #54941); CGO 23.1% success (#55078, already #54940, non-agentic CI workflow).
- **[declined, not a code gap]** shared-alerts.md/agent-performance-latest.md stale "deprecation candidate"/"100% AR" entries for 5 recovered agents (#55078) — runtime cache/repo-memory state, not a git-tracked file we can fix.
- **[declined, chronic, no re-file]** "Smoke Copilot" Google-domain firewall blocks (#55117, 38/41) — same symptom class as "Smoke Claude", closed 2x already (#54975, #54944); flagged instead as a candidate for a shared browser-automation allowlist fix across the smoke-test family, see [[known_patterns]].
- **[declined, too diffuse]** "(unknown)" blocked domain across 8 workflows (#55117) — no common root cause identifiable without raw per-run logs.
- **[declined, informational]** Daily Issues Report (#55100): 661/1000 unassigned issues, ~4 test-artifact-titled issues, 328-issue failure/agent/workflow cluster — triage-volume observations, no single specific fix.
- **[declined, too large]** Claude docs review Priority 2/3 (#55075): `gh aw init --engine claude` onboarding parity, inline WIF setup, growing example library toward 138-vs-60 gap — real but not quick-win scoped this cycle.
- **[declined, healthy/no action]** Issue Arborist (#55114, healthy housekeeping), Security Observability (#55117, 0.9% block rate, 0 DIFC events).

## Flagged Items (2026-08-23, ~12:30Z cycle, window since 06:23:00Z baseline #55027, 8 new discussions: 55020,55037,55046,55048,55050,55056,55060,55062)

- **[new, filed]** bare `fmt.Print` instead of `fmt.Fprint(os.Stdout, ...)` in `pkg/cli/status_command.go:295` and `pkg/cli/view_command.go:168` — Terminal Stylist (#55050); verified live, no dup.
- **[new, filed]** 0/44 workflows with active experiments declare a tracking `issue:` field — Daily Experiment Report (#55046) recommendation #3, filed for the 4 highest-value near-ready experiments (daily-security-red-team, ci-coach, daily-safe-output-optimizer, test-quality-sentinel); `issue:` confirmed as a real ADR-29618 schema field already used elsewhere.
- **[declined, chronic, no re-file]** Copilot Session Insights conversation-transcript fetch outage (#55037), 46th+ consecutive occurrence — standing chronic policy.
- **[declined, too large/unscoped]** Per-run outcome-metric instrumentation (token_count/success_rate/guardrail_pass) for experiments so `guardrail_metrics` stop reporting `unsupported` (#55046) — report frames as "next iteration"; overlaps Draft ADR-29985's scope but that ADR doesn't cover outcome metrics specifically; flagged for a future cycle with a narrower slice.
- **[declined, overlaps existing]** Prompt Clustering Cluster 0 (#55056, 46.6% merge, unbounded `[WIP]` investigation drafts) — process/prompt-guidance fix overlapping already-open #54232 (stale backlog-task screening); Cluster 5 (62.3% merge, wide-blast-radius version bumps) — process recommendation, not filed.
- **[declined, not a code gap]** "Legacy small-agent variant labels" causing balance-test noise (#55046, daily-caveman-optimizer/daily-doc-healer/daily-doc-updater) — verified current frontmatter only has 2 variants each; the noise is in historical state.json/jsonl records, not current config; not scoped enough to file.
- **[declined, insufficient baseline]** GitHub API Consumption Report (#55060), first-ever run, only ~17.2h of real data — PR Sous Chef ~16% of API calls noted but not yet actionable; revisit once real history accumulates.
- **[declined, healthy]** Auto-Triage (#55020, 100% success), arXiv digest (#55048, research directions only), Terminal Stylist's overall "mature/consistent" finding (#55050), Constraint Solving POTD (#55062) — no action.

## Flagged Items (2026-08-23, ~06:23Z cycle, window since 00:35:59Z baseline #54946 (recovered from own report body, memory timestamp was stale), 11 new discussions: 54937,54965,54970,54972,54984,54985,54989,54993,54999,55005,55007)

- **[new, filed]** proxy.golang.org/sum.golang.org blocked 259x in Daily Safe Output Integrator + Documentation Unbloat — Daily Firewall Report (#54965); same fix pattern as #54348/#54063/#48920/#48962, just not yet applied to these 2 workflows.
- **[new, filed]** registry.npmjs.org blocked 100% (3/3) in "Cache directory setup" workflow — same report; verified no existing open issue for this specific workflow.
- **[new, filed]** Quick Start Guide: "frontmatter" undefined + install-method ambiguity — Documentation Noob Tester (#54985)'s own 2 stated quick wins.
- **[new, filed]** stale inert cap_net_raw file capability on ping/mtr-packet — Firewall Escape Test (#54993, SECURE, no exploit); minor hardening cleanup.
- **[new, filed]** all 53 eslint-factory rules registered at "warn" not "error" — ESLint Refiner (#55005); proven-correct rule (require-http-response-error-listener) caught a real crash bug (#55002) that sat unfixed because nothing gates on warnings.
- **[new, filed]** LintMonster's "planned" path-join cleanup (pkg/gitutil, pkg/repoutil, 2 findings) was never actually filed — closed the planned-vs-filed gap.
- **[declined, already tracked]** PR Sous Chef Process Safe Outputs batch failure, 3rd occurrence, and audit/logs step-log gap — Safe Output Health Monitor (#54989); already open #53263/#54756.
- **[declined, chronic, no re-file]** GitHub Remote MCP Auth Test toolset unavailability (#55007) — 17th+ occurrence, durable-fix issue #54739 already closed without effect.
- **[declined, already self-filed]** Sergo's manualpathconcat ADD_ASSIGN gap (#54984) — #54983. ESLint Refiner's 3 individual rule-quality findings (#55005) — self-filed per report (only the systemic warn/error gap filed separately).
- **[declined, likely subsumed, unconfirmed]** Compiler Code Quality Report's extractAdditionalConfigurations 186-line function (#54972, first run) — probably covered by LintMonster's #54699 consolidated pkg/workflow function-length tracker, but not line-by-line confirmed; flagged for next-cycle spot-check if it recurs.
- **[declined, healthy]** Auto-Triage (#54937, 100% success), LintMonster's 733 already-tracked largefunc findings (#54970), Sergo's clean re-verification of #54718-sibling fixes (#54984), Docs Noob Tester's longer-term (non-quick-win) recommendations, "copilot was here" smoke test (#54999) — no action.


---
_[Entries before 2026-08-22 trimmed 2026-09-08 for repo-memory size hygiene; older history exists in prior git revisions of this file if needed.]_

## Flagged Items (2026-09-08, ~01:05Z cycle, baseline #59283, window since 18:40:46Z, 4 new discussions: 59287,59295,59299,59301)

- **[new, filed]** 6 `shared/mcp/*.md` configs (agentdb, ast-grep, azure-devops, microsoft-docs, skillz, tavily) use `allowed: ["*"]` wildcard tool exposure instead of an explicit allowlist, unlike the already-hardened `azure.md` sibling (7 explicit read-only tools, 2026-05-19 security decision) — MCP Inspector Report #59287, live-verified via grep across all 7 files. `mempalace.md` bundled in: 4 explicitly-allowed mutating tools (`mempalace_add_drawer`, `mempalace_delete_drawer`, `mempalace_kg_add`, `mempalace_kg_invalidate`) with no secret/auth gating, confirmed via grep. This extends the single-server `tavily` "stale wildcard TODO" lesson from 2026-09-01 (see [[known_patterns]]) — same finding has now recurred across 5+ MCP Inspector cycles for tavily specifically, but this is the first time the full 6-config wildcard scope + mempalace mutating-tool angle has been bundled and filed together. Dedup-checked: local `weekly-issues-data/issues.json` grep (0 matches for "wildcard"/"mempalace"/server names) + `mcp__github__search_issues` (0 unfiltered matches for "mempalace mutating tools"; "wildcard tool allowlist" query returned only 1 unrelated closed issue #25105 about gateway enforcement, plus 10 filtered by integrity policy — same chronic redaction pattern, not evidence of a duplicate).
- **[declined, chronic, no new evidence]** Copilot PR Prompt Analysis #59299's "operational value study" template (50% merge rate this run) — same maintainer research-campaign pattern (ADR-55155) already traced and downgraded to watch-only across many prior cycles (2026-09-06 ~01:08Z onward); no new evidence changes that verdict.
- **[declined, chronic, informational]** Lockfile Statistics #59295 (299 workflows, no material day-over-day change) — recommendation to check "6 workflows missing missing_data/missing_tool/noop scaffold" is a vague aggregate count with no workflow names given in the report; not independently actionable without further identification work this cycle. Daily Performance Summary #59301 (0% discussion answer rate, 329 open issues) — same chronic informational pattern as #59034/#58871/#58609 lineage, declined repeatedly.
- Very narrow window (~6.4h, 4 discussions) — lowest-yield cycle in recent history; only 1 genuinely new fileable item.
- `agenticworkflows logs` not invoked this cycle — narrow window with all 4 discussions being routine analytics/fleet-health reports already gave sufficient signal (Lockfile Stats + Daily Performance Summary both touch fleet health), consistent with prior cycles' narrow-window fallback pattern.
- Repo-memory hygiene: trimmed `flagged_items.md` (124KB→47KB, cut before 2026-08-22), `known_patterns.md` (93KB→45KB, cut before 2026-08-20), `trend_data.md` (67KB→47KB, cut before 2026-08-19) this cycle — completing the trim-oldest-entries treatment flagged as needed since the 2026-09-06 ~17:50Z cycle. `processed-discussions.md` (14KB) and `extracted-tasks.md` (39KB) left untrimmed, still well under any size concern.
- Next cycle: watch pickup on the MCP wildcard-allowlist issue. If a future MCP Inspector run still shows any of the 6 configs as wildcard, that's evidence the issue wasn't actioned (not a new finding).
