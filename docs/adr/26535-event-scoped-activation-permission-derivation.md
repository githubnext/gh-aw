# ADR-26535: Event-Scoped Activation Permission Derivation

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw compile` command generates a lock file that includes an activation job with a `permissions` block and a GitHub App token minting step. Before this change, whenever a compiled workflow configured `reaction` or `status-comment` triggers, the compiler unconditionally granted `issues: write`, `pull-requests: write`, and `discussions: write` — even if the workflow only triggers on `issues` events and never touches pull requests or discussions. This violated the principle of least privilege and caused compiled lock files to request far broader scopes than any workflow operation actually required. The `on:` section in gh-aw workflow frontmatter is a superset of real GitHub event names: it also contains metadata trigger fields (`reaction`, `status-comment`, `command`, etc.) that are interpreted by the framework, not forwarded to GitHub. Distinguishing real event names from metadata fields is therefore a prerequisite to computing accurate permissions.

### Decision

We will derive activation job permissions by parsing the `on:` section YAML at compile time, filtering out known metadata trigger fields, and granting only the write scopes required by the real GitHub event types that are configured. `issues: write` is granted only when `issues`, `issue_comment`, or `pull_request` events are present (since reactions and status comments on issues/PRs use the Issues REST API). `pull-requests: write` is granted when `pull_request` or `pull_request_review_comment` events are present, or when `issue_comment` is present with PR reactions enabled (because `issue_comment` fires for PR comments and GitHub requires `pull-requests: write` to react to PR comments). `discussions: write` is granted only when `discussion` or `discussion_comment` events are present. A fallback to the previous broad-grant behavior is preserved for synthetic or test `WorkflowData` instances where the `on:` section is empty.

### Alternatives Considered

#### Alternative 1: Retain Broad Permission Grants

Always grant `issues: write`, `pull-requests: write`, and `discussions: write` whenever reactions or status comments are configured. This is the pre-existing behavior and is trivially correct (it never under-grants). It was rejected because it violates the principle of least privilege: a workflow that only triggers on issues should not carry `pull-requests: write` or `discussions: write` in its lock file, as those scopes could be exploited or trigger unexpected behavior.

#### Alternative 2: Explicit Permission Declaration in Workflow Frontmatter

Require workflow authors to declare which GitHub API scopes they need (e.g., `permissions: issues: write`) directly in the workflow markdown. This would be explicit and flexible, but it places the burden of correct permission reasoning on workflow authors — who are often non-engineers — and creates a class of misconfiguration bugs where a workflow runs with too few permissions because the author forgot to declare a scope. Automated derivation at compile time is more reliable.

#### Alternative 3: Runtime Permission Escalation

Request a minimal token at activation time and escalate permissions lazily when an operation fails due to insufficient scope. This approach is more dynamic but requires network round-trips to detect permission failures and complicates error handling and auditability. Compile-time derivation is simpler to reason about and does not require runtime infrastructure changes.

### Consequences

#### Positive
- Compiled lock files satisfy least-privilege: a workflow that only reacts to issues will no longer carry `pull-requests: write` or `discussions: write` scopes.
- The permission derivation logic is centralized in two new functions (`addActivationInteractionPermissionsMap`, `activationEventSet`), making future changes to permission rules easier to locate and audit.
- The metadata field allowlist (`activationMetadataTriggerFields`) is explicit and visible, replacing implicit substring-matching heuristics.
- Regression tests directly verify least-privilege behavior for issue-only and PR-review-comment-only trigger configurations.

#### Negative
- The compiler now has a dependency on YAML parsing of the `on:` section at permission-derivation time, which adds a new failure mode: a malformed `on:` section will silently fall back to an empty event set and grant no permissions for reactions/status comments (though this is unlikely in practice since the same section is validated earlier in the compile pipeline).
- The fallback to broad permissions for empty `on:` sections means unit tests that use synthetic `WorkflowData` without a populated `on:` field will continue to get over-broad permissions, masking potential regressions in tests that don't exercise the full compile pipeline.

#### Neutral
- The `activationMetadataTriggerFields` allowlist must be kept in sync with the set of metadata keys that gh-aw recognizes in the `on:` block. Adding a new metadata key without updating this list will cause it to be treated as a real GitHub event and potentially produce incorrect permission grants.
- The change affects both the activation job `permissions` block and the GitHub App token minting permissions — both call-sites now share the same derivation logic, which is a desirable consistency improvement.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Activation Permission Derivation

1. Implementations **MUST** derive activation job write permissions from the set of real GitHub event types present in the `on:` section, not from the presence of `reaction` or `status-comment` configuration alone.
2. Implementations **MUST NOT** grant `pull-requests: write` in the activation job unless `pull_request`, `pull_request_review_comment`, or `issue_comment` is among the configured trigger events and the reaction/status-comment configuration includes pull requests. (`issue_comment` events fire for both issue comments and PR comments; since PR comments require `pull-requests: write` for reactions, the presence of `issue_comment` with PR reactions enabled mandates this permission.)
3. Implementations **MUST NOT** grant `discussions: write` in the activation job unless `discussion` or `discussion_comment` is among the configured trigger events.
4. Implementations **MUST NOT** grant `issues: write` solely for reaction/status-comment purposes unless `issues`, `issue_comment`, or `pull_request` is among the configured trigger events.
5. Implementations **MUST** apply the same permission derivation logic to both the activation job `permissions` block and the GitHub App token minting permissions.

### Metadata Field Filtering

1. Implementations **MUST** maintain an explicit allowlist of gh-aw metadata trigger fields (e.g., `reaction`, `status-comment`, `command`, `slash_command`, `label_command`, `stop-after`, `github-token`, `github-app`) and **MUST** exclude them from the derived event set.
2. Implementations **MUST NOT** treat an unrecognized key in the `on:` map as a metadata field; unknown keys **SHOULD** be treated as real GitHub event names to avoid silent under-granting.

### Fallback Behavior

1. When the `on:` section is absent or empty at permission derivation time, implementations **SHOULD** fall back to granting the full set of broad permissions (`issues: write`, `pull-requests: write`, `discussions: write`) and **MUST** emit a diagnostic log message explaining that the broad fallback is in use.
2. Implementations **MUST NOT** silently grant zero permissions for reactions or status comments when the `on:` section is malformed; a parse failure **SHOULD** be logged and the empty event set **SHOULD** result in no reaction/status-comment-related permissions being added.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24488870005) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
