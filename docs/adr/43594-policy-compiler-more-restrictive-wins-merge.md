# ADR-43594: Policy Compiler with More-Restrictive-Wins Merge Semantics

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw agent orchestrator needs a runtime authorization layer that controls what actions an agent may take for a given intent (e.g., which tools it may call, whether it may auto-merge, how many attempts it may make). Before this PR, the `Authorizer.AuthorizeTool` function was unimplemented — any policy defined in `.github/intent-policy.json` was purely advisory with no runtime effect.

The governance spec (`specs/intent-attribution-agent-governance.md`) requires that policy precedence follows a strict hierarchy: organization constraints > repository constraints > intent-specific rules > workflow defaults > agent request. The key safety constraint is fail-closed behavior: when no rules match (or intent is ambiguous), the system MUST apply the most restrictive policy possible rather than granting elevated authority.

A multi-rule policy system needs a merge strategy that determines which value wins when rules at different precedence levels specify conflicting constraints. The wrong strategy could allow a lower-precedence rule to silently weaken a security constraint established by a higher-precedence rule.

### Decision

We will implement a `PolicyCompiler` in `pkg/intent/policy.go` that compiles an ordered list of `PolicyRule` entries into a single `ExecutionPolicy` using **more-restrictive-wins** merge semantics. Rules are processed in declaration order (highest to lowest precedence). For each field, the merge function always keeps the value that represents the stricter constraint: `DeniedTools` and `RequiredChecks` use union, `AllowedTools` uses intersection when both sides constrain, `HumanApprovalRequired` uses OR, `AutoMergeAllowed` uses AND, and `MaxAttempts` uses min. The baseline is always `safestDefaultPolicy()` — a fail-closed policy that denies all write operations and requires human approval.

### Alternatives Considered

#### Alternative 1: Last-Write-Wins (Declaration Order Override)

Each rule applies sequentially, with later rules fully overriding earlier rules' values. This is the simplest implementation and matches how many configuration systems work (e.g., environment variable layering).

This was rejected because it allows lower-precedence rules (e.g., intent-specific) to silently weaken constraints established by higher-precedence rules (e.g., organizational). An org rule setting `human_approval_required: true` could be cleared by a downstream repo rule setting it `false`. This violates the spec's requirement that higher-precedence rules must not be weakened.

#### Alternative 2: Single Flat Policy (No Rule Precedence)

Eliminate rule scopes and precedence entirely. Maintain a single global `ExecutionPolicy` per repository, loaded from a config file. Intent-specific behavior is encoded as label-to-policy mappings in a flat lookup table, not as composable rules.

This was rejected because it cannot express the layered governance model required by the spec (org-wide defaults + repo overrides + intent-specific adjustments). A flat model would require duplicating the most restrictive constraints in every policy entry, making the config error-prone and difficult to audit.

### Consequences

#### Positive
- Fail-closed by default: when no rules match, `safestDefaultPolicy()` produces `autonomy=propose_only`, `write_scope=none`, `human_approval_required=true`, `auto_merge_allowed=false`, `max_attempts=1`.
- Higher-precedence org/repo constraints cannot be weakened by lower-precedence intent rules; the merge invariant is enforced at the `mergePolicy()` level, making it impossible to bypass by rule ordering.
- `RuleIDs` field records which rules contributed to the compiled policy, enabling auditability of each policy decision.
- 3 unit tests covering org → repo → intent precedence are included and passing.

#### Negative
- The compiled `ExecutionPolicy` is currently **not wired to runtime enforcement**: no orchestrator path calls `AuthorizeTool` or checks `WriteScope`/`HumanApprovalRequired`/`RequiredChecks` at execution time. Policy today is advisory only. All fields in `ExecutionPolicy` are documented as "Not wired" in the spec's implementation audit table.
- `PolicyCondition.Matches` checks label values as flat strings across all dimensions (domain, priority, risk). Callers must ensure label values are unique across dimensions to avoid false positives (e.g., a domain value that collides with a priority value would cause incorrect rule matches).

#### Neutral
- The `PolicyCompiler` lives in `pkg/intent/policy.go` alongside the existing attribution types; it does not yet have its own sub-package. A follow-up will move authorization logic to `pkg/intent/authz` when `AuthorizeTool` is implemented.
- The merge semantics for `AllowedTools` are asymmetric: if neither side specifies tools, the result is unrestricted; if only one side specifies tools, that restriction is adopted; if both sides restrict, the intersection is used. This means adding `AllowedTools` to a rule makes it harder to relax at lower precedence levels.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
