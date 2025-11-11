---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: write
fingerprint: example-fingerprint-2024
safe-outputs:
  create-issue:
    title-prefix: "[Example] "
    labels: [example, automated]
---

# Example Fingerprint Workflow

This is an example workflow that demonstrates the fingerprint feature.

When this workflow creates an issue, it will include a hidden HTML comment:

```html
<!-- fingerprint: example-fingerprint-2024 -->
```

This fingerprint can be used to:
- Search for all assets created by this workflow
- Track and manage related assets across the repository
- Filter issues, discussions, PRs, and comments by fingerprint

The fingerprint must be:
- At least 8 characters long
- Contain only alphanumeric characters, hyphens, and underscores
- Unique across your workflows for effective tracking

## Example Output

Create an issue with the title "Test Issue with Fingerprint" and body content explaining how the fingerprint feature works.
