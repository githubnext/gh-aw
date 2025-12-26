## Flagged Items for Monitoring (2025-12-26)

- MCP client breakage: audit reports invalid URL and missing `read_buffer.cjs` across Smoke Copilot variants plus safeoutputs failures in Smoke Claude—likely tied to recent MCP HTTP transport changes (commit 2dd6edb); blocks MCP coverage.
- GitHub MCP absent in `research.md`/`daily-news.md`/`daily-firewall-report.md`, causing 92 GitHub API/Web blocks and inflated firewall denials; urgent to wire GitHub MCP toolsets.
- Missing tools persist: safeinputs-gh absent in Smoke Codex/Safe Inputs; Tidy still lacks make/go/npm/golangci-lint; Spec-Kit Execute blocked on directory creation.
- Permission-denied noise (90 hits) across AI Moderator, Issue Monster, Tidy, Spec-Kit, and smoke workflows; needs sandbox/permission review to curb warning volume.
- Success rate regressed to ~65% (24 failures in 68 runs) with 4 missing-tool and 5 MCP failures in last day; health trend reversing prior rebound.

## Flagged Items for Monitoring (2025-12-25)

- Daily Copilot PR Merged still throws a missing-tool (safeinputs/GITHUB_TOKEN wiring) in latest run (§20507004818); fix safeinputs-gh config and token availability.
- Firewall denials climbing again (29.7% weekly; api.github.com/github.com + LinkedIn blocks). Copilot workflows like `research.md`/`daily-news.md` still need GitHub MCP toolsets instead of direct API/network allowlists.
- Tidy lacks build tools (make/go/npm/golangci-lint) leading to repeated permission/missing-tool warnings; Issue Monster hitting GitHub search errors; Smoke Copilot Playwright missing Playwright MCP; Spec-Kit Execute blocked on directory creation; all surfaced in Dec 25 audit (§20498588239).
- Copilot token spend concentrated: CI Cleaner alone is 26% of 3-day Copilot tokens; Issue Monster + Tidy are high-frequency drivers. Efficiency is improving (-68% DoD) but these workflows remain hotspots.

## Flagged Items for Monitoring (2025-12-24)

- Daily Copilot PR Merged missing safeinputs-gh/invalid URL with GITHUB_TOKEN unavailable (run §20488854921); requires MCP wiring and token availability fix.
- Artifact Summary on `copilot/fix-discussion-permissions-issue` logged permission-denied warnings while generating analysis scripts; check permissions and safe output setup before rerun.
- Issue backlog ticking up: 47 open (was 34), with 113 created vs 100 closed in 3 days; monitor if creation rate stays above closures despite unlabeled set shrinking to 18 (2 open).
- Firewall denials temporarily at zero in last 5 runs, but recent history had ~30% denial rate—verify sustained improvement across more runs.

## Flagged Items for Monitoring (2025-12-23)

- Burst of failures from branch `copilot/create-custom-action-setup-activation` (multiple workflows failed in one push: Plan Command, Release, Q, Nitpick Reviewer, DeepReport, Scout, Spec-Kit dispatcher, etc.); needs root-cause to stop failure noise.
- Daily Copilot PR Merged still reports a missing tool (run §20464285031) despite success; confirm tool wiring/safeinputs.
- Firewall denial rate now ~30% with GitHub API and LinkedIn top blocks; Copilot workflows still lack GitHub MCP config and LinkedIn access intent is unclear.
- High-cost hourly CI Cleaner continues to dominate Copilot token spend (~29% of cost over 4 days); consider frequency reduction or no-op exit.
- Unlabeled issues grew to 20 total (2 open); maintain triage to prevent drift.

## Flagged Items for Monitoring (2025-12-22)

- Weekly Issue Summary workflow failing (last 7d logs show single failure among 10 runs); needs triage to restore weekly reporting.
- Missing GitHub MCP read_issue capability in Dev workflow causing missing-tool error (run §20435819459); add tool or adjust permissions.
- Daily Copilot PR Merged needs safeinputs-gh tool to avoid missing-tool failures (run §20435787142); ensure workflow wiring matches instructions.
- Issue backlog uptick: open issues climbed to 45 (was 21 on 12/20) with 85 created in last 3 days; watch for sustained growth.
- Unlabeled issues remain (19 total, 5 open); consider triage to prevent drift.

## Flagged Items for Monitoring (2025-12-20)

- Safe outputs env var miss: Daily Issues Report generator lacks `GH_AW_ASSETS_BRANCH`, causing the lone safe_outputs failure; needs workflow config fix.
- AI Moderator failures: Multiple consecutive AI Moderator issue_comment runs failed (e.g., [§20401747970](https://github.com/githubnext/gh-aw/actions/runs/20401747970), [§20401612678](https://github.com/githubnext/gh-aw/actions/runs/20401612678), [§20401610443](https://github.com/githubnext/gh-aw/actions/runs/20401610443), [§20401406395](https://github.com/githubnext/gh-aw/actions/runs/20401406395)); investigate trigger inputs/permissions.
- Copilot GitHub access: Firewall report shows ongoing `api.github.com` blocks; Copilot workflows still need consistent GitHub MCP server configuration instead of direct API calls.
- Schema mismatch: Runtime accepts string `max-turns` while schema is integer-only; update schema or tighten runtime for consistency.
- Onboarding gaps: Doc noob test highlights missing “what is this tool” framing, glossary/tooling definitions, and clearer lockfile guidance; actionable doc quick wins remain open.
