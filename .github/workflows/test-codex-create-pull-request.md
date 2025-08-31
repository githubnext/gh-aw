---
on:
  workflow_dispatch:

engine: 
  id: codex

safe-outputs:
  create-pull-request:
    title-prefix: "[codex-test] "
    labels: [codex, automation, bot]
---

Add a file "TEST.md" with content "Hello from Codex"

Add a log file "foo.log" containing the current time. This is just a log file and isn't meant to go in the pull request.

Create a pull request with title "A Pull Request from Codex" and body "World"

Add a haiku about GitHub Actions and AI to the PR body.