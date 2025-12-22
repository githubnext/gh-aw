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
