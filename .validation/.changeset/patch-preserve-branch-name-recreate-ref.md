---
"gh-aw": patch
---

Add `recreate-ref` option to `create-pull-request` safe outputs. When set to `true` together with `preserve-branch-name: true`, the handler force-deletes an existing remote branch ref and recreates it from the agent's local HEAD (force-push semantics), enabling workflows that intentionally maintain long-lived reusable branches across iterations (e.g. autoloop programs whose previous PR was merged but whose branch still exists). When `recreate-ref` is omitted or `false` (the default), an existing remote branch under `preserve-branch-name: true` causes the handler to fall back (e.g. open an issue) rather than overwrite the remote ref.
