---
"gh-aw": patch
---

Fixed `create_pull_request` transport artifacts (patch and bundle) being based on the merge-base with the default branch when a workflow runs from a ref that is not contained in the default branch (for example a `workflow_dispatch` on a feature branch).

In that situation the merge-base is far behind the checked-out commit, so the generated patch/bundle contained every commit of the dispatched branch in addition to the agent's own commits. On a partial clone (a `blob:none` fetch marks `origin` as a promisor remote) it failed outright: diffing from the older base required base-side blobs that were never fetched, and the lazy hydration fetch is unauthenticated because gh-aw checks out with `persist-credentials: false`.

Full mode now prefers `GITHUB_SHA` as the base when it is an ancestor of the agent branch but is not contained in the default branch. The patch then holds exactly the agent's commits and needs no objects beyond the checkout. Runs from the default branch are unchanged, since there the merge-base already equals `GITHUB_SHA`.

Also improved the diagnostics: branch existence is now determined from local refs only, so a network or authentication failure is no longer reported as `Branch 'X' does not exist locally`, and failures that look like a failed lazy object fetch on a partial clone now say so explicitly.
