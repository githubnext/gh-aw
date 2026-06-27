# ADR-41829: Consolidate `sliceutil.SortedKeys` as Canonical Map-Key Utility and Redistribute Misplaced Helpers

**Date**: 2026-06-27
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

### Context

The codebase had accumulated a widespread manual pattern for collecting and sorting map keys: `make → append → sort.Strings`. This idiom was duplicated at 40+ sites across `pkg/cli`, `pkg/workflow`, and `pkg/parser`, even though `sliceutil.SortedKeys` already existed and served this exact purpose. Separately, several general-purpose helper functions had drifted into files where they did not belong semantically — for example, `isDescendant` (a generic YAML-indent helper) lived in `codemod_github_app.go`, and safe-output helpers accumulated in `notify_comment.go`. This created maintenance friction: new contributors reinvented the utility, and unrelated helpers were hard to discover in their misplaced locations.

### Decision

We will adopt `sliceutil.SortedKeys` as the single canonical way to collect and sort map keys throughout the codebase, replacing all manual `make → append → sort.Strings` occurrences. Concurrently, we will enforce module cohesion by relocating misplaced helper functions to files whose names reflect their domain — generic YAML helpers to `yaml_frontmatter_utils.go`, engine-to-API-host logic to `engine_api_targets.go`, and safe-output env-var construction to `safe_outputs_env.go`. The duplicate `toEnvVarCase` function is removed in favour of the existing `normalizeJobNameForEnvVar`, which covers the same cases.

### Alternatives Considered

#### Alternative 1: Keep the manual `make → append → sort.Strings` idiom

Every callsite continues to inline the three-step collect-then-sort pattern. This requires no refactor effort and avoids touching 59 files. Rejected because the duplication had already reached 40+ sites and was growing; new callsites were consistently reinventing the pattern rather than discovering the existing utility, confirming the drift would continue unchecked.

#### Alternative 2: Use stdlib `maps.Keys` + `slices.Sort` (Go 1.21+)

Replace the manual idiom with `maps.Keys(m)` followed by `slices.Sort(...)`. This is idiomatic stdlib and requires no project-specific utility. Rejected because `sliceutil.SortedKeys` is already established in the project and composes both steps in one call, reducing boilerplate to a single expression. Migrating to a new stdlib pair would only exchange one two-line idiom for another without adding the readability gain of the single-call form; it would also invalidate any existing usages of `sliceutil.SortedKeys` elsewhere.

#### Alternative 3: Address only the deduplication fix in `cache_integrity.go`; leave the sweep for a later PR

A narrow fix to the inconsistency in `canonicalReposScope` (which hand-rolled dedup while `sliceutil.Deduplicate` was called on the same file) could be merged independently. Rejected because the broader sweep was machine-generated and low-risk; deferring it would leave the 40+ duplicated sites in place with no clear owner to address them later.

### Consequences

#### Positive
- 725 lines deleted, 302 added — net −423 lines of boilerplate across 66 files, improving signal-to-noise ratio in the codebase.
- A single canonical pattern for sorted map iteration eliminates the "which way should I do this?" question for contributors.
- Misplaced helpers are now co-located with semantically related code, making them discoverable and reducing surprise when reading individual files.
- Removing the `toEnvVarCase` duplicate eliminates a potential divergence point if the normalization logic ever needs to change.

#### Negative
- `sliceutil` is now a transitive import dependency for many more packages; any future breaking change to `SortedKeys` (signature, behaviour, or package path) requires updating all 59+ files.
- The sweep is mechanical and broad; reviewers must verify that no callsite had intentional non-lexicographic ordering or special nil-vs-empty semantics that `SortedKeys` does not preserve (e.g., `update_container_pins.go` adds an explicit `nil → []string{}` guard after the replacement).

#### Neutral
- The `import` block in many files changes from `"sort"` to `"github.com/github/gh-aw/pkg/sliceutil"`; tooling (goimports, linters) must be aware of the project-internal package path.
- Function relocation changes which file a `git blame` points to for `isDescendant`, `isExpressionValue`, `getEngineAPIHosts`, `buildSafeOutputJobsEnvVars`, and `systemSafeOutputJobNames` — history is preserved via `git log --follow`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
