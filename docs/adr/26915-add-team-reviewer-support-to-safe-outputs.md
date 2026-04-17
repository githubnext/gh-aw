# ADR-26915: Add Team Reviewer Support to `create-pull-request` and `add-reviewer` Safe Outputs

**Date**: 2026-04-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `create-pull-request` and `add-reviewer` safe output handlers previously only supported individual user reviewers through a `reviewers` field. GitHub's pull request API exposes a distinct `team_reviewers` parameter for requesting reviews from GitHub teams (identified by team slugs). Workflows that needed to assign team reviewers had no supported mechanism — team slug values passed through the `reviewers` field would be silently dropped before reaching the GitHub API. This gap blocked users who want to enforce team-level code review policies through agentic workflows.

### Decision

We will add a dedicated `team-reviewers` field to both the `create-pull-request` and `add-reviewer` safe output configurations. This field accepts either a single team slug string or an array of team slugs, mirroring the normalization behavior of the existing `reviewers` field. The compiler will pass configured team reviewers to the corresponding handler config keys (`team_reviewers` for `create_pull_request`, `allowed_team_reviewers` for `add_reviewer`), and the JavaScript runtime handlers will forward them to GitHub's `pulls.requestReviewers` API. The `add-reviewer` safe output's validation will be updated to require at least one of `reviewers` or `team_reviewers` rather than requiring `reviewers` unconditionally.

### Alternatives Considered

#### Alternative 1: Reuse the existing `reviewers` field with a prefix convention

Team slugs could be distinguished from user handles by a naming convention (e.g., `team:platform-reviewers`). This avoids schema changes but introduces an implicit, fragile parsing convention with no schema-level validation support. It also deviates from GitHub's own API model, which treats user and team reviewers as distinct parameters — mixing them into one field risks future ambiguity and makes allowlist configuration more complex.

#### Alternative 2: Add team reviewer support only to `add-reviewer`, not `create-pull-request`

Since `add-reviewer` is a discrete action, it could be modified first as a lower-risk change. However, `create-pull-request` also configures reviewers at PR creation time through the same handler infrastructure. Splitting the implementation would result in inconsistent behavior between the two safe outputs, requiring users to work around the gap with an extra `add-reviewer` step after every PR creation.

#### Alternative 3: Expose a generic `extra_params` pass-through in the safe output config

A free-form escape hatch would let callers forward arbitrary GitHub API parameters including `team_reviewers`. This was not chosen because it bypasses the allowlist security model that safe outputs are built on — the design requires that all outputs be validated against operator-configured constraints before reaching the GitHub API.

### Consequences

#### Positive
- Workflows can now request team reviews at PR creation time and via the `add-reviewer` handler, removing a gap in the safe output API surface.
- The implementation is consistent with the existing `reviewers` field: single-string-to-array normalization, allowlist filtering, and schema validation all work identically for team reviewers.
- Existing `copilot` reviewer handling is unaffected and continues to be requested through its own code path.

#### Negative
- Operators must now configure `allowed_team_reviewers` in their handler config to restrict which teams can be requested as reviewers, adding a new allowlist to maintain.
- The `add-reviewer` safe output now uses a `requiresOneOf` validation rule instead of a hard `required: true` on `reviewers`, which is a slight loosening of the input schema that must be communicated to existing users.

#### Neutral
- The JSON schemas (`main_workflow_schema.json`, `safe_outputs_tools.json`) and TypeScript config typings are updated to reflect the new field.
- Go and JavaScript tests are extended to cover handler config emission, pass-through, copilot coexistence, and allowlist filtering for team reviewers.
- Team slug length is validated against a new `MaxGitHubTeamSlugLength = 100` constant, consistent with how usernames are validated via `MaxGitHubUsernameLength`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Safe Output Configuration

1. Implementations **MUST** accept a `team-reviewers` field in both the `create-pull-request` and `add-reviewer` safe output configuration blocks.
2. Implementations **MUST** normalize a single team-slug string value for `team-reviewers` to a single-element array before YAML unmarshaling.
3. Implementations **MUST NOT** silently drop `team-reviewers` values during config parsing or handler config generation.
4. Implementations **MUST** validate each team slug against `MaxGitHubTeamSlugLength` (100 characters).

### Handler Config Generation

1. Implementations **MUST** emit `team_reviewers` in the `create_pull_request` handler config when `team-reviewers` is configured.
2. Implementations **MUST** emit `allowed_team_reviewers` in the `add_reviewer` handler config when `team-reviewers` is configured.
3. Implementations **MUST NOT** merge team reviewer slugs into the user `reviewers` / `allowed` handler config fields.

### Runtime Behavior

1. The `create_pull_request` handler **MUST** forward configured team reviewers to the `team_reviewers` parameter of the GitHub `pulls.requestReviewers` API call.
2. The `add_reviewer` handler **MUST** filter team reviewers from tool output against the `allowed_team_reviewers` allowlist before forwarding to the GitHub API.
3. The `add_reviewer` handler **MUST** accept tool outputs that contain `team_reviewers` but no `reviewers`, and vice versa; at least one **MUST** be present.
4. The Copilot reviewer **SHOULD** continue to be requested via its dedicated code path and **MUST NOT** be affected by team reviewer changes.

### Schema and Validation

1. The workflow schema (`main_workflow_schema.json`) **MUST** declare `team-reviewers` as an optional field accepting a string or array of strings for `create-pull-request`, and as an optional array of strings for `add-reviewer`.
2. The safe output tool schema (`safe_outputs_tools.json`) **MUST** reflect that `add_reviewer` requires at least one of `reviewers` or `team_reviewers` via an `anyOf` constraint.
3. Implementations **MUST** apply `ItemSanitize: true` to `team_reviewers` field validation, consistent with `reviewers`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24580129798) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
