# ADR-70997: Enforce Detection Gates for Custom Workflow Jobs in the Compiled Job Graph

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

### Context

Agentic workflows can declare custom top-level `jobs` alongside the compiler-owned jobs (`activation`, `agent`, `detection`, `safe_outputs`, and so on). Threat detection runs after `agent` and decides whether the agent's output is safe to use. Before this change, a custom job could use agent results before detection finished. It could also skip a failed detection with a status-override condition such as `if: always()` or `failure()`, or sit between `agent` and `detection` in the graph. Warn-mode detection makes this harder: the `detection` job can succeed while its `detection_success` output reports unsafe results, so checking only the job result is not enough. Compiler threat specification CTR-005 (Unsafe Output Route) did not yet cover custom jobs.

### Decision

We will require every custom job to run either before `agent` or after a successful, clean `detection`. The compiler enforces this on the final job graph, after typed output-derived dependencies have been lowered. A new pass, `guardCustomJobDetection`, builds a transitive `JobDependencyGraph` from the compiled jobs and does three things:
- rejects custom jobs that are unordered relative to `agent` or that sit between `agent` and `detection`
- adds `detection` directly to the `needs` of every downstream custom job, including reusable workflow calls
- wraps the job's `if` in an outer guard that cannot be overridden: `needs.detection.result == 'success' && needs.detection.outputs.detection_success == 'true' && (<authored condition>)`

The `custom-job-detection` lint rule enforces the same contract on the Actions and AW ASTs with transitive dependencies. Workflows that explicitly disable detection are exempt. The contract is recorded in CTR-005 of compiler threat specification 1.0.66.

### Alternatives Considered

#### Alternative 1: Lint-only enforcement (authors must add the gate themselves)

Keep `custom-job-detection` as a lint rule that reports missing `needs: detection` or unguarded status functions, without changing the compiled output. This keeps author intent visible in the compiled YAML and avoids rewriting conditions. We rejected it because lint covers only what it analyzes. It does not resolve imports, so shared fragments are missed, and authors can ignore or misconfigure it. A security boundary for tainted agent output should hold no matter what the author writes.

#### Alternative 2: Gate only on detection job status (`needs.detection.result == 'success'`)

Rely on the job result, or on Actions' implicit success gate, without checking the `detection_success` output. This produces a simpler guard. We rejected it because warn-mode findings leave the `detection` job successful while its output marks the results unsafe. Downstream jobs would still receive flagged output.

#### Alternative 3: Enforce on the authored (pre-lowering) dependency graph

Validate and inject guards when custom jobs are first parsed. We rejected this because the compiler later adds dependencies from typed output references. Checking the authored graph could miss jobs that only become downstream of `agent` after lowering. It could also misclassify jobs that sit between `agent` and `detection`.

### Consequences

#### Positive
- Downstream custom jobs, including reusable workflow calls and jobs using `always()`, `failure()`, or `cancelled()`, can no longer use agent output after detection fails, is cancelled or skipped, or reports warn-mode findings.
- The compiler and the lint rule share one transitive dependency model (`aw.JobDependencyGraph`), and CTR-005 records the contract in the threat specification with mapped tests.
- Problem graphs, such as unordered custom jobs or jobs between `agent` and `detection`, fail at compile time with a clear `jobs.<id>.needs` error.

#### Negative
- Authors can no longer run post-agent custom jobs on detection failure, for example to send failure notifications or clean up. The guard rewrites authored `if` conditions and cannot be turned off without disabling detection entirely.
- The compiled conditions are longer and partly redundant, for example `needs.detection.result == 'success'` appears both in the guard and in the authored condition. This made lock files across many workflows (smoke, release, and others) churn.
- This can break existing workflows: custom jobs that are neither before `agent` nor after `detection`, or that `detection` depends on while also depending on `agent`, now fail to compile. Authored `if` values must be a single valid expression, so interpolated text inside them is rejected.

#### Neutral
- Compiler-owned jobs and `safe-outputs.jobs` are exempt from this pass because they already get compiler-generated guards.
- Workflows that explicitly disable detection are exempt, so whether the gate applies depends on detection being enabled.
- The reference docs (`steps-jobs.md`, `workflow-linters.md`) and the threat specification changelog were updated to describe the contract.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
