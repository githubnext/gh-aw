---
"gh-aw": patch
---

Handle GitHub Actions PR creation permission errors by setting an `error_message` output
and adding an auto-filed issue handler with guidance when Actions cannot create or
approve pull requests in the repository.

This patch documents the change: the create-pull-request flow now emits a helpful
`error_message` output when permissions block PR creation, and the conclusion job
can use that to file or update an issue with next steps and links to documentation.

