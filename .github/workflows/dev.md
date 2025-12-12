---
on: 
  issues:
    types: [opened]
    lock-for-agent: true
name: Dev
description: Summarize issue description and add as comment
timeout-minutes: 5
strict: false
engine: copilot

permissions: read-all

tools:
  github: false
safe-outputs:
  add-comment:
    max: 1
---

Read the issue description and create a concise summary of it. Post the summary as a comment on the issue.
