# ADR-44202: Detection Job Needs Derived from Engine Env Expressions

**Date**: 2026-07-08
**Status**: Draft
**Deciders**: pelikhan, Copilot SWE Agent

---

### Context

The generated `detection` job inherits environment variables from the engine configuration used for threat detection. Some workflows set those values with expressions such as `${{ needs.select_model.outputs.model }}`, where the expression is only valid if the detection job directly depends on the referenced custom job.

Before this change, the detection job always used `needs: [agent, activation]` and did not scan its effective engine environment for `needs.<job>.outputs.*` references. As a result, expressions that resolved correctly in the `agent` and `conclusion` jobs silently evaluated to empty strings in the detection job at runtime. The detection job also supports a `safe-outputs.threat-detection.engine` override, so dependency inference must follow the effective detection engine env rather than always reading the top-level engine config.

### Decision

`buildDetectionJob` will derive additional direct dependencies from the effective detection engine env. The compiler starts with the required built-in dependencies (`agent` and `activation`), scans the env values for `needs.<customJob>.outputs.*` references, appends the referenced custom jobs, and deduplicates the resulting `needs` list.

When `safe-outputs.threat-detection.engine.env` is present, it overrides the top-level `engine.env` for this analysis, matching the execution path already used by the detection job. If the env references a built-in job that cannot become a direct detection dependency, the compiler emits the same warning pattern already used by the main agent job so the workflow author sees the misconfiguration at compile time.

### Alternatives Considered

#### Alternative 1: Leave the detection job dependency list unchanged

We could keep the detection job limited to `agent` and `activation` and document that `engine.env` `needs.*` expressions are unsupported there. This was rejected because the detection job already inherits those env values, so leaving the dependency gap in place would preserve a silent runtime failure mode for otherwise valid workflow configurations.

#### Alternative 2: Reuse the top-level engine env scan without honoring the detection override

We could mirror the existing main-job logic by scanning only `data.EngineConfig.Env`. This was rejected because it would compute dependencies from the wrong configuration whenever `safe-outputs.threat-detection.engine.env` overrides the top-level env, which can both miss required jobs and add unnecessary ones.

### Consequences

#### Positive

- Detection-job env expressions that reference custom job outputs now resolve correctly at runtime.
- Detection-job dependency inference now matches the effective threat-detection engine configuration instead of a possibly stale top-level env.
- Built-in-job references in detection env expressions now surface as compile-time warnings instead of failing silently.

#### Negative

- The detection job may wait for additional custom jobs before starting, which can lengthen workflow execution when authors reference more upstream outputs.
- Workflows that accidentally relied on the previous empty-string behavior will now observe the intended dependency ordering and warning output.

#### Neutral

- The change is limited to detection-job dependency inference and warning behavior; it does not change how agent output artifacts or detection steps are executed once the job starts.
- Regression coverage now includes override, deduplication, built-in warning, and compiled-lock assertions for the detection job path.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
