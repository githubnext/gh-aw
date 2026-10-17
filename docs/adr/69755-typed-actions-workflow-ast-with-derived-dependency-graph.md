# ADR-69755: Typed GitHub Actions Workflow AST with Derived Dependency Graph

**Date**: 2026-10-17
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify other reviewers]

---

### Context

The gh-aw compiler builds GitHub Actions lockfiles by assembling `workflow.Job` values whose `needs`, `if`, `outputs`, `with`, and `secrets` fields are hand-written strings (for example, `If: "needs.safe_outputs.outputs.call_workflow_name == '...'"` with a separately maintained `Needs: []string{"safe_outputs"}`). Generators must keep expression strings and dependency lists consistent by hand, so a missing `needs` edge, a reference to a nonexistent step output, or a forward reference is only caught (if at all) by GitHub at runtime. A typed AST design already exists in `githubnext/ado-aw`, but it targets multiple pipeline backends (including Azure), which gh-aw does not need. Any change must preserve existing lockfile formatting, action-pin comments, scalar quoting, and dry-run behavior so that the ~344 compiled workflows do not drift.

### Decision

We will add a GitHub Actions-only typed workflow AST in a new package, `pkg/workflow/actions`, modelling workflows, runner jobs, reusable-workflow calls, `run`/`uses` steps, and typed expressions (`OutputRef`, `SecretRef`, `VariableRef`, `Binary`, `Call`, etc.). A `Resolve` pass validates output references (job-scoped step IDs, no forward references, no missing producers, no cycles, no duplicate IDs) and automatically derives `needs` edges and cross-job output bindings without mutating its input; `Lower`/`Emit` produce deterministic YAML. The existing compiler integrates it incrementally: `workflow.Job` gains an optional `Actions *actions.Job` contribution that `JobManager.resolveActions()` resolves after job augmentation and before rendering, leaving existing string-based job fields untouched. We port the ado-aw design but deliberately drop the pipeline package, Azure backend, and platform-neutral envelopes.

### Alternatives Considered

#### Alternative 1: Keep string-based job fields and add post-hoc validation

Continue writing `needs`/`if`/`outputs` as strings and add a validator that parses expressions to check that referenced jobs appear in `needs`. This is lower churn, but it keeps dependency maintenance manual, requires parsing free-form expression strings, and cannot structurally prevent issues such as interpolating dynamic values into `run` scripts. It was rejected because it detects inconsistencies rather than eliminating them.

#### Alternative 2: Port the multi-backend ado-aw AST wholesale

Adopt the full ado-aw model, including its platform-neutral pipeline layer and Azure backend, for possible future CI targets. This would maximize code reuse with the upstream design, but adds abstraction layers, compatibility aliases, and concepts (non-Actions properties) that gh-aw never emits. It was rejected in favor of a native Actions-only model that rejects non-Actions properties outright.

#### Alternative 3: Rewrite the whole compiler onto the AST in one change

Replace `workflow.Job` rendering entirely with `actions.Lower`. This would give a single source of truth, but would risk lockfile drift across hundreds of generated workflows (formatting, pin comments, quoting) and require a very large, hard-to-review change. It was rejected in favor of the incremental `Job.Actions` contribution bridge.

### Consequences

#### Positive
- `needs` edges and job-output bindings are derived from typed `OutputRef` usage, removing a class of hand-maintained dependency bugs (missing producers, forward references, cycles are rejected at compile time).
- Generators can migrate one job at a time (starting with call-workflow dispatch conditions) without changing YAML for untouched jobs; graph construction is validated against all 385 repository workflow files.
- Expressions are rendered consistently and dynamic values are steered through `env` rather than interpolated into `run` bodies, supporting template-injection safety.

#### Negative
- Two parallel representations of job fields coexist (string fields and `Job.Actions`); the doc explicitly forbids setting a field in both forms, which is a convention not fully enforced by types.
- Adds ~2,000 lines of new code plus a new compile pass (`resolveActions`) on every compilation, increasing maintenance surface and compiler complexity.
- Rendered expressions change shape (e.g., `if: ${{ (needs.... == '...') }}` instead of a bare expression), causing lockfile diffs as generators migrate.

#### Neutral
- Existing opaque output strings are passed into the graph as `Literal` values, so legacy outputs participate in validation but are not themselves type-checked.
- The design intentionally drops multi-backend support; adding another CI target later would require a new decision.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
