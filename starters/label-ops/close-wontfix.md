---
name: Close Won't Fix
description: Close issues automatically when the wontfix label is applied and post a brief explanation.
on:
  label_command: wontfix
permissions:
  contents: read
  issues: read
safe-outputs:
  add-comment:
    max: 1
  close-issue:
---

Post a polite comment explaining that the issue is being closed as won't fix, summarizing the decision in one sentence based on the issue title and body.
Thank the reporter for raising it and suggest they open a new issue if circumstances change.
Then close the issue.
