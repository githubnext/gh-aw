---
name: First Contributor Welcome
description: Greet first-time contributors when they open their first issue or pull request.
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
safe-outputs:
  add-comment:
    max: 1
---

If `author_association` is `FIRST_TIME_CONTRIBUTOR`, `FIRST_TIMER`, or `NONE` and the author is not a bot, post a warm welcome comment thanking them for their contribution.
Include links to CONTRIBUTING.md and CODE_OF_CONDUCT.md, and offer to answer any questions they might have.
Otherwise call `noop` — existing contributors do not need a welcome message.
