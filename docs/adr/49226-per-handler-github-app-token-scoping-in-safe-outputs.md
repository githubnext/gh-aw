# ADR-49226: Per-Handler GitHub App Token Scoping in Safe-Outputs

**Date**: 2026-07-31
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflows using the safe-outputs system that require different GitHub permission scopes across handlers (e.g. `issues:write` for `add-comment` and `actions:write` for `dispatch-workflow`) were forced to use a single top-level `github-app` block that had to satisfy all permission requirements simultaneously. This violated the principle of least privilege: a single app token held broader permissions than any individual handler needed. There was no mechanism to assign separate GitHub Apps — each scoped to only its required permissions — to individual safe-output handlers. The only workaround was replacing built-in safe-output handlers with custom jobs, which added boilerplate and operational overhead.

### Decision

We will introduce a per-handler `github-app` override in `BaseSafeOutputConfig` that, when set, causes the compiler to mint a dedicated token step (`{handler-key}-app-token`) for that handler during workflow generation. The token minting loop in `buildHandlerManagerStep` iterates all registered `safeOutputHandlers` and emits a token step for each handler whose `GitHubApp` field is set. Each handler's token step uses only the permissions declared by that handler's `PermissionBuilder`. The `resolveHandlerGitHubToken` helper selects the per-handler token expression when available, falling back to the explicit `github-token`. The global `github-app` continues to serve as a default fallback for handlers without an override. Struct field discovery is handled via reflection in `getHandlerGitHubApp`, which checks for a direct `GitHubApp` field or a `BaseSafeOutputConfig.GitHubApp` embedded field.

### Alternatives Considered

#### Alternative 1: Custom Safe-Output Jobs (Status Quo Workaround)

Each handler that needs its own app scoping is replaced by a user-authored custom job that calls `actions/create-github-app-token` independently and passes the minted token explicitly. This was the only working path before this change. It was rejected because it requires boilerplate per handler, increases workflow file maintenance burden, and pushes an infrastructure concern (token scoping) into each user's workflow definition rather than the framework.

#### Alternative 2: Single App with Combined Permissions

Continue requiring a single GitHub App that holds the union of all permissions required by all enabled safe-output handlers. This is operationally simpler — one app to manage — but is rejected because it violates least privilege and makes it impossible for organizations with separate apps per permission boundary (e.g. a dedicated issues-writer app vs. an actions-dispatcher app) to use the built-in safe-output handlers without merging those apps.

#### Alternative 3: OIDC-Based Per-Handler Token Derivation

Use GitHub OIDC to issue short-lived tokens scoped per handler without any app credentials stored as secrets. This was considered but rejected because it requires the workflow repository to be configured as an OIDC subject in each target resource, adds complexity to the token-derivation model, and is not consistently available across all safe-output handler permission surfaces (some require app tokens specifically, not OIDC tokens).

### Consequences

#### Positive
- Each handler token carries only the permissions required for that handler, enforcing least privilege at the token-minting layer.
- Organizations with separate GitHub Apps per permission domain can now use built-in safe-output handlers without workarounds.
- Token scoping is a framework concern handled at compile time; user workflow YAML remains clean.
- The pattern generalises uniformly across all ~44 registered handlers via a single loop rather than per-handler special-casing.

#### Negative
- `getHandlerGitHubApp` uses reflection to discover the `GitHubApp` field, coupling the implementation to struct field names (`GitHubApp`, `BaseSafeOutputConfig`). Renaming these fields without updating the reflection call silently breaks per-handler token minting.
- Users who want full per-handler isolation must register and maintain additional GitHub Apps — one per permission boundary — which increases the App management surface area in their organization.

#### Neutral
- The previous `create-check-run`-specific special case in `buildHandlerManagerStep` is replaced by the general loop, making `create-check-run` consistent with all other handlers.
- Handlers that do not set a per-handler `GitHubApp` are unaffected; their behaviour is unchanged (they continue to use the global app token or explicit `github-token`).
- The new test file (`safe_outputs_app_test.go`) covers the `add-comment`, `dispatch-workflow`, and multi-handler independence scenarios but does not cover every handler — edge cases in less-common handlers may surface in integration testing.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
