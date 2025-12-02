---
"gh-aw": patch
---

Refactor the create-issue Copilot assignment to run in a separate step

Created a dedicated output (`issues_to_assign_copilot`) and separate
post-step that uses the agent token (`GH_AW_AGENT_TOKEN`) to assign
the agent to newly-created issues. This change isolates Copilot
assignment from regular post-steps and improves permission separation.

