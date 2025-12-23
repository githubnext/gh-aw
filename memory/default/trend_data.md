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
