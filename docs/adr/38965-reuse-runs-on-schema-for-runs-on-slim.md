# ADR-38965: Reuse the runs-on schema and rendering pipeline for runs-on-slim

**Date**: 2026-06-13
**Status**: Accepted

## Context

`runs-on-slim` selects the runner for all framework/generated jobs (activation, safe-outputs, unlock, APM, etc.), while `runs-on` selects the runner for the main agent job. `runs-on` already accepts the full set of GitHub Actions runner forms — a plain string label, an array of labels, or a `{ group, labels }` runner-group object — but `runs-on-slim` was schema-validated and parsed as a string only. Self-hosted users who select runners by label array or runner group therefore could not route framework jobs to the same runner they use for `runs-on`, and any such value failed to compile. The two fields configure the same concept (runner selection) and should accept the same syntax.

## Decision

We will treat `runs-on-slim` as the same kind of value as `runs-on` rather than as a distinct string field. Concretely: the JSON schema entry for `runs-on-slim` now `$ref`s the shared `#/$defs/github_actions_runs_on` definition; the in-memory `FrontmatterConfig.RunsOnSlim` field changes from `string` to `any` and is validated through the existing `validateRunsOnValue`; and `WorkflowData.RunsOnSlim` holds a **rendered `runs-on:` YAML snippet** (produced by the same extraction path as `runs-on`) instead of a bare label. Downstream consumers re-indent that snippet for the framework job context via helpers (`formatRunsOnSnippetForInlineValue`, `indentYAMLLines`). This guarantees parity with `runs-on` and avoids a second, drift-prone validation/rendering path.

## Alternatives Considered

### Alternative 1: Keep `runs-on-slim` as a string and document the limitation
Leave the type as `string` and tell users that `runs-on-slim` cannot mirror an array or runner-group value. Rejected because it permanently blocks legitimate self-hosted configurations and forces an inconsistent mental model where two runner-selection fields accept different syntax.

### Alternative 2: Add a separate schema and parser for `runs-on-slim`'s array/object forms
Duplicate the array/runner-group validation and YAML-rendering logic specifically for `runs-on-slim`. Rejected because it duplicates non-trivial logic already maintained for `runs-on`, inviting divergence over time as one path gains features or fixes the other misses.

## Consequences

### Positive
- `runs-on-slim` reaches full parity with `runs-on`, accepting string, label-array, and `{ group, labels }` forms.
- Validation and rendering reuse the existing shared `runs-on` schema and code paths, so future changes apply to both fields automatically.
- Self-hosted setups that select runners by label array or runner group can now route framework jobs correctly.

### Negative
- The internal contract of `RunsOnSlim` changes: `FrontmatterConfig.RunsOnSlim` becomes `any` and `WorkflowData.RunsOnSlim` now carries a rendered `runs-on:` snippet rather than a bare label, requiring every consumer (serialization, framework-job formatting, central slash-command resolution) to be updated and re-tested.
- Indentation handling adds complexity: snippets must be re-indented for differing YAML contexts, introducing helper functions whose correctness depends on the exact upstream rendering format.

### Neutral
- Existing tests were updated to expect rendered `runs-on:` snippets instead of bare labels, and new tests cover the array and runner-group forms.
- Reference docs, self-hosted-runner guidance, workflow constraints, and editor autocomplete metadata were updated to describe the expanded syntax.

---
