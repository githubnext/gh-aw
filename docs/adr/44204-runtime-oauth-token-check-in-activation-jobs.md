# ADR-44204: Runtime OAuth Token Detection in Activation Jobs

**Date**: 2026-07-08
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

GitHub OAuth tokens (prefixed `gho_`) are user-scoped credentials that cannot be restricted to specific repositories and are tied to the user's session lifecycle. When these tokens are mistakenly configured as `COPILOT_GITHUB_TOKEN`, `GH_AW_GITHUB_TOKEN`, or `GH_AW_GITHUB_MCP_SERVER_TOKEN` in automation workflows, they represent an over-privileged credential that violates least-privilege principles. The system currently has no automated enforcement to prevent operators from using OAuth tokens where fine-grained PATs are required. Detecting this at runtime — during the activation job, before agent execution begins — provides a clear error with actionable remediation before any downstream harm occurs.

### Decision

We will inject a `check-oauth-tokens` step into every activation job via a new compiler method (`addActivationOAuthTokenCheckStep`) that runs a shell script (`check_oauth_tokens.sh`) checking for the `gho_` prefix in the three token environment variables. On detection, the step writes an actionable error to `$GITHUB_STEP_SUMMARY` linking to PAT creation docs and exits 1, blocking the workflow run. The check is inserted after the engine-specific `validate-secret` step to preserve existing validation ordering.

### Alternatives Considered

#### Alternative 1: Static analysis at compile time

Detect OAuth token patterns during workflow compilation (e.g., scanning configured secret expressions before generating lock files). This would catch the misconfiguration earlier in the developer workflow.

Not chosen because: secret values are not available at compile time — only the secret names/expressions are known. The `gho_` prefix is a runtime value, making compile-time detection impossible without additional out-of-band secret resolution infrastructure.

#### Alternative 2: Documentation and policy enforcement only

Warn in onboarding documentation and repository READMEs that OAuth tokens must not be used, relying on developer awareness rather than automated enforcement.

Not chosen because: this approach has no enforcement mechanism; the problem is already known (the PR description cites over-provisioning as the motivation) but relies on every operator reading and following guidance correctly. Automated enforcement is more reliable at scale across many workflow configurations.

#### Alternative 3: Validation in `getOctokit` / SDK layer

Add the OAuth token check inside the `getOctokit` helper so any code attempting to use the token would fail.

Not chosen because: this fails later in the agent execution phase after environment setup is complete, making error attribution harder. Failing fast in the activation job (before checkout and MCP gateway startup) produces a cleaner, faster failure with a better user-facing message.

### Consequences

#### Positive
- Operators receive an immediate, actionable error message in `$GITHUB_STEP_SUMMARY` with a direct link to create a fine-grained PAT, reducing time to resolution.
- Over-privileged OAuth tokens are systematically blocked from reaching agent execution across all activation jobs without requiring per-workflow configuration.
- The check is skipped when tokens are not set, so workflows not using these secrets are unaffected.

#### Negative
- Adds a step to every activation job, increasing job duration by a small but nonzero amount (shell script execution overhead).
- If GitHub changes the OAuth token prefix convention (currently `gho_`), the check becomes outdated without any compile-time visibility into the stale logic.
- Operators who intentionally use OAuth tokens for local development or non-production scenarios will be blocked and must either use a PAT or modify the check.

#### Neutral
- The `engine.env` override mechanism for `COPILOT_GITHUB_TOKEN` is respected by the new step, consistent with how `validate-secret` handles engine-specific overrides.
- All existing tests for activation job compilation required updating to account for the new step's presence and ordering.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
