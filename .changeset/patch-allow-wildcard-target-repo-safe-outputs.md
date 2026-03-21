---
"gh-aw": patch
---

Fixed safe-output handler parsing to allow `target-repo: "*"` across add-comment, create-issue, create-discussion, close-entity, add-reviewer, and create-pull-request handlers so wildcard-targeted handlers are preserved in `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG`.
