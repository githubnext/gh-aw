# ADR-25948: Version-Gated Integrity Reactions for MCPG Allow-Only Policy

**Date**: 2026-04-13
**Status**: Draft
**Deciders**: lpcox, Copilot (inferred from PR #25948)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler generates MCP gateway (MCPG) guard policies that control which tool calls agents are allowed to make. Until now, integrity promotion and demotion was determined solely by static fields (`min-integrity`, `repos`) in the `allow-only` policy block. A new capability in MCPG v0.2.18 allows reaction-based integrity signals: GitHub reactions (e.g., 👍, ❤️) from maintainers can dynamically promote or demote the content integrity level, enabling lightweight, in-band approval workflows without requiring separate label-based gating. Introducing this capability requires extending the compiler in a way that is both backwards-compatible with existing workflows and gated to MCPG versions that support it.

### Decision

We will introduce a `integrity-reactions` feature flag that workflow authors must explicitly opt into, combined with a semver version gate that ensures the feature is only compiled into guard policies when the configured MCPG version is `>= v0.2.18`. A shared `injectIntegrityReactionFields()` helper centralizes the injection logic and is called from both the MCP renderer (`mcp_renderer_github.go`) and the DIFC proxy policy builder (`compiler_difc_proxy.go`), ensuring consistent behavior across all policy code paths. The default MCPG version (`v0.2.17`) is deliberately below the minimum, so no existing workflow is affected without an explicit opt-in.

### Alternatives Considered

#### Alternative 1: Unconditional Rollout (No Feature Flag)

Add `endorsement-reactions` and `disapproval-reactions` to the allow-only policy for all workflows that already set `min-integrity`. This would require no feature flag infrastructure but would silently change the behaviour of every existing workflow using integrity gating as soon as MCPG >= v0.2.18 is deployed. Reaction fields default to empty arrays in MCPG so the net change would likely be benign, but the compiler would generate different output for unchanged workflow files, violating the principle that `make recompile` is idempotent without frontmatter changes. This alternative was rejected because it breaks the stable, reproducible lock-file guarantee.

#### Alternative 2: Separate Policy Type for Reaction-Based Integrity

Introduce a new top-level policy key (e.g., `reaction-integrity`) separate from the existing `allow-only` block, requiring workflow authors to restructure their guard policy when adding reactions. This would be a cleaner schema evolution in isolation but would break the conceptual unity of the guard policy (integrity level and reactions belong to the same policy object in MCPG) and would force unnecessary churn for adopters already using `min-integrity`. It was rejected because the MCPG data model treats reactions as additional fields within the existing `allow-only` block, so mirroring that structure in the frontmatter is more natural and less disruptive.

#### Alternative 3: Compiler-Inlined Version Check Instead of Helper

Duplicate the semver version-gate logic inline at each call site (MCP renderer and DIFC proxy builder) rather than centralizing it in `mcpgSupportsIntegrityReactions()` and `injectIntegrityReactionFields()`. This would eliminate the shared helper but scatter the version-comparison logic and the reaction-injection logic across multiple files, making it harder to update the minimum version or add new reaction fields in the future. It was rejected because the injection logic is non-trivial (four optional fields, two code paths) and centralization reduces the surface area for bugs when either code path is later changed.

### Consequences

#### Positive
- Existing workflows are completely unaffected — `make recompile` produces no diff unless the `integrity-reactions` feature flag is explicitly enabled in frontmatter.
- A single `injectIntegrityReactionFields()` helper ensures both the MCP renderer and DIFC proxy policy builder stay in sync when reaction fields are added or modified.
- Compile-time validation (`validateIntegrityReactions()`) catches invalid reaction content enum values and missing `min-integrity` prerequisites before any workflow runs.
- The semver gate pattern is consistent with the `version-gated-no-ask-user-flag` decision (ADR-25822), reinforcing a repository-wide convention for introducing MCPG-version-specific features.

#### Negative
- Workflow authors who want reaction-based integrity must add both `features: integrity-reactions: true` and update their MCPG version to `>= v0.2.18` — a two-part opt-in that could cause confusion if only one is set (though validation errors guide the author).
- The `getDIFCProxyPolicyJSON` function signature changed from `(githubTool any)` to `(githubTool any, data *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig)`, making it a slightly more complex internal API.
- The `ensureDefaultMCPGatewayConfig(data)` call was moved earlier in `buildStartDIFCProxyStepYAML` to ensure the gateway config is populated before policy injection — a subtle ordering dependency that future maintainers must preserve.

#### Neutral
- The `validReactionContents` enum set matches the GitHub GraphQL `ReactionContent` enum at the time of writing; if GitHub adds new reaction types, the validation set must be updated manually.
- The "latest" version string is treated as always supporting the feature — a pragmatic choice that simplifies CI pipelines that pin to `latest`, at the cost of slightly weaker version semantics.
- JSON schema (`main_workflow_schema.json`) was extended with enum constraints for the new fields, providing IDE autocompletion and static validation independent of the Go validation layer.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Feature Flag and Version Gate

1. Implementations **MUST NOT** inject `endorsement-reactions`, `disapproval-reactions`, `disapproval-integrity`, or `endorser-min-integrity` into any MCPG guard policy unless the `integrity-reactions` feature flag is explicitly enabled in the workflow frontmatter.
2. Implementations **MUST NOT** inject reaction fields if the effective MCPG version is below `v0.2.18`, even when the feature flag is enabled.
3. Implementations **MUST** treat the string `"latest"` (case-insensitive) as satisfying the minimum MCPG version requirement.
4. Implementations **MUST** treat any non-semver MCPG version string (other than `"latest"`) as failing the version gate, defaulting to conservative rejection.
5. Implementations **MUST** use `DefaultMCPGatewayVersion` when no MCPG version is explicitly configured, which **MUST** be a version below `MCPGIntegrityReactionsMinVersion` to preserve backwards compatibility.

### Reaction Field Injection

1. Implementations **MUST** inject reaction fields via the shared `injectIntegrityReactionFields()` helper — direct inline injection at individual call sites is **NOT RECOMMENDED**.
2. `injectIntegrityReactionFields()` **MUST** be called in all policy-generation code paths, including the MCP renderer (`mcp_renderer_github.go`) and the DIFC proxy policy builder (`compiler_difc_proxy.go`).
3. Implementations **MUST** inject reaction fields into the inner `allow-only` policy map, not into the outer policy wrapper object.
4. Implementations **SHOULD** call `ensureDefaultMCPGatewayConfig(data)` before invoking `injectIntegrityReactionFields()` to guarantee the gateway config is non-nil.

### Validation

1. Implementations **MUST** validate that `endorsement-reactions` and `disapproval-reactions` contain only values from the GitHub `ReactionContent` enum: `THUMBS_UP`, `THUMBS_DOWN`, `HEART`, `HOORAY`, `CONFUSED`, `ROCKET`, `EYES`, `LAUGH`.
2. Implementations **MUST** return a compile-time error if any reaction array field is set without the `integrity-reactions` feature flag.
3. Implementations **MUST** return a compile-time error if the `integrity-reactions` feature flag is enabled but the MCPG version is below `v0.2.18`.
4. Implementations **MUST** return a compile-time error if `endorsement-reactions` or `disapproval-reactions` are set without `min-integrity` being configured.
5. Implementations **MUST** validate that `disapproval-integrity`, when set, is one of: `"none"`, `"unapproved"`, `"approved"`, `"merged"`.
6. Implementations **MUST** validate that `endorser-min-integrity`, when set, is one of: `"unapproved"`, `"approved"`, `"merged"`.

### Schema

1. The JSON schema for workflow frontmatter **MUST** define `endorsement-reactions` and `disapproval-reactions` as arrays of strings constrained to the `ReactionContent` enum values.
2. The JSON schema **MUST** define `disapproval-integrity` and `endorser-min-integrity` as strings constrained to their respective valid integrity level sets.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular: injecting reaction fields without the feature flag, injecting reaction fields when the MCPG version is below `v0.2.18`, or omitting validation of reaction enum values are all non-conformant behaviors.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
