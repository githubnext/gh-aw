## Known Patterns (2025-12-20)

- Safe outputs config gap persists: Daily Issues Report safe_outputs failed once for missing `GH_AW_ASSETS_BRANCH`, indicating workflows still ship without required env vars.
- Copilot GitHub API blocks continue: Firewall report shows `api.github.com` remains top-denied domain (23% of denials) due to missing GitHub MCP configuration in Copilot workflows.
- AI Moderator instability: Multiple consecutive AI Moderator runs failed on issue_comment triggers (run IDs around §20401747970), suggesting a reproducible configuration or permission fault.
- Runtime-schema skew: Runtime accepts string `max-turns` while schema declares integer; schema documentation needs to align with runtime behavior.
- Beginner doc friction remains: Noob test reaffirms missing “what is this tool” framing, glossary gaps, and unclear lockfile guidance for first-time users.
- JavaScript surge: Code metrics note ~83% JS LOC growth over 30d with a short-term dip in test ratio, highlighting need to keep JS tests and docs in sync.
