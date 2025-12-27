## Known Patterns (2025-12-27)

- Code quality modules trend positive: Go Fan reviews for golangci-lint, lipgloss, and huh all rate integration as exemplary with only optional doc/feature experiments (incremental lint docs, lipgloss list/tree, accessible mode flag). Reinforces strong dependency hygiene.
- Schema consistency remains high (95%+) per Dec 26 checker; remaining gaps are UX-level (firewall log-level accepts any string, safe-outputs max defaults not fully documented).
- Recent workflow window (last ~12h) mostly stable: 20 runs, one failure (Q), two cancels (Tidy), one missing-tool in Smoke Codex; Tidy and Great Escapi generate most warnings/errors but complete.
- Issue backlog improving: weekly cache shows 270 issues with 34 open and only 13 unlabeled; creation/closure over last 3d nearly balanced (115 created / 114 closed).

## Known Patterns (2025-12-26)

- Workflow health dipped: latest audit shows 64.7% success (44/68) with MCP init failures (invalid URL/read_buffer) and safeoutputs outages hitting smoke suites; permission-denied warnings persist across AI Moderator, Issue Monster, Tidy, Spec-Kit.
- Firewall denials trending higher (25.2% overall; 92 GitHub API/Web blocks, 90 LinkedIn) with research.md and daily-news.md still missing GitHub MCP and driving GitHub/API blocks; package registries (npmjs/pypi) blocked in smoke-codex-firewall.
- Copilot tokens stable at ~133M/day ($2.66) but concentrated: CI Cleaner 17%, Tidy 10%, Issue Monster 8.4%; efficiency flat day-over-day after prior drop.
- Issue backlog widening again: weekly slice 266 issues with 44 open (was 24), last 3d 101 created vs 90 closed; unlabeled set shrank to 13 (0 open).

## Known Patterns (2025-12-25)

- Workflow health rebounded: daily audit shows 87.93% success (+32% vs Dec 24) with error/warning volume cut by ~68%; remaining pain points are Tidy lacking build tools, Issue Monster GitHub search errors, Playwright MCP absence, and permission-denied noise.
- Firewall denials are rising again (29.7% current week) with api.github.com/github.com blocks signaling Copilot workflows still skipping GitHub MCP; LinkedIn remains top blocked (90 requests) but is expected.
- Copilot token usage is trending down sharply (-68% day-over-day) while spend remains concentrated (CI Cleaner 26% of 3-day total; Issue Monster and Tidy high-frequency drivers); efficiency gains emerging but hot workflows remain.
- Issue backlog improved: weekly slice at 240 issues with only 24 open and zero unlabeled open; last 3 days closed 113 vs 92 created, reversing prior backlog growth.

## Known Patterns (2025-12-24)

- Daily Copilot PR Merged continues to misconfigure safeinputs (missing safeinputs-gh tool, invalid URL, GITHUB_TOKEN unavailable) and surfaces the week’s only missing-tool hit.
- Firewall denials dropped to zero in the latest sample after sustained ~30% rates; likely temporary relief that needs confirmation over more runs.
- Issue creation is outpacing closure again (113 created vs 100 closed in 3 days), pushing open count to 47, though unlabeled issues dipped to 18 total (2 open).
- Artifacts Summary run on `copilot/fix-discussion-permissions-issue` logged permission-denied warnings while trying to create analysis scripts, hinting at repo/write gaps for that branch-driven workflow.

## Known Patterns (2025-12-23)

- Firewall denials trending upward (≈30% current week) with GitHub API and LinkedIn still top blocks, pointing to missing GitHub MCP config plus questionable external scraping attempts.
- Copilot token usage remains concentrated in scheduled maintenance (Hourly CI Cleaner, Issue Monster, Tidy); average cost per run stays low but high-frequency schedules drive spend.
- Workflow failures spiked from a push to `copilot/create-custom-action-setup-activation` that triggered many workflows at once (Plan Command, Release, Q, Nitpick Reviewer, DeepReport, Scout, etc.).
- Issue backlog improved (open down to 34 from 45) while creation remains high (≈105 created vs 98 closed in 3 days); unlabeled set grew slightly to 20 total but only 2 are open.
- Missing-tool surfaced again in Daily Copilot PR Merged (one run with missing tool despite success), signaling incomplete wiring.

## Known Patterns (2025-12-22)

- Issue intake spike: 85 issues created in last 3 days with 66 closures; open count rose to 45 (was 21 on 12/20), signaling backlog growth despite healthy throughput.
- Reporting coverage broadened: new automated briefs (repository tree map, dependency quality, issue-linking arborist, static analysis, prompt/MCP analyses) keep daily visibility high.
- Missing-tool cases persist but are localized: Dev workflow lacks GitHub MCP read_issue capability; Daily Copilot PR Merged relies on safeinputs-gh and failed once without it.
- Weekly Issue Summary workflow is currently red, contrasting with mostly green AI Moderator runs (several successes in latest 20 runs).
- Unlabeled issues remain a small but present backlog (19 total, 5 open), risking triage drift.

## Known Patterns (2025-12-20)

- Safe outputs config gap persists: Daily Issues Report safe_outputs failed once for missing `GH_AW_ASSETS_BRANCH`, indicating workflows still ship without required env vars.
- Copilot GitHub API blocks continue: Firewall report shows `api.github.com` remains top-denied domain (23% of denials) due to missing GitHub MCP configuration in Copilot workflows.
- AI Moderator instability: Multiple consecutive AI Moderator runs failed on issue_comment triggers (run IDs around §20401747970), suggesting a reproducible configuration or permission fault.
- Runtime-schema skew: Runtime accepts string `max-turns` while schema declares integer; schema documentation needs to align with runtime behavior.
- Beginner doc friction remains: Noob test reaffirms missing “what is this tool” framing, glossary gaps, and unclear lockfile guidance for first-time users.
- JavaScript surge: Code metrics note ~83% JS LOC growth over 30d with a short-term dip in test ratio, highlighting need to keep JS tests and docs in sync.
