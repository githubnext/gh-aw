---
"gh-aw": minor
---

Remove legacy support for the `GH_AW_COPILOT_TOKEN` secret name.

This change removes the legacy fallback to `GH_AW_COPILOT_TOKEN`. The effective token lookup chain is now:

- `COPILOT_GITHUB_TOKEN` (recommended)
- `GH_AW_GITHUB_TOKEN` (legacy)

If you were relying on `GH_AW_COPILOT_TOKEN`, update your repository secrets and workflows to use `COPILOT_GITHUB_TOKEN` or `GH_AW_GITHUB_TOKEN`.

## Migration

Run the following to remove the old secret and add the new one:

```bash
gh secret remove GH_AW_COPILOT_TOKEN -a actions
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "YOUR_PAT"
```

This follows the precedent set when `COPILOT_CLI_TOKEN` was removed in v0.26+. All workflow lock files have been regenerated to reflect this new token chain.

