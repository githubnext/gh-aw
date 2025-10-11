---
"gh-aw": patch
---

Remove push trigger from repo-tree-map agentic workflow

The workflow now only triggers via manual `workflow_dispatch`, preventing unnecessary automatic runs when the workflow lock file is modified.
