# ADR-45445: Link Experiment Success Metrics to Declared Evals

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: Unknown

---

### Context

The agentic workflow framework supports two orthogonal features: `experiments` (A/B variant testing) and `evals` (LLM-judge question batteries). Prior to this change, an experiment's `metric` field accepted only free-form metric names (e.g., `aic`), with no awareness of `evals` definitions. This meant an author could write `metric: eval:focused` in their frontmatter, but the compiler would silently accept it even if `focused` was not declared under `evals:`, or if no `evals` block existed at all. The gap allowed misconfigured eval-backed experiments to compile successfully and fail only at runtime — or produce silently incorrect results.

### Decision

We will extend the experiment `metric` field to support eval-backed references via two canonical forms (`eval:<id>` and `evals.<id>`, with an optional suffix `evals.<id>.<suffix>` reserved for future derived metrics) and enforce at compile time that any eval-referencing metric points to a declared `evals` question ID. The compiler returns targeted errors for a missing ID token, an unknown eval ID, or an eval-referencing metric when no `evals` block is declared.

### Alternatives Considered

#### Alternative 1: Keep metrics as unvalidated free-form strings

The `metric` field would remain a plain string with no special parsing. Authors who want eval-backed metrics would document a convention informally. This approach is simpler and requires no compiler changes, but it allows silent misconfiguration: a typo in an eval ID (`eval:focussed`) compiles cleanly and only surfaces as a missing or wrong metric at analysis time, long after runs have been collected.

#### Alternative 2: Auto-infer the eval metric when a single eval is declared

If exactly one eval question is declared and the experiment has no explicit `metric`, the compiler could automatically treat that eval as the success metric. This would reduce frontmatter verbosity for simple cases, but it introduces implicit coupling that is hard to reason about when multiple evals are declared, and it hides the decision in a convention rather than making it explicit in the workflow file.

#### Alternative 3: Introduce a separate `eval_metric` field on experiments

A new top-level `eval_metric` key on the experiment object would hold the eval reference, keeping `metric` strictly for non-eval metric names. This is explicit but adds a second field that partially overlaps `metric` in purpose, complicates schema documentation, and requires authors to learn two fields where one would suffice.

### Consequences

#### Positive
- Compile-time validation catches unknown or malformed eval references before any experiment runs are collected, reducing silent data-quality failures.
- The `eval:<id>` / `evals.<id>` syntax is first-class in the schema and spec documentation, making the experiment–eval linkage discoverable and self-documenting.
- The `evals.<id>.<suffix>` form is reserved and parsed correctly today, leaving a clear extension point for derived metrics (e.g., `yes_rate`) without a future breaking change.

#### Negative
- The `metric` field now carries dual semantics (free-form name vs. structured eval reference), requiring a parsing step that did not exist before; authors must understand the `eval:` / `evals.` prefixes to use eval-backed metrics.
- Existing experiment configs that inadvertently used an `eval:`-prefixed metric string as a literal metric name will now trigger a validation error and need to be updated.

#### Neutral
- The validation function (`validateExperimentMetricReferences`) is wired into `extractAdditionalConfigurations` after eval parsing, so it has access to the fully-resolved `EvalsConfig`; this ordering is a structural constraint future callers must preserve.
- Experiments that use non-eval metrics (`metric: aic`) are entirely unaffected — the parser returns early on strings that do not match either prefix.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
