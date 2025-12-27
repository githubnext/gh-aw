## Trend Data (as of 2025-12-27T02:03:56Z)

- Workflow snapshot (last 20 runs, ~12h): 1.1h duration, 6,415,700 tokens, ~$1.23 est cost, 38 turns, 79 errors, 162 warnings, 2 missing-tools. Outcomes: 8+ successes (Tidy x3, Great Escapi, Issue Monster, Smoke Claude, Plan Command, Smoke Codex), 1 failure (Q), 2 cancels (Tidy), 1 in-progress (DeepReport). Smoke Codex logged the only missing-tool; Tidy/Great Escapi own most warnings/errors.
- Token highlights: Great Escapi consumed ~2.1M tokens; Tidy runs ranged 0.2–0.6M tokens with repeated warnings/errors despite success.
- Codebase health (Dec 26 report): LOC steady at 530,568 with workflows dominating (328,678 LOC, 62%); churn very high over 7d (313 commits, -710k net lines) and 10 large files flagged; quality score 94.6/100.
- Schema consistency (Dec 26): 95%+ alignment; low-priority gaps in firewall log-level validation and documenting safe-outputs max defaults.
- Module reviews (Dec 26–27): golangci-lint, lipgloss, and huh integrations rated exemplary with optional follow-ups (document incremental linting, clarify lipgloss version pin, add huh accessible mode flag).
- Issue stats (weekly slice): 270 total; 34 open / 236 closed. Created last 3 days: 115; closed last 3 days: 114. Unlabeled issues: 13 total. Label leaders: ai-generated 154, plan 153, enhancement 92, documentation 46, automation 37.

## Trend Data (as of 2025-12-26T15:10:00Z)

- Workflow snapshot (last 20 runs, ~3d): 1.6h total duration, 10,246,346 tokens, ~$5.51 est cost, 141 turns, 366 errors, 269 warnings, 1 missing-tool; notable failures in Smoke Claude and Super Linter, missing tool surfaced in Smoke Codex.
- Audit health (Dec 26): 68 runs across 35+ workflows with 64.7% success (44/68), 1,384 errors/warnings, 4 missing tools, 5 MCP failures; MCP init invalid URL/read_buffer errors and safeoutputs failures dominate.
- Token trend: Copilot tokens stable at 133,075,886 tokens/day ($2.66) over 341 runs; spend concentrated in CI Cleaner (17.2%), Tidy (10.2%), Issue Monster (8.4%); efficiency flat day-over-day.
- Firewall trend: Daily report shows 25.2% denial rate (104 denied / 309 allowed; 10 domains) with GitHub API/Web (92 blocks) and LinkedIn (90) leading; research.md and daily-news.md lacking GitHub MCP are primary drivers; npmjs/pypi also blocked in smoke-codex-firewall.
- Issue stats (weekly slice): 266 total; 44 open / 222 closed. Created last 3 days: 101; closed last 3 days: 90. Unlabeled issues: 13 total, 0 open. Label leaders: ai-generated 128, plan 127, enhancement 76, automation 45, documentation 44.

## Trend Data (as of 2025-12-25T15:10:00Z)

- Workflow snapshot (last ~10 runs, ~7d): 1.0h total duration, 5,650,141 tokens, ~$3.76 est cost, 82 turns, 477 errors, 196 warnings, 1 missing-tool. Recent runs mostly success across Documentation Unbloat, Semantic Function Refactoring, Lockfile Stats, Issue Arborist; only missing-tool surfaced in Daily Copilot PR Merged.
- Token trend: Copilot token report shows 129,922,734 tokens over Dec 23-25 ($1,948.84, 302 runs) with 68% day-over-day drop; top spenders CI Cleaner (26% of total), Issue Monster, Tidy; average cost/run $6.45.
- Workflow health trend: Daily audit reports success rate jump to 87.93% (vs 55.8% prior day) with errors down to 323 from 1,021; persistent issues are missing build tools (Tidy), GitHub search errors (Issue Monster), Playwright MCP missing, permission-denied noise.
- Firewall trend: Daily firewall report shows denial rate back up to 29.7% this week (104 denied of 413 total), led by linkedin.com (90 blocks) and api.github.com/github.com (92 blocks) due to missing GitHub MCP in Copilot workflows.
- Issue stats (weekly slice): 240 total; 24 open / 216 closed. Created last 3 days: 92; closed last 3 days: 113. Unlabeled issues: 19 total, 0 open. Label leaders: ai-generated 106, plan 105, enhancement 62, automation 37, documentation 36.

## Trend Data (as of 2025-12-24T15:14:44Z)

- Workflow snapshot (last 5 runs, ~24h): 11.1m total duration, 982,526 tokens, ~$0.00 reported cost, 8 errors, 21 warnings, 1 missing-tool. Outcomes: Tidy (success + one cancel + one in_progress), DeepReport (in_progress), Daily Copilot PR Merged (in_progress, missing safeinputs-gh), Artifacts Summary (in_progress, permission denied warnings), Smoke Codex Firewall (success).
- Missing tools: safeinputs-gh missing/invalid URL in Daily Copilot PR Merged (run §20488854921) alongside GITHUB_TOKEN not available errors.
- Issue stats (weekly slice): 236 total; 47 open / 189 closed. Created last 3 days: 113; closed last 3 days: 100. Unlabeled issues: 18 total (2 open). Label leaders: ai-generated 104, plan 103, enhancement 58, automation 37, documentation 34, automated-analysis 29.
- Firewall snapshot (last 5 runs): 14/14 requests allowed, 0 denied; domains allowed include api.github.com and copilot endpoints—needs a larger window to confirm reversal of prior ~30% denial rate.
- Discussion cadence (Dec 24): lock file stats (#7514), static analysis (#7512), repo quality/governance (#7510), firewall (#7496), audit report (#7490), agent performance (#7494), type consistency (#7483), copilot prompt analysis (#7485), token report (#7484), MCP structural analysis (#7482), documentation consolidation (#7481), campaign portfolio (#7480), daily status (#7479), doc noob test (#7478), actionlint module review (#7472), GitHub MCP remote tools (#7473), daily issues (#7471), prompt clustering (#7468), artifacts usage (#7458), daily status (#7457).

## Trend Data (as of 2025-12-23T15:11:04Z)

- Workflow snapshot (last 50 runs, ~7d): 56.1m total duration, 7,780,920 tokens, ~$3.59 estimated cost, 482 errors, 238 warnings, 1 missing-tool. Outcomes: success 10 / failure 36 / cancelled 1 / pending 3; failures dominated by push to `copilot/create-custom-action-setup-activation` (Plan Command, Release, Q, Nitpick Reviewer, DeepReport, Scout, etc.).
- Missing tools: Daily Copilot PR Merged (run §20464285031) still reports 1 missing tool even on success.
- Issue stats (weekly slice): 235 total; 34 open / 201 closed. Created last 3 days: 105; closed last 3 days: 98. Unlabeled issues: 20 total (2 open). Label leaders: ai-generated 115, plan 114, enhancement 53, automation 30.
- Discussion cadence (Dec 23): lock file stats (#7404), issue-linking arborist (#7400), static analysis (#7399), firewall (#7390), prompt analysis (#7388), copilot token report (#7383), Go module review (#7356), type consistency (#7368), doc consolidation (#7365), MCP structural analysis (#7367), daily status (#7363), doc noob test (#7357), copilot agent analysis (#7350), agent performance report (#7348), portfolio dashboard (#7231 updated).
- Firewall trend: denial rate rising to ~29.7% this week; top blocks remain GitHub API domains and LinkedIn.
- Token trend: Copilot workflows consumed 159.6M tokens over Dec 20-23 (~$4.79) with Hourly CI Cleaner alone at 29% of cost; average cost/run $0.0119, weekday spike on Dec 22.

## Trend Data (as of 2025-12-22T15:11:25Z)

- Workflow snapshot (last ~20 runs): 10 runs, 33.6m total duration, 2,123,217 tokens, ~$2.42 estimated cost, 199 errors, 109 warnings, 2 missing-tool incidents. Outcomes: 1 failure (Weekly Issue Summary), 1 cancelled (Tidy), remainder success with multiple green AI Moderator runs.
- Missing tools: GitHub MCP read_issue missing in Dev workflow (run §20435819459); safeinputs-gh required in Daily Copilot PR Merged (run §20435787142).
- Issue stats (weekly slice): 232 total; 45 open / 187 closed. Created since Dec 19: 85; closed since Dec 19: 66. Unlabeled issues: 19 total (5 open). Label leaders: ai-generated 112, plan 112, enhancement 44, documentation 35, automation 34, automated-analysis 29.
- Discussion cadence: Dec 22 delivered repository tree map (#7273), lock file stats (#7270), dependency quality (#7251), issue-linking arborist (#7259), static analysis (#7257), firewall (#7242), prompt/MCP analyses (#7238, #7236), doc noob test (#7222), Go type consistency (#7235), plus daily status/prompt clustering/portfolio updates.

## Trend Data (as of 2025-12-20T23:58:21Z)

- Workflow snapshot (last 30 runs): 1.9h total duration, 17,615,499 tokens, ~$4.70 estimated cost, 509 errors, 268 warnings, 6 missing-tool incidents. Failures cluster in AI Moderator issue_comment runs (e.g., §20401747970, §20401612678, §20401610443, §20401406395).
- Issue stats (weekly slice): 211 total; 21 open / 190 closed. Created since Dec 17: 63; closed since Dec 17: 56. Labels remain concentrated (`ai-generated` 112, `plan` 112, `enhancement` 36, `documentation` 34, `automated-analysis` 27). Unlabeled issues: 19 total (1 open).
- Discussion cadence: New reports on Dec 20 include runtime type behavior (7091), safe output health (7089), daily code metrics (7069), daily issues (7067), lockfile stats (7062), static analysis (7057), prompt analysis (7047), firewall (7048), documentation consolidation (7044), doc noob test (7039), prompt clustering (7037), daily copilot agent analysis (7036), workflow audit (7035).
- Firewall trend: Denial rate 16.8% over last 7 days (17.1% overall 30d); `api.github.com` remains top block (134 hits) signaling persistent MCP misconfiguration in Copilot runs.
- Safe outputs: 45 executions / 1 failure in last 24h (97.78% success); failure due to missing `GH_AW_ASSETS_BRANCH` in Daily Issues Report generator.
