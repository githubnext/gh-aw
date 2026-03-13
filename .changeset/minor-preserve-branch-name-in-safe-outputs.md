---
"gh-aw": minor
---

Add `preserve-branch-name: true` option to `create-pull-request` safe outputs. When enabled, the agent-specified branch name is used as-is — no random salt suffix is appended and the name is not lowercased. Invalid characters are still replaced for security. Useful when the target repository enforces branch naming conventions such as Jira keys in uppercase (e.g. `bugfix/BR-329-red`).
