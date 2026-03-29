---
"gh-aw": patch
---

Security: exclude `COPILOT_GITHUB_TOKEN` and `GITHUB_MCP_SERVER_TOKEN` from the agent container's visible environment using AWF's new `--exclude-env` flag (requires AWF v0.26.0+). This prevents a prompt-injected agent from exfiltrating these tokens via bash tools such as `env` or `printenv`. AWF's API proxy handles authentication for these tokens transparently. Bumps the default firewall version to v0.26.0.
