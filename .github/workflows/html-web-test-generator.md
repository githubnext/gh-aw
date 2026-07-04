---
emoji: 🧪
description: Analyzes code changes from the last 24 hours and generates Playwright web tests for any modified HTML pages
on:
  schedule: daily on weekdays
  workflow_dispatch:
max-ai-credits: 500
permissions:
  contents: read
  pull-requests: read
tools:
  edit:
  bash:
    - "cat *"
    - "date *"
steps:
  - name: Discover HTML changes and coverage gaps
    run: |
      mkdir -p /tmp/gh-aw/data
      TODAY=$(date -u +%Y-%m-%d)

      # Find HTML files modified in the last 24 hours
      MODIFIED=$(git log --since="24 hours ago" --name-only --pretty=format: -- "*.html" \
        | sort -u | grep -v '^$')

      if [ -z "$MODIFIED" ]; then
        echo '{"run_date":"'"$TODAY"'","files":[]}' > /tmp/gh-aw/data/html-changes.json
        exit 0
      fi

      # Build JSON array: one entry per HTML file with slug, test_path, has_test
      jq -n \
        --arg today "$TODAY" \
        --argjson files "$(
          echo "$MODIFIED" | while IFS= read -r f; do
            stem=$(basename "$f" .html)
            dir=$(dirname "$f" | sed 's|^docs/public/||')
            slug=$([ "$dir" = "." ] && echo "$stem" || echo "${dir//\//-}-$stem")
            test_path="docs/tests/${slug}.spec.ts"
            has_test=$([ -f "$test_path" ] && echo "true" || echo "false")
            printf '{"html_path":"%s","slug":"%s","test_path":"%s","has_test":%s}\n' \
              "$f" "$slug" "$test_path" "$has_test"
          done | jq -s '.'
        )" \
        '{run_date: $today, files: $files}' \
        > /tmp/gh-aw/data/html-changes.json
safe-outputs:
  create-pull-request:
    title-prefix: "[web-tests] "
    labels: [automation, web-tests]
    draft: true
    expires: 7d
    protected-files: fallback-to-issue
    allowed-files:
      - "docs/tests/**/*.spec.ts"
network:
  allowed:
    - defaults
timeout-minutes: 20
---

# HTML Web Test Generator

Generate Playwright spec files for HTML pages that were modified in the last 24 hours and lack test coverage.

## Input

Read `/tmp/gh-aw/data/html-changes.json`. It has this shape:

```json
{
  "run_date": "YYYY-MM-DD",
  "files": [
    { "html_path": "docs/public/editor/index.html", "slug": "editor-index",
      "test_path": "docs/tests/editor-index.spec.ts", "has_test": false }
  ]
}
```

- If `files` is empty → `noop`: "No HTML files were modified in the last 24 hours."
- If all entries have `has_test: true` → `noop`: "All modified HTML files already have Playwright test coverage."

## Generate Tests

For each entry where `has_test` is `false`:

1. `cat <html_path>` to read the page structure.
2. Write `<test_path>` using the `edit` tool following the conventions of existing files in `docs/tests/`:
   - `import { test, expect } from '@playwright/test';`
   - `test.describe('<Page Name>')` block
   - `beforeEach`: navigate to the page URL (derived from `html_path` relative to `docs/public/`) + `waitForLoadState('networkidle')`
   - At least three tests: page renders (key heading/landmark visible), interactive elements (buttons/links present), accessibility basics (non-empty title, alt text on images)
   - `const` over `let`; use `page.locator()`; one-line comment per test

## Create Pull Request

After writing all test files, open a PR titled:
`Add Playwright tests for HTML pages modified on <run_date>`

PR body (GitHub-flavored markdown):

```
## Web Test Generation Report — <run_date>

### Tests Generated
<bullet list: test_path — one-line coverage description>

### Already Covered
<bullet list of html_path entries where has_test was true, or "None">
```

Call `noop` when no test files were written.
