# ADR-43641: Add BinEval-Style Evals Support to Workflow Compiler

**Date**: 2026-07-06
**Status**: Draft
**Deciders**: Unknown

---

### Context

Workflow authors currently have no structured way to verify whether an agentic run met its goals after `safe_outputs` completes. Without automated evaluation, quality regressions are only detected through manual review. The project needs a machine-readable, historically trackable signal that answers binary questions such as "Did the generated code compile?" or "Is the implementation limited to the requested change?" These signals must be persistent and comparable across runs so trends can be identified.

### Decision

We will add a first-class `evals` property to the workflow frontmatter that accepts a list of binary YES/NO evaluation questions (BinEval style). The compiler will parse this configuration into a typed `EvalsConfig` struct, attach it to `WorkflowData`, and inject a dedicated `evals` job positioned after `safe_outputs` and before the conclusion job. Evaluation results will be stored as JSONL artifacts and persisted to a dedicated `evals/<workflow-id>` git branch for historical comparison.

### Alternatives Considered

#### Alternative 1: Inline results in the existing safe_outputs artifact

Rather than a separate `evals` job and dedicated JSONL artifact, evaluation metadata could be appended to the `safe_outputs` artifact or as a step summary within the existing conclusion job. This would avoid a new job in the pipeline graph. It was not chosen because it couples evaluation output to the safe_outputs lifecycle, complicates artifact schema versioning, and makes it harder to persist evaluations independently on a dedicated git branch without touching safe_outputs infrastructure.

#### Alternative 2: Free-form LLM-as-judge rubric scoring instead of binary questions

A richer evaluation format (multi-dimensional scores, rubrics, or structured rationale) could provide more nuanced feedback than YES/NO answers. This was not chosen because binary questions have a well-defined failure mode (any NO is an actionable regression), produce machine-readable output trivially, and require a simpler prompt and response-parsing strategy — reducing the chance of ambiguous LLM output and lowering per-evaluation cost when using a small model.

### Consequences

#### Positive
- Enables automated, run-by-run quality tracking with results stored as JSONL for downstream analysis and dashboarding.
- Per-question model override (`model` field on each `EvalDefinition`) allows cost optimization — cheap models for simple factual checks, more capable models for nuanced questions.
- Both shorthand (plain list) and extended (object with `questions`, `model`, `runs-on`) forms are supported, keeping simple cases simple.

#### Negative
- The `buildEvalsJob` implementation is a stub returning `nil` (no-op). Users who configure `evals` today see an experimental warning but no job is compiled or executed — the feature is partially shipped.
- Adding `evals` as a reserved job name and new pipeline slot increases the job ordering complexity (`ensureConclusionIsLastJob` must account for the new tail dependency once the implementation lands).

#### Neutral
- The `evals` frontmatter field is typed as `any` in `FrontmatterConfig` to support both list and object forms; the strongly-typed `*EvalsConfig` lives on `WorkflowData` after parsing, following the same pattern as other configuration fields.
- `EvalsBranchPrefix` (`"evals"`) and `EvalsArtifactName` (`"evals"`) are added as constants, reserving these names in the built-in job namespace.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
