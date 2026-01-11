---
name: Code Simplifier
description: Analyzes recently modified code and creates pull requests with simplifications that improve clarity, consistency, and maintainability while preserving functionality
on:
  schedule: daily
  skip-if-match: 'is:pr is:open in:title "[code-simplifier]"'

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: code-simplifier

imports:
  - shared/reporting.md

safe-outputs:
  create-pull-request:
    title-prefix: "[code-simplifier] "
    labels: [refactoring, code-quality, automation]
    reviewers: [copilot]

tools:
  github:
    toolsets: [default]
  edit:
  bash:
    - "git log --since='24 hours ago' --pretty=format:'%H %s' --no-merges"
    - "git show *"
    - "date -d '1 day ago' '+%Y-%m-%d'"
    - "date -v-1d '+%Y-%m-%d'"
    - "date +%Y-%m-%d"
    - "make test-unit"
    - "make lint"
    - "make build"
    - "npm test"
    - "npm run lint"
    - "npm run build"
    - "pytest"
    - "flake8 ."
    - "pylint ."
    - "python -m py_compile *.py"
    - "cat AGENTS.md"
    - "cat DEVGUIDE.md"
    - "cat CLAUDE.md"
    - "cat .github/instructions/developer.instructions.md"
    - "find . -name '*.go' -o -name '*.js' -o -name '*.ts' -o -name '*.tsx' -o -name '*.cjs' -o -name '*.py'"
    - "grep -r '*' ."

timeout-minutes: 30
strict: true
---

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
@./agentics/code-simplifier.md
