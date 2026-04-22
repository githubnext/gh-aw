# ADR-27750: Step-Boundary Detection for Pre-Steps YAML Insertion

**Date**: 2026-04-22
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler inserts user-defined `pre-steps` into built-in jobs (`pre_activation`, `activation`) using `insertPreStepsAfterSetupBeforeCheckout` in `pkg/workflow/compiler_jobs.go`. Internally, each workflow step is represented as a slice of raw YAML strings, where a single logical step (e.g. the `Setup Scripts` action) can span multiple consecutive slice entries. The prior insertion logic used `lastSetupIdx + 1` as the insertion index, which targets the element immediately following the line that contains `id: setup` — but that element may still be part of the same multi-line step block. Inserting there produces malformed YAML with duplicate `uses` and `with` keys, which causes downstream CI failures and is difficult to debug because the file appears syntactically unusual rather than obviously broken.

### Decision

We will determine the pre-steps insertion point by scanning forward from the last setup-related line to the next YAML step-item boundary — identified as a line that, after trimming leading whitespace, begins with `- `. This boundary reliably marks the start of a new step block regardless of how many lines the preceding step occupies. If no such boundary is found, insertion falls back to appending at the end of the steps slice.

### Alternatives Considered

#### Alternative 1: Parse a Structured YAML AST

Use a YAML parser (e.g. `go.yaml.in/yaml/v3`) to build a typed step representation, then insert pre-steps at the correct position within the AST before serializing back to YAML. This eliminates ambiguity about step boundaries because the parser understands block structure natively.

This was not chosen for this fix because it would require replacing the `[]string` step representation throughout the compiler — a significantly larger refactor with a wider blast radius than a targeted bug fix warrants. The structured-AST approach is the better long-term architecture and should be revisited as a separate initiative (see Negative consequences).

#### Alternative 2: Track Step Boundaries as Metadata at Compilation Time

When emitting YAML lines into the steps slice, also record the index of the first line of each step in a parallel metadata slice. The insertion function would then consult this index rather than scanning for `- ` prefixes.

This was not chosen because it requires changes to every code path that writes to the steps slice and introduces a metadata-sync invariant that is easy to violate silently. The scanning approach is self-contained and requires no changes to callers.

### Consequences

#### Positive
- Pre-steps are now inserted at a true step boundary, eliminating the duplicate-key YAML corruption.
- The fix is self-contained within `insertPreStepsAfterSetupBeforeCheckout`; no caller changes are required.
- Regression tests cover both `uses:`-style and `run:`-style pre-steps for `pre_activation` and `activation` jobs, and assert that the generated lock file is valid YAML.

#### Negative
- String scanning for YAML structure is inherently fragile: if a future step body contains a line that starts with `- ` at the expected indentation (e.g. a multi-line `run:` script using YAML block scalars), the scanner could misidentify it as a step boundary and insert pre-steps too early.
- The root cause — representing steps as `[]string` of raw YAML rather than typed objects — is not addressed. The fix is a targeted patch on top of a data model that makes step-boundary detection fundamentally error-prone. A structured-AST refactor remains necessary to fully resolve this class of issue.

#### Neutral
- The fallback behavior (append at end when no boundary is found) is now explicitly logged via `compilerJobsLog.Print`, improving observability for edge cases.
- The `go.yaml.in/yaml/v3` package is added as a test dependency to validate that generated lock files parse successfully.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Pre-Steps Insertion

1. Implementations **MUST** determine the pre-steps insertion index by scanning forward from the last setup-step line to find the first subsequent line whose content, after trimming leading whitespace, starts with `- `.
2. Implementations **MUST NOT** use a fixed `lastSetupIdx + 1` offset as the insertion index when the steps slice may contain multi-line step representations.
3. Implementations **MUST** fall back to appending pre-steps at the end of the steps slice when no step-item boundary is found after the last setup line.
4. Implementations **SHOULD** emit a log message when the fallback append path is taken, to aid debugging.

### Generated YAML Validity

1. Implementations **MUST** produce a lock file whose `jobs` section is valid YAML after pre-steps insertion.
2. Implementations **MUST NOT** emit a `jobs` step block that contains duplicate YAML keys (e.g. duplicate `uses` or `with` fields) as a result of mid-block insertion.

### Regression Testing

1. New or modified implementations of `insertPreStepsAfterSetupBeforeCheckout` **MUST** include tests that assert correct ordering: pre-steps appear after the complete setup step body and before the first downstream built-in step.
2. Tests **SHOULD** validate that the generated lock file parses successfully as YAML (not merely that it contains the expected strings).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24759852068) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
