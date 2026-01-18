---
engine: copilot
on: manual
safe-outputs:
  create-issue:
    title: "Example Issue with Visible Expiration"
    expires: 7  # Expires in 7 days
    labels: [example, ephemeral]
---

Create an example issue to demonstrate the new visible expiration format.

The issue body should contain:
- A description of the issue
- Information about automatic expiration

Create the issue with title "Example: Visible Expiration Checkbox" and body:

```
## Example Issue

This issue demonstrates the new visible expiration format for GitHub Agentic Workflows.

### What Changed

Previously, expiration was hidden in an XML comment:
\`\`\`html
<!-- gh-aw-expires: 2026-01-25T12:00:00.000Z -->
\`\`\`

Now, expiration is visible as a checkbox that users can interact with:
\`\`\`markdown
- [x] expires <!-- gh-aw-expires: 2026-01-25T12:00:00.000Z --> on Jan 25, 2026, 12:00 PM UTC
\`\`\`

### Benefits

1. **Visible**: Users can see when the issue will expire
2. **Configurable**: Users can uncheck the box to prevent expiration
3. **Informative**: Shows both ISO date and human-readable format

### How It Works

- The maintenance workflow checks for issues with checked expiration boxes
- Only issues with \`- [x] expires\` (checked) will be automatically closed
- Uncheck the box to keep the issue open past the expiration date
```
