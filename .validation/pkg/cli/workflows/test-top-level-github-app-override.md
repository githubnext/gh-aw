---
on:
  issues:
    types: [opened]
  reaction: eyes
  github-app:
    app-id: ${{ vars.ACTIVATION_APP_ID }}
    private-key: ${{ secrets.ACTIVATION_APP_KEY }}
permissions:
  contents: read

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  github-app:
    app-id: ${{ vars.SAFE_OUTPUTS_APP_ID }}
    private-key: ${{ secrets.SAFE_OUTPUTS_APP_KEY }}
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Section-Specific GitHub App Takes Precedence Over Top-Level

This workflow demonstrates that section-specific github-app configurations take precedence
over the top-level github-app fallback.

- `on.github-app` is explicitly set → activation uses ACTIVATION_APP_ID, not APP_ID
- `safe-outputs.github-app` is explicitly set → safe-outputs uses SAFE_OUTPUTS_APP_ID, not APP_ID

The top-level github-app (APP_ID) is only used as a fallback when sections do not define
their own github-app.
