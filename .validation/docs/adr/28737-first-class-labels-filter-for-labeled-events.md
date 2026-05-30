# ADR-28737: First-Class `on.labels` Filter for Label-Triggered Workflow Events

**Date**: 2026-04-27
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

GitHub Actions does not provide a native label-name filter for events such as `pull_request_target` with `types: [labeled]`. Workflows that needed to respond only to specific labels had no clean mechanism — the only available workaround was to include an `exit 1` guard inside a workflow step. This caused every unrelated label-add event to show as a red ❌ failed run on CI dashboards rather than a clean gray ⊘ skip, degrading signal quality for teams monitoring pull request activity. The gh-aw compiler already provides analogous filters for contributor roles (`on.roles`) and bot identifiers (`on.bots`), establishing a precedent for injecting GitHub Actions `if:` expressions from frontmatter fields.

### Decision

We will add a first-class `on.labels` field to the gh-aw workflow frontmatter. When present, the compiler injects a job-level `if:` condition on the `pre_activation` job that skips the entire job when the triggering label does not match any of the listed names. Events that carry no label data (e.g., `workflow_dispatch`, `push`, `schedule`) are always allowed through via a `github.event.label.name == ''` guard, so non-labeled triggers are not inadvertently blocked. The field mirrors the existing `roles` and `bots` filter shape, accepting either a single string or an array. A `trigger_label` field is also added to the `aw_context` object so AI agents can read the triggering label name directly from their context payload.

### Alternatives Considered

#### Alternative 1: Step-level `exit 1` guard

Workflow authors could add an explicit shell guard (e.g., `if [[ "${{ github.event.label.name }}" != "panel-review" ]]; then exit 1; fi`) inside the first pre-activation step. This was the de-facto workaround before this ADR. It was rejected because `exit 1` marks the job as **failed** (red ❌) rather than **skipped** (gray ⊘), adding persistent noise to CI dashboards and causing confusion when authors see failures on label events they deliberately did not intend to handle.

#### Alternative 2: Step-level `if:` conditions injected on each generated step

The compiler could inject a step-level `if:` expression on every generated step rather than a single job-level condition. This was rejected because it produces a more complex compiled output, still allows the job header to show as running in the GitHub UI (not a clean skip), and does not achieve the gray ⊘ appearance that a job-level `if:` provides.

#### Alternative 3: Native GitHub Actions event filtering

GitHub Actions supports filtering by branch name or file path at the event trigger level but does not support filtering by label name. There is no native `on.pull_request_target.labels` equivalent. This alternative is not viable and was not seriously considered.

### Consequences

#### Positive
- Unmatched label events now appear as ⊘ Skipped rather than ❌ Failed, eliminating CI dashboard noise on repositories that use many labels.
- The implementation follows the established `roles`/`bots` compiler pattern, keeping the frontmatter API and internal compiler code consistent and predictable.
- The `trigger_label` field in `aw_context` gives AI agents access to the triggering label name without requiring payload inspection.

#### Negative
- The `on.labels` field is a gh-aw-specific frontmatter extension with no GitHub Actions native counterpart; users reading raw YAML may expect native behavior.
- The `github.event.label.name == ''` pass-through guard is non-obvious in compiled output; readers may not immediately understand why non-labeled events are unconditionally allowed through.

#### Neutral
- The `hasSafeEventsOnly()` event-counting function must explicitly exclude `labels` from its loop, mirroring the existing exclusions for `roles`, `bots`, `command`, `stop-after`, and `reaction`.
- The JSON schema (`main_workflow_schema.json`) is updated to reflect `on.labels` as a `oneOf` string-or-array field, aligning static validation with runtime behavior.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Label Filter Field

1. The `on.labels` frontmatter field **MUST** accept either a single non-empty string or a non-empty array of non-empty strings.
2. Each label name value **MUST NOT** be an empty string.
3. The `on.labels` array **MUST NOT** contain more than 50 entries.
4. When `on.labels` is absent, the compiler **MUST NOT** inject any label-based `if:` condition into the compiled output.

### Compiled Output

1. When `on.labels` is set, the compiler **MUST** inject a job-level `if:` condition on the `pre_activation` job.
2. The injected condition **MUST** evaluate to true when `github.event.label.name` is an empty string, passing through events that carry no label payload (e.g., `workflow_dispatch`, `push`, `schedule`).
3. The injected condition **MUST** evaluate to true when `github.event.label.name` equals any of the label names specified in `on.labels`, using strict string equality (`==`).
4. The injected condition **MUST NOT** use case-insensitive matching; label names **MUST** be matched exactly as specified in the frontmatter.
5. When `on.labels` is combined with an existing job-level `if:` condition (e.g., from a top-level `if:` field), the compiler **MUST** combine both conditions using logical AND (`&&`), with the label condition as the first operand.

### Event Counting

1. The `labels` key under `on:` **MUST** be excluded from the event-type count computed by `hasSafeEventsOnly()`, consistent with the treatment of `roles`, `bots`, `command`, `stop-after`, and `reaction`.

### Agent Context

1. `buildAwContext()` **MUST** include a `trigger_label` field in the returned context object.
2. `trigger_label` **MUST** be set to `context.payload?.label?.name` when a label payload is present, and **MUST** default to an empty string (`""`) for events without label data.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25006216146) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
