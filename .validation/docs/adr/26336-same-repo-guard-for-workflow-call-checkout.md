# ADR-26336: Same-Repo Guard for workflow_call Checkout with Default GITHUB_TOKEN

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The activation job in gh-aw compiles a `.github` sparse checkout step for workflows triggered via `workflow_call`. When a workflow in a calling repository (e.g., `org/app`) invokes a reusable workflow in a different callee repository (e.g., `org/.github`), the `GITHUB_TOKEN` available at runtime is always scoped to the *calling* repository. Attempting a checkout of the callee repository with that token fails with "Repository not found." This failure mode was introduced when the `.github` checkout was moved into the activation job. Users who configure a custom activation token (`activation-github-token` or `activation-github-app`) are not affected because their token has broader scope.

### Decision

We will inject a compile-time `if:` guard onto checkout steps when no custom activation token is configured (i.e., only `GITHUB_TOKEN` is available). The guard condition `steps.resolve-host-repo.outputs.target_repo == github.repository` ensures the checkout step runs only for same-repo invocations, where `GITHUB_TOKEN` can access the target repository. For cross-repo invocations without a custom token, the checkout step is skipped, restoring the pre-regression behavior. Users who need cross-repo checkout can opt in by configuring a custom activation token in their workflow frontmatter.

### Alternatives Considered

#### Alternative 1: Raise a Compile-Time Error for Cross-Repo workflow_call Without Custom Token

The compiler could emit an error (or warning) when it detects a `workflow_call` trigger without a custom activation token, instructing the user to configure one. This was not chosen because it would be a breaking change for users who currently rely on cross-repo `workflow_call` with `GITHUB_TOKEN` and do not need the `.github` checkout — i.e., it conflates two separate concerns (token scope and checkout need) and would require all cross-repo callers to immediately add custom tokens or opt-out configuration.

#### Alternative 2: Add an Explicit Opt-In Configuration Field

Introduce a frontmatter field (e.g., `activation-skip-cross-repo-checkout: true`) that users must set to suppress the checkout in cross-repo scenarios. This was not chosen because it shifts the burden onto every user of cross-repo `workflow_call`, requires documentation and migration for existing workflows, and cannot be a safe default — the safer default is to not attempt a checkout that will fail.

#### Alternative 3: Detect the Token Scope at Runtime and Skip Dynamically

Instead of injecting a compile-time condition, the activation job could probe the target repository at runtime (e.g., via a `gh api` call) and skip the checkout if access is denied. This was not chosen because it introduces a network call in the critical path of every activation job, adds latency and failure modes, and requires additional permissions or error-handling logic that is harder to maintain than a simple static condition.

### Consequences

#### Positive
- Same-repo `workflow_call` invocations continue to work without any change.
- Cross-repo `workflow_call` invocations with only `GITHUB_TOKEN` no longer fail with "Repository not found" — they silently skip the checkout, matching pre-regression behavior.
- Users with custom activation tokens are unaffected and retain full cross-repo checkout capability.
- The fix is a compile-time transformation with no runtime cost.

#### Negative
- Cross-repo `workflow_call` callers who rely on `.github` checkout content (e.g., shared agent definitions) will silently skip that checkout when using only `GITHUB_TOKEN`. They must configure a custom token to re-enable the checkout; this is not immediately obvious from the skipped step alone.
- The guard is injected via string manipulation on already-generated YAML step strings (`injectIfConditionAfterName`), which is fragile with respect to changes in YAML indentation conventions. If the step generator's indentation changes, the 8-space prefix assumption in `injectIfConditionAfterName` will produce invalid YAML.

#### Neutral
- The token-scope check (`activationToken == "${{ secrets.GITHUB_TOKEN }}"`) relies on a string comparison against the literal expression. Any future change to how the default token is represented would require updating this comparison.
- Existing smoke-test `.lock.yml` files for `workflow_call` scenarios are recompiled to include the guard, serving as golden-file regression tests.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Checkout Guard Injection

1. When compiling a `workflow_call` activation job, implementations **MUST** check whether a custom activation token is configured.
2. When no custom activation token is configured (i.e., the activation token equals the default `${{ secrets.GITHUB_TOKEN }}` expression), implementations **MUST** inject an `if:` guard on each generated checkout step that restricts execution to invocations where the target repository equals the calling repository.
3. When a custom activation token is configured, implementations **MUST NOT** inject the same-repo guard — the checkout **SHALL** run unconditionally as before.
4. The injected guard condition **MUST** be `steps.resolve-host-repo.outputs.target_repo == github.repository`.
5. The `if:` field **MUST** be placed immediately after the step's `name:` line and before any other step fields.

### Token-Scope Detection

1. Implementations **MUST** treat the literal string `${{ secrets.GITHUB_TOKEN }}` as the sentinel value representing the default (unscoped) token.
2. Implementations **MUST NOT** apply the same-repo guard when the activation token is any value other than `${{ secrets.GITHUB_TOKEN }}`, including custom token expressions or app-based tokens.
3. Implementations **SHOULD** log a message when the guard is injected, to aid debugging of unexpected checkout skips.

### YAML Indentation Consistency

1. The `if:` field injected by the guard **MUST** use the same indentation level as other step fields generated by `GenerateGitHubFolderCheckoutStep`.
2. If the step string structure changes such that the indentation convention cannot be reliably detected, implementations **MUST NOT** silently emit malformed YAML — they **SHOULD** log a warning and return the unmodified step string.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24431549578) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
