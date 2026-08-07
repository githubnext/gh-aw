# ADR-51154: Split AWF Helpers into Focused Single-Responsibility Modules

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/awf_helpers.go` had grown into a monolithic file mixing several unrelated concerns: AWF command and argument assembly, environment variable exclusion, ARC/DinD path rewriting and image digest helpers, and AWF version capability gates. As new features were added to each area, the file became difficult to navigate, review, and test independently. The existing public API (`BuildAWFCommand`, `BuildAWFArgs`, `ComputeAWFExcludeEnvVarNames`, etc.) remained stable, but the implementation had become a maintenance liability.

### Decision

We will split `awf_helpers.go` into four focused modules within the same `workflow` package, grouped by responsibility: command and argument assembly (`awf_command_builder.go`), environment variable filtering and max-AI-credits helpers (`awf_env.go`), ARC/DinD path rewriting and container digest helpers (`awf_arc_dind.go`), and AWF version capability gates (`awf_feature_flags.go`). The residual `awf_helpers.go` retains shared constants, the config type, and small scaffolding that is referenced by all modules. The public API is unchanged.

### Alternatives Considered

#### Alternative 1: Keep Everything in a Single File

Retain `awf_helpers.go` as-is and instead apply in-file region comments (e.g., `// --- ARC/DinD ---`) to separate concerns visually. This requires zero structural change and has no merge-conflict risk.

Not chosen because the file was already ~900 lines and growing. Region comments do not enforce boundaries, do not help IDEs surface individual concerns, and would still require reviewers to parse the entire file. The problem recurs as each area continues to grow.

#### Alternative 2: Extract Each Concern into Its Own Sub-Package

Move each concern into a child package under `pkg/workflow/` (e.g., `pkg/workflow/awfcmd`, `pkg/workflow/awfenv`). This is the strongest form of separation and enables independent import graphs.

Not chosen because it requires changing all call sites to use the new package paths, breaks the existing `workflow`-internal function references (ARC/DinD helpers use unexported helpers shared with the command builder), and introduces an import cycle risk given the circular cross-references between concerns. The split-within-same-package approach achieves improved navigability at lower refactoring cost.

### Consequences

#### Positive
- Each module is independently navigable and can be reviewed or tested in isolation.
- Future additions to a specific concern have a clear, well-scoped home, reducing scope creep into unrelated files.
- Test files for individual concerns (e.g., env helper edge cases) can target a single module without touching the others.
- No caller changes required — the public API is preserved exactly.

#### Negative
- More files in a single package means contributors must remember which file contains which function; Go's intra-package visibility means there is no enforced boundary.
- The split does not prevent future drift back toward a monolith if new helpers are placed in `awf_helpers.go` without discipline.

#### Neutral
- Existing tests that exercise the public API continue to work without modification, since function signatures and package paths are unchanged.
- The `awf_helpers.go` residual file retains shared constants and types; this file will still grow if new cross-cutting constants are added.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
