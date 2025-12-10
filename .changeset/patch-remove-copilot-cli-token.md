---
"gh-aw": patch
---

Remove support for the `COPILOT_CLI_TOKEN` environment variable.

This change removes `COPILOT_CLI_TOKEN` from the Copilot engine secrets list, token
precedence logic, and trial support. Documentation and tests were updated. Workflows
that currently rely on `COPILOT_CLI_TOKEN` must migrate to `COPILOT_GITHUB_TOKEN`.

Migration example:

```bash
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "(your-github-pat)"
```
