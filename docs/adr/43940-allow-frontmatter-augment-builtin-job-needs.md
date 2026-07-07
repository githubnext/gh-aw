# ADR-43940: Allow Frontmatter Augmentation of Compiler-Generated Job `needs` Dependencies

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw workflow compiler generates a fixed set of built-in jobs (`activation`, `agent`, `detection`, `safe_outputs`, `conclusion`) with implicit `needs` dependencies derived from `engine.env` inference. Authors who need a custom job (e.g., a model-selection job) to complete before a built-in job runs have no supported mechanism to express this ordering. Implicit inference does not cover all dependency patterns — for example, a custom job whose output is consumed via `${{ jobs.select_model.outputs.model }}` inside a built-in job's configuration is not automatically detected. Without a way to express explicit ordering, workflows are blocked by gaps in the inference engine.

### Decision

We will introduce a post-graph compiler pass (`applyBuiltinJobNeedsAugmentations`) that reads `jobs.<built-in>.needs` entries from the workflow frontmatter and merges them additively into the compiler-computed dependency list for each named built-in job. The merge is de-duplicated and strictly additive — compiler-required dependencies are never removed. Alias normalization (`pre-activation`/`pre_activation`, `safe-outputs`/`safe_outputs`) is applied so authors can use either hyphen or underscore variants. Validation errors are raised at compile time for unknown targets, self-dependencies, and augmenting a built-in job that is not generated in the current workflow.

### Alternatives Considered

#### Alternative 1: Expand Implicit `engine.env` Inference

Extend the existing `engine.env` dependency inference to detect more patterns — for example, statically analyze workflow expressions (`${{ jobs.X.outputs.Y }}`) within built-in job configuration to auto-derive the dependency on job `X`. This would require no new frontmatter schema and would be transparent to authors. It was not chosen because static expression analysis over arbitrary frontmatter values is fragile and difficult to extend; it would silently miss references in computed strings, template variables, or job steps that are not directly in the job's frontmatter config block.

#### Alternative 2: Introduce a Distinct `before:` Ordering Directive

Add a new top-level or per-job frontmatter key (e.g., a `before: [custom_job]` field under each built-in job section) with explicit ordering semantics separate from GitHub Actions' `needs`. This would create a clean API boundary between compiler-internal dependency management and author-supplied ordering hints. It was not chosen because it diverges from the GitHub Actions `needs` convention that workflow authors already understand, and introduces an additional concept without providing meaningful additional safety or expressive power.

### Consequences

#### Positive
- Authors can now precisely express dependency ordering for built-in jobs without workarounds or reliance on implicit inference.
- The additive-only merge design guarantees existing compiler-computed dependencies are never removed, preserving correctness invariants for built-in job execution order.
- Validation at compile time (unknown target, self-dependency, absent generated job) surfaces errors early, preventing silent misconfiguration at runtime.

#### Negative
- The compiler now has a load-bearing pass ordering constraint: augmentation must run after the full job graph is constructed (so all job names are known for validation) but before conclusion normalization (which depends on the final dependency set). Future changes to the `buildJobs` pipeline must respect this ordering.
- The `jobs.<built-in>` namespace in the frontmatter is now dual-purpose: it was previously used only for pre-steps and setup configuration; it now also carries dependency information. This increases the semantic surface authors must understand.

#### Neutral
- Alias normalization (`pre-activation` → `pre_activation`, `safe-outputs` → `safe_outputs`) is applied consistently for both the target job name and the needs entries. Authors may use either form.
- Integration tests are added to the `pkg/workflow` package covering the happy path and the unknown-target error path; the self-dependency and absent-generated-job error paths are not yet integration-tested.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
