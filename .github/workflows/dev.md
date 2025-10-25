---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  edit:
  web-fetch:
safe-outputs:
  push-to-pull-request-branch:
    if-no-changes: ignore
---

# Quote of the Day

Fetch a quote of the day and push it to `docs/quotes.md` in the pull request branch.

## Instructions

1. Fetch a quote of the day from a free API (e.g., https://api.quotable.io/random or similar)
2. Format the quote nicely in markdown with the author and date
3. Write the quote to `docs/quotes.md`
4. The file should contain:
   - The quote text
   - The author
   - Today's date
   - Formatted in clean markdown

Example format:
```markdown
# Quote of the Day

**Date**: [current date]

> [quote text]

— **[author]**
```

5. Use the push-to-pull-request-branch safe output to commit the changes
