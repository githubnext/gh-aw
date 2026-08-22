---
"gh-aw": patch
---

Compilation now fails with an actionable error when a `safe-outputs` `github-token` expression of the form `${{ steps.<id>.outputs.<name> }}` cannot resolve in a job that consumes it — either because the job has no step with that id, or because the minting step runs after the first consumer (for example when the step is declared under `safe-outputs.steps`, which runs after the `safe_outputs` checkout). Previously the reference compiled to a lock file that failed `actionlint` and produced an empty token at runtime. Use `pre-steps:` (agent job) and `jobs.safe_outputs.pre-steps:` / `jobs.conclusion.pre-steps:` to mint the token ahead of every consumer.
