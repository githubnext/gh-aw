---
"gh-aw": patch
---

Fail-fast safe-output execution when a code-push step such as `push_to_pull_request_branch` or `create_pull_request` fails and include the failure context in the agent conclusion. Remaining safe outputs are now cancelled with a clear reason and the code push errors surface in the failure issue/comment.
