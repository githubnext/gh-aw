---
"gh-aw": minor
---

Add `preserve-branch-name: true` option to `create-pull-request` safe outputs. When enabled, no random salt suffix is appended to the agent-specified branch name. Invalid characters are still replaced for security, and casing is always preserved regardless of this setting. Useful when the target repository enforces branch naming conventions such as Jira keys in uppercase (e.g. `bugfix/BR-329-red`).
