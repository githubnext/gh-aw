
**Copilot Assignment Failed**: The workflow created an issue but could not assign the Copilot coding agent to it. This typically happens when:

- The `GH_AW_AGENT_TOKEN` secret is missing or has expired
- The token does not have the `issues: write` permission
- The Copilot coding agent is not available for this repository
- GitHub API credentials are invalid (`Bad credentials`)

**Failed assignments:**
{issues}

To resolve this, verify that:
1. The `GH_AW_AGENT_TOKEN` secret is configured in your repository settings
2. The token belongs to an account with an active Copilot subscription
3. The token has `issues: write` permission for this repository

```bash
gh aw secrets set GH_AW_AGENT_TOKEN --value "YOUR_TOKEN"
```

See: https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/auth.mdx
