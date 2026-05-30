# ADR-35939: Anthropic Workload Identity Federation Support in the Compiler

**Date**: 2026-05-30
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The AWF firewall api-proxy already supports Anthropic Workload Identity Federation (WIF), but the `gh-aw` compiler did not: `EngineAuthConfig` (in `pkg/workflow/engine.go`) modeled only the Azure WIF provider, and `ClaudeEngine` (in `pkg/workflow/claude_engine.go`) unconditionally required the static `ANTHROPIC_API_KEY` secret. As a result, a workflow configured for Anthropic WIF would fail the compiler's secret-validation step before the agent ever ran, even though no static key is needed in that mode. The constraint is that WIF-configured workflows authenticate via short-lived GitHub OIDC tokens exchanged through a federation rule, not a long-lived API key, so the compiler must be able to recognize that mode and suppress the static-key requirement while still emitting the `AWF_AUTH_*` environment variables the firewall sidecar consumes.

### Decision

We will add Anthropic WIF as a first-class authentication provider in the compiler, gated by `engine.auth.type: github-oidc` together with `engine.auth.provider: anthropic`. `EngineAuthConfig` gains a `Provider` discriminator and four Anthropic-specific fields (`AnthropicFederationRuleID`, `AnthropicOrganizationID`, `AnthropicServiceAccountID`, `AnthropicWorkspaceID`); `parseEngineAuthConfig` reads the matching frontmatter keys; and `applyEngineAuthEnv` emits `AWF_AUTH_PROVIDER` plus the four `AWF_AUTH_ANTHROPIC_*` variables. When the new `isAnthropicWIF` helper detects this mode, `ClaudeEngine.GetRequiredSecretNames` drops `ANTHROPIC_API_KEY` (while still collecting MCP secrets) and `GetSecretValidationStep` returns an empty no-op step. The design deliberately mirrors the existing Azure WIF structure rather than inventing a new shape.

### Alternatives Considered

#### Alternative 1: Require a placeholder `ANTHROPIC_API_KEY` secret for WIF workflows

We could leave `ClaudeEngine` unchanged and ask WIF users to register a dummy `ANTHROPIC_API_KEY` repository secret to satisfy the existing validation step. Rejected because it forces every WIF consumer to provision a meaningless secret, defeats the security benefit of federation (no long-lived key on the repo), and produces a confusing validation step that checks for a value the runtime never uses.

#### Alternative 2: A generic, provider-agnostic auth field bag

Instead of per-provider named fields, `EngineAuthConfig` could carry an opaque `map[string]string` of auth parameters passed straight through to `AWF_AUTH_*` env vars. Rejected because it loses compile-time structure: typos in frontmatter keys would silently flow through, the workflow schema could not validate field names, and the Azure path already uses explicit named fields — a split convention would be harder to maintain than parallel named blocks.

#### Alternative 3: Reuse the existing Azure WIF fields generically

Because both Azure and Anthropic WIF exchange a GitHub OIDC token, we considered renaming the Azure fields to provider-neutral names and reusing them for Anthropic. Rejected because the two providers carry genuinely different identifiers (tenant/client/scope/cloud vs. federation-rule/organization/service-account/workspace); overloading one field set would conflate distinct concepts and make the emitted `AWF_AUTH_*` contract ambiguous.

### Consequences

#### Positive
- Workflows configured for Anthropic WIF now compile and pass secret validation instead of failing on a missing `ANTHROPIC_API_KEY`.
- The compiler stops requiring a static API key in WIF mode, removing a long-lived credential from the repository's secret surface.
- The implementation mirrors the established Azure WIF pattern (named fields, `applyEngineAuthEnv` mapping), keeping the two providers structurally consistent and easy to reason about.
- MCP secrets are still collected in WIF mode, so workflows that combine Anthropic WIF with MCP gateways keep working unchanged.

#### Negative
- `EngineAuthConfig` grows a second provider-specific field block, and any future provider will add another — the per-provider duplication is accepted now but will eventually motivate a more general abstraction.
- The `AWF_AUTH_ANTHROPIC_*` environment-variable names form an implicit contract with the firewall api-proxy sidecar; the two sides must be kept in sync, and a rename on either side silently breaks authentication at runtime rather than at compile time.
- WIF mode is detected by a string-pair check (`type == "github-oidc" && provider == "anthropic"`); a typo in either frontmatter value silently falls back to static-key behavior and the old failure mode.

#### Neutral
- `provider` becomes a meaningful discriminator on `engine.auth`; existing Azure configs that omit `provider` are unaffected because the Azure branch keys off the Azure-specific fields.
- The frontmatter keys (`federation-rule-id`, `organization-id`, `service-account-id`, `workspace-id`) map 1:1 to the emitted `AWF_AUTH_ANTHROPIC_*` env vars, following the same kebab-case-to-SCREAMING_SNAKE convention used elsewhere.
- Test coverage was added across `claude_engine_test.go`, `engine_config_test.go`, `engine_helpers_secrets_test.go`, and `secret_validation_test.go`, exercising the helper, env mapping, secret skipping, and validation no-op.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter and Configuration Model

1. The compiler **MUST** accept an optional `engine.auth.provider` string distinguishing the WIF provider; the recognized values **MUST** include `azure` and `anthropic`.
2. When `engine.auth.provider` is `anthropic`, the compiler **MUST** read the frontmatter keys `federation-rule-id`, `organization-id`, `service-account-id`, and `workspace-id` into the corresponding `EngineAuthConfig` Anthropic fields.
3. The compiler **MUST NOT** reuse the Azure-specific fields to carry Anthropic WIF identifiers; the two provider field sets **MUST** remain distinct.

### WIF Detection and Secret Handling

1. Anthropic WIF mode **MUST** be recognized only when `engine.auth.type` is `github-oidc` and `engine.auth.provider` is `anthropic`.
2. The WIF-detection helper **MUST** return false when `WorkflowData`, its `EngineConfig`, or its `Auth` is `nil`.
3. When Anthropic WIF is active, `ClaudeEngine.GetRequiredSecretNames` **MUST NOT** include `ANTHROPIC_API_KEY` and **MUST** still return the common MCP secrets that the workflow requires.
4. When Anthropic WIF is active, `ClaudeEngine.GetSecretValidationStep` **MUST** return an empty (no-op) step.
5. When Anthropic WIF is not active, the compiler **MUST** preserve the prior behavior of requiring and validating `ANTHROPIC_API_KEY`.

### Environment Variable Emission

1. When `engine.auth.provider` is set, the compiler **MUST** emit an `AWF_AUTH_PROVIDER` environment variable with that value.
2. For each non-empty Anthropic WIF field, the compiler **MUST** emit the matching environment variable: `AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID`, `AWF_AUTH_ANTHROPIC_ORGANIZATION_ID`, `AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID`, and `AWF_AUTH_ANTHROPIC_WORKSPACE_ID`.
3. The compiler **MUST NOT** overwrite an `AWF_AUTH_*` environment variable that has already been set (e.g. via `engine.env`).
4. The compiler **MUST NOT** emit an `AWF_AUTH_ANTHROPIC_*` variable whose source field is empty.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26687105132) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
