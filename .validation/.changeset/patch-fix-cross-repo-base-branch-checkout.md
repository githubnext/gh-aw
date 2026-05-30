---
"gh-aw": patch
---

Fixed `safe_outputs` checkout failure when cross-repo `create-pull-request` items are present. The `extract-base-branch` step now skips items whose `repo` field is set to a repository other than the workflow repository (`GITHUB_REPOSITORY`), so a feature branch that only exists in a target repo can no longer cause the workflow-repo checkout to fail.
