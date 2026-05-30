# ADR-34144: Cluster Config Preprocessing Helpers and Co-locate Workflow Helper Hotspots

**Date**: 2026-05-23
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

A semantic clustering audit of `pkg/workflow` surfaced several organizational hotspots: three near-identical `addSafeOutput*GitHubTokenForConfig` methods on `Compiler` duplicated branching logic for GitHub App token resolution; generic raw-config normalizers (`preprocessBoolFieldAsString`, `preprocessIntFieldAsString`, `preprocessStringArrayFieldAsTemplatable`, `preprocessProtectedFilesField`, `preprocessExpiresField`) were scattered across `templatables.go`, `parse_helpers.go`, and `config_helpers.go`; the structural `runs-on` shape validator `validateRunsOnValue` lived in `frontmatter_parsing.go` rather than next to `validateRunsOn` in `runs_on_validation.go`; and `git_helpers_wasm.go` retained `RunGit` and `GetCurrentGitTag` stubs whose non-WASM counterparts no longer exist. Two call sites also shadowed the validator name `validateRunsOnValue` with a local variable, creating a small but real readability hazard. ADR-29336 established that `preprocessProtectedFilesField` belongs in `parse_helpers.go`, so any move away from that file is an explicit revision of that prior decision rather than an accidental drift.

### Decision

We will (1) extract a shared `addResolvedSafeOutputGitHubTokenForConfig` resolver and re-express the three per-target token methods as thin wrappers around it, preserving the behavioral split that only standard and Copilot handlers may use GitHub App tokens while assign-to-agent bypasses them; (2) introduce `pkg/workflow/config_preprocessing.go` as the single home for generic raw-config normalization helpers, superseding the relevant clause of ADR-29336 for `preprocessProtectedFilesField`; (3) move `validateRunsOnValue` into `runs_on_validation.go` so structural and semantic `runs-on` validation live together and rename the two shadowing local variables to `formattedRunsOn`; (4) delete the unused `RunGit` and `GetCurrentGitTag` WASM stubs and the matching stale doc references. The driving principle is the file-naming convention established in ADR-27325 and reinforced in ADR-28282/29336: a file's name should accurately communicate the semantic domain of every function inside it, and duplicate logic should be collapsed when the behavioral variation can be expressed as a parameter.

### Alternatives Considered

#### Alternative 1: Keep all preprocessors in `parse_helpers.go` (status quo per ADR-29336)

Leave generic preprocessors where ADR-29336 placed them and only consolidate the safe-output token resolvers. This was rejected because `parse_helpers.go` had already accumulated a mix of pure coercion helpers (`coerceStringOrArrayField`, `parseStringSliceAny`) and stateful preprocessors that mutate `configData` in place — two distinguishable responsibilities. Splitting them gives each file a single clear charter: `parse_helpers.go` for coercion, `config_preprocessing.go` for normalization-with-side-effects. The cost of revising ADR-29336 is low because the prior decision was itself a relocation pass and inherits a "follow naming intent" rationale that this PR extends.

#### Alternative 2: Pass a flag instead of a resolver function

Replace the three safe-output token methods with a single method that accepts a token-target enum (`safeOutput`, `copilot`, `agent`) and switches internally. This was rejected because it pushes the per-target token resolver dispatch into the consolidated function, recreating the branching that the refactor is trying to eliminate. Passing the resolver function directly keeps `addResolvedSafeOutputGitHubTokenForConfig` agnostic of which token target it serves and lets each wrapper own its own resolver choice.

#### Alternative 3: Leave the WASM stubs in place as future-proofing

Keep `RunGit` and `GetCurrentGitTag` as WASM-only stubs in case future code paths need them. This was rejected because their non-WASM counterparts have already been removed, so the stubs are guaranteed-unreachable code that misleads readers about which Git surface area is supported. Reintroducing them later if needed is a one-line change.

### Consequences

#### Positive
- The three safe-output token methods now share one resolver path, eliminating ~60 lines of duplicated branching and centralizing the GitHub App token precedence rule.
- All generic raw-config preprocessors are co-located in `config_preprocessing.go`, making it obvious where to add a new templatable-or-literal preprocessor.
- `validateRunsOnValue` is next to `validateRunsOn` in `runs_on_validation.go`, and a focused unit test (`TestValidateRunsOnValue`) now covers its shape-validation branches directly.
- Removing the dead WASM stubs makes the WASM build surface match the non-WASM one — readers see only the Git operations actually available.
- Renaming local `validateRunsOnValue` variables to `formattedRunsOn` removes the function-vs-variable shadowing hazard that made the two call sites harder to grep.

#### Negative
- Revises ADR-29336's normative requirement that `preprocessProtectedFilesField` reside in `parse_helpers.go`. Future contributors must consult this ADR alongside 29336 to know the current placement rule.
- Adds one new file (`config_preprocessing.go`) to `pkg/workflow/`, marginally increasing file count in a directory that is already large.
- The shared `addResolvedSafeOutputGitHubTokenForConfig` introduces an `allowGitHubApp bool` parameter; readers must trace into the function to see which wrappers pass which value, where previously the behavior was visible at each call site.

#### Neutral
- No public API surface changes; all moved functions retain their signatures and package-level visibility.
- `git blame` for the moved functions will surface this PR rather than original authorship without `--follow`.
- Header comments in `templatables.go`, `parse_helpers.go`, and `validation_helpers.go` are updated to point readers at `config_preprocessing.go`; this is documentation-only churn.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Config Preprocessing Helpers (`pkg/workflow`)

1. The following functions **MUST** reside in `pkg/workflow/config_preprocessing.go`:
   - `preprocessBoolFieldAsString`
   - `preprocessIntFieldAsString`
   - `preprocessStringArrayFieldAsTemplatable`
   - `preprocessProtectedFilesField`
   - `preprocessExpiresField`
2. New generic raw-config normalization helpers that mutate a `map[string]any` in place before YAML unmarshaling **MUST** be added to `config_preprocessing.go` and **MUST NOT** be placed in `parse_helpers.go`, `templatables.go`, `config_helpers.go`, or `validation_helpers.go`.
3. `parse_helpers.go` **MUST NOT** contain stateful preprocessors that mutate `configData`; its scope **MUST** be limited to pure coercion helpers that return new values without side effects.
4. `templatables.go` **MUST NOT** contain raw-config preprocessors; its scope **MUST** be limited to templatable field types and their YAML-emission helpers.
5. Schedule-specific preprocessing **MAY** remain in its existing file when it depends on schedule-specific types or context that would not naturally fit a generic preprocessor file.
6. This section supersedes the clause in [ADR-29336](29336-relocate-misplaced-functions-to-semantic-homes.md) that placed `preprocessProtectedFilesField` in `parse_helpers.go`.

### Safe-Output GitHub Token Resolution (`pkg/workflow`)

1. The three per-target safe-output token methods (`addSafeOutputGitHubTokenForConfig`, `addSafeOutputCopilotGitHubTokenForConfig`, `addSafeOutputAgentGitHubTokenForConfig`) **MUST** delegate to a single shared resolver function rather than each duplicate the GitHub App token precedence logic.
2. The shared resolver **MUST** accept an `allowGitHubApp` boolean (or equivalent) parameter that gates whether `data.SafeOutputs.GitHubApp` is considered.
3. The assign-to-agent token method **MUST** pass `allowGitHubApp = false` so that GitHub App installation tokens are never used for the Copilot assignment API.
4. New safe-output token resolution paths **SHOULD** be expressed as additional thin wrappers around the shared resolver rather than as new copies of the precedence logic.

### `runs-on` Validation Co-location (`pkg/workflow`)

1. `validateRunsOnValue` **MUST** reside in `pkg/workflow/runs_on_validation.go` alongside `validateRunsOn` and `extractRunnerLabels`.
2. New `runs-on` shape or semantic validators **MUST** be added to `runs_on_validation.go` and **MUST NOT** be placed in `frontmatter_parsing.go`.
3. Local variables holding a formatted `runs-on` string **MUST NOT** be named `validateRunsOnValue` (or any other name that shadows an exported validator); `formattedRunsOn` **SHOULD** be used instead.

### WASM Git Surface (`pkg/workflow`)

1. `git_helpers_wasm.go` **MUST NOT** export Git helper functions whose non-WASM counterparts do not exist.
2. Doc comments in `git_helpers.go` **MUST NOT** reference Git helpers that have been removed.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, adding a new generic preprocessor outside `config_preprocessing.go`, reintroducing duplicated safe-output token resolver branching, placing a `runs-on` validator outside `runs_on_validation.go`, or shipping a WASM Git stub without a matching non-WASM definition — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26319884249) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
