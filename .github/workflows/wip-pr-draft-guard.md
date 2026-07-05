---
private: true
emoji: "🚧"
name: WIP PR Draft Guard
description: Auto-convert pull requests with WIP title prefixes to draft and leave guidance
on:
  pull_request:
    types: [opened, edited]
if: startsWith(github.event.pull_request.title, '[WIP]') || startsWith(github.event.pull_request.title, 'WIP:') || startsWith(github.event.pull_request.title, '[wip]') || startsWith(github.event.pull_request.title, 'wip:')
permissions:
  contents: read
  pull-requests: read
  issues: read
  copilot-requests: write
engine:
  id: copilot
  copilot-sdk: true
safe-outputs:
  update-pull-request:
    title: false
    body: false
    max: 1
  add-comment:
    max: 1
    hide-older-comments: true
  noop:
    max: 1
timeout-minutes: 5
---

# WIP PR Draft Guard

When this workflow runs, the triggering pull request title already matches a WIP prefix.

1. If the pull request is not draft, call `update_pull_request` with `draft: true`.
2. Post exactly one short `add_comment` guidance message:
   `This pull request was marked as draft because its title starts with a WIP prefix ([WIP], WIP:, [wip], or wip:). Remove the prefix and mark it ready for review when it's ready.`
3. If no update is needed, call `noop` with a short reason.
