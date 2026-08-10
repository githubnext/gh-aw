## Flagged Items (2026-08-10)

- **[P0 process]** DeepReport repo-memory persistence still broken despite #51172 closed — source file `.github/workflows/deep-report.md` unfixed. Filed dedicated issue with exact line numbers/fix. Verify next cycle: does `memory/deep-report` branch have a new commit?
- **[P0 process]** Systemic "closed without verified fix" pattern across 5 bug lineages (repo-memory, copilot-session transcripts, firewall/MCP logs, Quick Start jargon, Quick Start engine guidance). Filed process-gate issue. Watch whether future DeepReport cycles start citing merged PR SHAs when they claim a fix.
- Firewall/MCP raw-log retention: 0.0% coverage today, 6th closure-without-fix. Filed issue proposing upload-side glob/path-depth hypothesis (by analogy to the repo-memory bug) as an untried debugging angle.
- Copilot Session Insights: 44+ consecutive days zero transcripts (down from "71+ days" reported 2026-08-07 — inconsistent counter, worth reconciling whether this is the same gap restarting or a different counting method). Not re-filed as a 7th duplicate; folded into process issue as evidence.
- Quick Start docs jargon + engine-choice guidance: both flagged again today after 6-7 prior closures each. Not re-filed as new duplicates; folded into process issue as evidence.
- audit-workflows' own recommendations.json/workflow-trends.json frozen at 2026-07-06 snapshot (patch-size revert). Filed issue (reformat to indent=2).
- "Q" and "ESLint Monster" missing gh-aw-detection. Filed issue.
- PR Code Quality Reviewer: 32/38 fleet firewall blocks against api.individual.githubcopilot.com, concurrent with an open credential-auth-failure issue (#51802) — commented with the correlation, did not file separately.
- Cross-engine driver_exit failures: 10% in a fresh 50-run sample (claude/codex/copilot/pi). Chronic known issue (`copilot-sdk-driver-failures`); intentionally not re-filed, per prior cycle's judgment that it's a broader root-cause effort.
