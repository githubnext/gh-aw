# ADR-43508: Dedicated GitHub App Token for Daily AI Credits Guardrail

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (AI-generated from PR #43508 by pelikhan)

---

### Context

The `max-daily-ai-credits` guardrail enforces a per-workflow 24-hour AI Credits budget by making GitHub API calls at runtime to read workflow run data. Prior to this change, those API calls reused the activation token or `GITHUB_TOKEN`, meaning the guardrail's own HTTP traffic counted against the primary app's credit allotment. For organizations running many workflows with tight credit budgets, this creates a feedback loop: the guardrail intended to protect the budget itself consumes credits from the same pool it is protecting.

The guardrail needs the `actions: read` permission to fetch workflow run metadata. In multi-app setups, granting this permission to a separate, purpose-specific GitHub App isolates the guardrail's credential footprint from the main automation app.

### Decision

We will extend the `max-daily-ai-credits` frontmatter field to accept an object form with a required `value` sub-key (the credit limit) and an optional `github-app` sub-key (a `GitHubAppConfig` specifying `client-id` and `private-key`). When the `github-app` sub-key is present, the compiler emits a `daily-aic-app-token` step using `actions/create-github-app-token` before the guardrail steps, and all three places that previously passed the activation token to the guardrail (`github-token` parameter and `GH_AW_GITHUB_TOKEN` env var) now call `resolveDailyAICToken`, which returns the minted token when configured and falls back to the activation token otherwise. The scalar form (`max-daily-ai-credits: 10000`) is preserved unchanged for backwards compatibility.

### Alternatives Considered

#### Alternative 1: Keep using the activation token / GITHUB_TOKEN (status quo)

The simplest option: no schema change, no new step, no additional GitHub App required. It was rejected because the problem it addresses — guardrail API calls depleting the primary app's credit budget — is a real and actionable cost that users with tight credit quotas will hit. The status quo provides no escape hatch.

#### Alternative 2: Add a separate top-level `daily-aic-github-app` frontmatter field

A dedicated top-level key (e.g. `daily-aic-github-app: {client-id: ..., private-key: ...}`) would keep the credit limit and the app config in orthogonal fields. This was not chosen because it would require users to specify the guardrail configuration across two unrelated frontmatter keys, and it obscures the relationship between the two settings. Nesting the `github-app` sub-key inside `max-daily-ai-credits` makes the association explicit and keeps the entire guardrail configuration co-located.

#### Alternative 3: Promote the existing `github_app` schema definition and reuse it at the top level

A variant of Alternative 2: reuse the existing `$defs/github_app` JSON Schema definition at a new top-level key, avoiding the object-form change to `max_daily_ai_credits_limit`. This shares the same discoverability drawbacks as Alternative 2 and was not preferred for the same reasons.

### Consequences

#### Positive
- Guardrail API calls no longer consume credits from the primary activation app or the default `GITHUB_TOKEN`, breaking the feedback loop where the guardrail erodes the budget it protects.
- The scalar form (`max-daily-ai-credits: N`) is fully backwards compatible; existing workflows need no changes unless they want the dedicated-token behavior.
- The `actions: read` permission required by the guardrail can be scoped to a minimal-privilege app, reducing the blast radius of a compromised credential.
- The `ignore-missing-key` pattern on `GitHubAppConfig` is inherited, so workflows that use expression-backed secrets (`${{ vars.APP_ID }}`) continue to handle missing or empty values gracefully by falling back to `GITHUB_TOKEN`.

#### Negative
- The `max-daily-ai-credits` frontmatter field now has a polymorphic type (scalar integer/string **or** object), increasing schema complexity and the surface area of the parser (`extractMaxDailyAICObjectValue` must be called at every point that previously consumed the raw value directly).
- Users who want the dedicated-token behavior must provision, register, and maintain an additional GitHub App with `actions: read` on the target repository, which is operational overhead not required by the status quo.
- The compiled lock file gains a new step (`daily-aic-app-token`) in workflows that use the object form, making the compiled output slightly larger and harder to read for those workflows.

#### Neutral
- The `resolveDailyAICToken` function mirrors the existing `resolveActivationToken` indirection pattern already present in the compiler; new readers must trace both call sites to understand full token resolution logic.
- JSON Schema validation is extended with a `oneOf` branch for the object form; CI schema validation tests cover the new branch.
- The step ID `daily-aic-app-token` is a new named constant (`dailyAICAppTokenStepID`) referenced in both production code and tests, establishing a shared string that must remain stable across refactors.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
