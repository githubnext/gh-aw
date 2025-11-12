---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: write
campaign: example-fingerprint-2024
safe-outputs:
  create-issue:
    title-prefix: "[Example] "
    labels: [example, automated]
---

# Example Fingerprint Workflow

This is an example workflow that demonstrates the campaign feature.

When this workflow creates an issue, it will include a hidden HTML comment:

```html
<!-- campaign: example-fingerprint-2024 -->
```

This campaign can be used to:
- Search for all assets created by this workflow
- Track and manage related assets across the repository
- Filter issues, discussions, PRs, and comments by campaign

The campaign must be:
- At least 8 characters long
- Contain only alphanumeric characters, hyphens, and underscores
- Unique across your workflows for effective tracking

## Example Output

Create an issue with the title "Test Issue with Fingerprint" and body content explaining how the campaign feature works.
