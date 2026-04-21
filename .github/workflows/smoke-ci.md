---
name: Smoke CI
description: Smoke CI workflow that comments via safeoutputs CLI without invoking an LLM
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize, reopened]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: copilot
  command: >-
    bash -lc 'mkdir -p /tmp/gh-aw/cache-memory /tmp/gh-aw/repo-memory/default;
    printf "%s\n" "${GITHUB_RUN_ID}" >> /tmp/gh-aw/cache-memory/runs.txt;
    printf "%s\n" "${GITHUB_RUN_ID}" >> /tmp/gh-aw/repo-memory/default/runs.txt;
    if [ "${GITHUB_EVENT_NAME}" = "pull_request" ]; then
    safeoutputs add_comment --body "✅ smoke-ci: safeoutputs CLI comment + comment-memory run (${GITHUB_RUN_ID})";
    mkdir -p /tmp/gh-aw/comment-memory;
    HAIKU="CI lights the path\nGreen checks bloom at dawn\nQuiet bots still sing";
    if compgen -G "/tmp/gh-aw/comment-memory/*.md" > /dev/null; then
    for memory_file in /tmp/gh-aw/comment-memory/*.md; do printf "\n%s\n" "$HAIKU" >>
    "$memory_file"; done; else printf "%s\n" "$HAIKU" >
    /tmp/gh-aw/comment-memory/default.md; fi; else safeoutputs noop --message "smoke-ci:
    push event - no PR context, no action needed"; fi'
tools:
  mount-as-clis: true
  cache-memory: true
  repo-memory:
    branch-name: memory/smoke-ci
    file-glob: "runs.txt"
safe-outputs:
  comment-memory: true
  add-comment:
    hide-older-comments: true
    max: 1
  threat-detection: false
features:
  mcp-cli: true
timeout-minutes: 5
strict: true
---

Run exactly one `safeoutputs` CLI comment action, append the run ID to cache-memory and repo-memory `runs.txt`, and append a 3-line haiku to comment-memory markdown file(s).
If there is no PR context, use `safeoutputs noop` with a brief message.
Do not call any LLM tools or perform any additional analysis.
