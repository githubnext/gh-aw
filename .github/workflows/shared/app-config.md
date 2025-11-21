---
safe-outputs:
  app:
    id: ${{ vars.ORG_APP_ID }}
    secret: ${{ secrets.ORG_APP_PRIVATE_KEY }}
---

# Shared GitHub App Configuration

This shared workflow provides organization-wide GitHub App configuration for safe outputs.

## Usage

Import this configuration in your workflows:

```yaml
---
on: issues
permissions:
  contents: read
imports:
  - shared/app-config.md
safe-outputs:
  create-issue:
---

# Your Workflow

Your workflow content here...
```

## Override

You can override the app configuration in your workflow if needed:

```yaml
---
on: issues
permissions:
  contents: read
imports:
  - shared/app-config.md
safe-outputs:
  create-issue:
  app:
    id: ${{ vars.CUSTOM_APP_ID }}
    secret: ${{ secrets.CUSTOM_APP_SECRET }}
---

# Your Workflow

Your workflow content here...
```
