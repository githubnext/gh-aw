---
emoji: "🔥"
on: weekly
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  cli-proxy: true
  github:
safe-outputs:
  create-issue:
    title-prefix: '[dependabot-burner] '
imports:
  - shared/reporting.md


  - shared/observability-otlp.md
---
# Dependabot Burner

- Find all open Dependabot PRs.
- Create bundle issues, each for exactly **one runtime + one manifest file**.

{{#runtime-import shared/noop-reminder.md}}
