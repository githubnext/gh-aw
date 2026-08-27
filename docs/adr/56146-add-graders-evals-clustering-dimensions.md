# ADR-56146: Add Graders and Evals Clustering Dimensions to Cross-Run Audit

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw audit` cross-run command clusters runs along multiple dimensions (conclusion, task domain, execution style, resource profile) to support cohort analysis across audit sessions. However, grader outcomes and eval availability were not included as clustering dimensions, making it impossible to identify patterns like "all runs with grader failures" or "runs with evals present vs absent." This gap meant multi-run cohort analysis was incomplete for evaluating AI quality signals. The grader summary state is already extracted per-run via `extractGradersData`, and eval presence is detectable via `runHasEvals`, so the necessary data sources exist—only the dimension wiring was missing.

### Decision

We will add `graders` and `evals` as first-class clustering dimensions in `buildRunClusters`, deriving each value from the run's log path at `crossRunInput` construction time. Grader cluster values label homogeneous outcomes by their shared status (`pass`, `fail`, `error`, `unavailable`), use `mixed` for any run whose graders report more than one status, and `absent` when no grader data exists; eval cluster values are binary (present/absent). We will also extract the inline cluster-building code from `buildClusterAnalysis` into a dedicated `buildRunClusters` function so that adding new dimensions requires only a single localized change.

### Alternatives Considered

#### Alternative 1: Store Raw Grader/Eval Counts and Cluster Post-Query

Surface the raw `graders.Passed`, `graders.Failed`, etc. counts on `crossRunInput` without mapping them to a cluster key, leaving callers to define grouping logic at query or display time.

This was not chosen because the existing clustering pipeline is keyed on string dimension values; all other dimensions already normalize their data to a string key at input construction time. Diverging from this pattern would require special-casing the display layer for grader-specific aggregation and would not benefit from the existing trivial-dimension filter that drops single-value dimensions automatically.

#### Alternative 2: Implement Grader/Eval Clustering as a Separate Post-Processing Pipeline

Compute grader and eval clusters as a secondary pass after the main `buildClusterAnalysis` call, rather than integrating them into `crossRunInput`.

This was not chosen because it would duplicate the cluster-building infrastructure (`buildDimensionClusters`, trivial-filter logic) and would complicate the `ClusterAnalysis` assembly path without providing any benefit. All existing dimensions are derived inline from the same data sources, so consistency favors a single pipeline.

### Consequences

#### Positive
- Multi-run cohort reports now group by grader outcome and eval presence, enabling targeted debugging of quality regressions across sessions.
- Extracting `buildRunClusters` reduces the coupling between `buildClusterAnalysis` (which owns filtering and reporting) and the dimension enumeration; adding a new dimension now requires only a single function change.
- The homogeneous/`mixed` grader mapping keeps each cluster unambiguous: a cluster label describes every grader in the run, and heterogeneous runs are grouped together instead of being attributed to their most severe status.

#### Negative
- `deriveGradersClusterValue` and `deriveEvalsClusterValue` perform filesystem reads (via `extractGradersData` and `runHasEvals`) during `processedRunsToCrossRunInputs`, adding I/O per run at cross-run input construction time.
- All heterogeneous runs collapse into the single `mixed` key, so the specific combination of grader statuses (for example, mostly-passing versus mostly-failing) is not distinguishable from the cluster label alone.

#### Neutral
- The existing trivial-dimension filter means that if all runs in a cohort share the same grader outcome or all lack evals, those dimensions are dropped from the output, consistent with existing behavior for other dimensions.
- Test coverage is added as a targeted unit test rather than an integration test, which is consistent with the existing test suite structure in this package.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
