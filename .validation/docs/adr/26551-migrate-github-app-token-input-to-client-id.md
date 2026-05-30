# ADR-26551: Migrate GitHub App Token Input from `app-id` to `client-id` with Backward Compatibility

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `actions/create-github-app-token` action from GitHub has deprecated the `app-id` input field in favour of `client-id`. Compiled agentic workflows that mint GitHub App tokens were emitting upstream deprecation warnings because the code generator always emitted `with.app-id`. At the same time, existing workflow frontmatter authored by users already uses `app-id`, and a hard breaking change would silently break those workflows. The system needed to close the gap between what upstream now expects (`client-id`) and what authors had been writing (`app-id`), without requiring every user to manually update their files.

### Decision

We will migrate all generated token-minting steps to emit `client-id` instead of `app-id` when calling `actions/create-github-app-token`, while simultaneously accepting both field names in frontmatter parsing so that existing workflows continue to work. When both fields are present, `client-id` takes precedence. We will also ship an automated `gh aw fix` codemod (`github-app-app-id-to-client-id`) that rewrites `app-id` to `client-id` within `github-app` blocks, and update the JSON Schema to use `anyOf` with two valid required-field combinations (`client-id + private-key` or `app-id + private-key`). This approach eliminates deprecation noise immediately, preserves backward compatibility, and gives users a path to migrate automatically.

### Alternatives Considered

#### Alternative 1: Hard breaking change — remove `app-id` support immediately

Immediately stop accepting `app-id` in frontmatter and require `client-id` everywhere. This would be the simplest long-term state but would break all existing workflow configurations without warning and require every author to update their files before their workflows would compile. Given that users have no tooling to detect the breakage until compile time, this was rejected as too disruptive.

#### Alternative 2: Emit both `app-id` and `client-id` in generated output

Emit both fields in the generated YAML step to satisfy both old and new versions of `actions/create-github-app-token`. This would silence deprecation warnings on the new action but produce noisy, confusing generated output and could cause unexpected behaviour if the upstream action changes its precedence rules. It was rejected because it couples generated output to undocumented upstream precedence logic.

#### Alternative 3: Accept `app-id` in frontmatter only; always emit `client-id` in compiled output (chosen approach, with codemod)

Accept both field names in parsing, but always emit `client-id` in compiled output. Pair this with a codemod that rewrites frontmatter on demand. This cleanly separates authoring compatibility from compiled output correctness, gives a migration path, and avoids the noise of the hard-cut approach. This is the approach implemented in this PR.

### Consequences

#### Positive
- Deprecation warnings from `actions/create-github-app-token` are eliminated immediately once workflows are compiled or recompiled.
- Existing workflows using `app-id` continue to compile and run without modification.
- Authors can migrate automatically using `gh aw fix`, reducing manual effort.
- The JSON Schema accurately reflects the valid combinations, enabling editor validation for both old and new syntax.

#### Negative
- The internal `GitHubAppConfig.AppID` field now carries ambiguous semantics — it stores whatever the user provided (`client-id` or `app-id`) under a field named `AppID`. This mismatch between the Go field name and the preferred YAML key is mild technical debt.
- The `anyOf` schema constraint is harder to read and may produce less clear validation error messages than a single `required` array.
- Maintaining two accepted field names indefinitely (if `app-id` is never removed) adds ongoing surface area to the parser.

#### Neutral
- The `app-id` field in the JSON Schema is now documented as a deprecated alias, which may prompt questions from authors reading schema documentation.
- All existing tests that asserted `app-id` in compiled output were updated to assert `client-id`; new tests were added for `client-id` parsing paths.
- The codemod is registered in the global codemod registry and will run as part of `gh aw fix` alongside all other codemods.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Compiled Output — Token Minting Steps

1. All token-minting steps generated for `actions/create-github-app-token` **MUST** emit `with.client-id`, not `with.app-id`.
2. Generated workflows **MUST NOT** emit both `with.client-id` and `with.app-id` in the same step.

### Frontmatter Parsing — Field Acceptance

1. The frontmatter parser **MUST** accept `client-id` as the primary GitHub App identifier field within any `github-app` object.
2. The frontmatter parser **MUST** accept `app-id` as a backward-compatible alias for `client-id` within any `github-app` object.
3. When both `client-id` and `app-id` are present in the same `github-app` object, the parser **MUST** use `client-id` and **SHOULD** ignore `app-id`.
4. A `github-app` object **MUST** include `private-key` alongside either `client-id` or `app-id`; objects missing `private-key` **MUST** be rejected.

### Schema Validation

1. The JSON Schema for `github-app` objects **MUST** express validity via `anyOf` with two accepted combinations: `["client-id", "private-key"]` and `["app-id", "private-key"]`.
2. The `app-id` schema property **MUST** be documented as a deprecated alias for `client-id`.
3. Schema examples **SHOULD** use `client-id` rather than `app-id`.

### Codemod

1. The `github-app-app-id-to-client-id` codemod **MUST** rename `app-id` keys to `client-id` only within `github-app` YAML blocks.
2. The codemod **MUST NOT** rename `app-id` keys that appear outside of `github-app` blocks, regardless of nesting depth.
3. The codemod **MUST** be a no-op when no `github-app.app-id` fields are present in the frontmatter.
4. The codemod **MUST** be registered in the global codemod registry returned by `GetAllCodemods()`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24493163553) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
