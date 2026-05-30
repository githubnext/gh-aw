---
"gh-aw": patch
---

Allow `add_comment` safe outputs to be posted even when a preceding code push operation (`create_pull_request` or `push_to_pull_request_branch`) fails. The comment body is annotated with a warning note describing the failure so users see the outcome. Previously, the status comment was silently cancelled when patch application failed, leaving no update in the issue or PR.
