---
on:
  schedule:
    - cron: "0 9 * * 1"
  workflow_dispatch:
engine:
  id: claude
  max-turns: 5
permissions: read-all
safe-outputs:
  create-issue:
    title-prefix: "[Test] "
    labels: [test, automation]
    max: 1
  add-issue-comment:
tools:
  web-search:
  web-fetch:
network: defaults
cache:
  key: test-cache-${{ github.repository }}
  path: 
    - /tmp/test
  restore-keys:
    - test-cache-
timeout_minutes: 10
if: ${{ github.event_name == 'schedule' }}
---

# Test Agentic Workflow

This is a test workflow to validate the VSCode extension syntax highlighting and schema validation.

## Features to Test

- **YAML frontmatter highlighting**
- **Markdown content highlighting**
- **GitHub expressions**: ${{ github.actor }}
- **Include directives**: @include shared/common.md

## Code Examples

```yaml
# YAML code block
test:
  value: true
```

```javascript
// JavaScript code block
console.log("Hello, world!");
```

## Task List

- [ ] Test syntax highlighting
- [ ] Test schema validation
- [ ] Test auto-completion
- [x] Create test file