# ADR-26137: Propagate on.github-token to Activation Checkout and Lock File Hash Check Steps

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw compiler generates an activation job that includes several steps using GitHub API credentials: a reaction step, an add-comment step, a label-removal step, a sparse `.github/.agents` checkout step, and a "Check workflow lock file" (hash check) step. The `on.github-token` frontmatter field was already wired to the reaction, comment, and label-removal steps via `resolveActivationToken(data)`. However, the sparse checkout step and the lock file hash check step still used the runner's default `GITHUB_TOKEN`. In cross-org `workflow_call` scenarios—where a caller workflow in one GitHub organization invokes a callee workflow in a different organization—the default `GITHUB_TOKEN` cannot access the callee's repository contents or APIs. This causes the checkout step to fail silently and the hash check API to return HTTP 404, producing a false-positive `ERR_CONFIG: Lock file is outdated or unverifiable` error.

### Decision

We will add a `token string` parameter to `GenerateGitHubFolderCheckoutStep()` and propagate the resolved activation token—obtained via `resolveActivationToken(data)`—to both the sparse checkout step and the "Check workflow lock file" step in the activation job. When the token is empty or equals the literal string `${{ secrets.GITHUB_TOKEN }}`, no `token:` or `github-token:` field is emitted (preserving the default-token behavior for same-org scenarios). This pattern is consistent with the existing approach used for the reaction, comment, and label-removal steps.

### Alternatives Considered

#### Alternative 1: Always emit the token field (even for the default GITHUB_TOKEN)

Emit `token: ${{ secrets.GITHUB_TOKEN }}` unconditionally in the checkout step and `github-token: ${{ secrets.GITHUB_TOKEN }}` in the hash check step. This was considered because it would make the credential source explicit in all generated YAML. It was rejected because it creates unnecessary verbosity in the generated workflow YAML for the common same-org case, and because making the default explicit can mask misconfiguration (if a consumer accidentally sets `on.github-token` to the default secret reference, the emitted YAML would still be indistinguishable from the intended cross-org token).

#### Alternative 2: Create separate checkout step generators for cross-org vs. same-org scenarios

Introduce a new function (e.g., `GenerateCrossOrgGitHubFolderCheckoutStep`) that always emits a `token:` field, and keep the existing function unchanged for same-org use. This was considered because it avoids adding a parameter to an existing API. It was rejected because it duplicates the checkout step generation logic, increasing the maintenance burden, and because callers of `generateCheckoutGitHubFolderForActivation` would still need to decide which variant to call based on the same `resolveActivationToken` output—making the choice implicit rather than explicit.

#### Alternative 3: Resolve the token at the CallerSite inside generateCheckoutGitHubFolderForActivation only (not in hash check)

Apply token propagation only to the checkout step without changing the hash check step. This was considered as a minimal change. It was rejected because it leaves the hash check step vulnerable to the same cross-org 404 failure that the checkout fix addresses. The two steps both require API access to the callee repository, and the fix should be applied consistently.

### Consequences

#### Positive
- Cross-org `workflow_call` scenarios correctly use the configured token for both the sparse checkout and the lock file hash check, eliminating false-positive lock file verification errors.
- Token propagation is now consistent across all activation job steps that access repository content or the GitHub API.
- The convention (empty string or `${{ secrets.GITHUB_TOKEN }}` → no token field emitted) is enforced in a single location in `GenerateGitHubFolderCheckoutStep()`, making it easy to audit.

#### Negative
- `GenerateGitHubFolderCheckoutStep()` is a breaking API change: all existing callers must be updated to pass an explicit token argument (typically `""` for the default). This creates churn in call sites and tests.
- The suppression rule (`token == "" || token == "${{ secrets.GITHUB_TOKEN }}"`) encodes knowledge of a specific GitHub Actions expression string as a special sentinel, which is fragile if the expression format ever changes.

#### Neutral
- The generated smoke lock workflow YAML (`.github/workflows/smoke-copilot.lock.yml`) is updated to explicitly pass the token from the `on.github-token`-equivalent secret, keeping the generated and hand-maintained workflows consistent.
- Existing tests for `GenerateGitHubFolderCheckoutStep` require signature updates (passing `""` for the new token parameter) but their assertions remain unchanged for same-org behavior.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Token Parameter in GenerateGitHubFolderCheckoutStep

1. `GenerateGitHubFolderCheckoutStep` **MUST** accept a `token string` parameter after the `ref` parameter and before the `getActionPin` function parameter.
2. Implementations **MUST** emit a `token:` YAML field in the checkout step if and only if `token` is non-empty and not equal to the literal string `${{ secrets.GITHUB_TOKEN }}`.
3. Implementations **MUST NOT** emit a `token:` field when `token` is the empty string `""`.
4. Implementations **MUST NOT** emit a `token:` field when `token` is exactly `${{ secrets.GITHUB_TOKEN }}`.
5. Callers **MUST** pass an explicit token value; passing a non-empty value that is not `${{ secrets.GITHUB_TOKEN }}` **SHALL** result in the token being included in the generated YAML.

### Token Propagation in the Activation Job Compiler

1. The activation job compiler **MUST** call `resolveActivationToken(data)` once per activation job build and reuse the result for all steps that require credential access.
2. The resolved activation token **MUST** be passed to `GenerateGitHubFolderCheckoutStep()` for the `.github/.agents` sparse checkout step.
3. The "Check workflow lock file" step **MUST** emit a `github-token:` field using the resolved activation token if and only if that token is not equal to `${{ secrets.GITHUB_TOKEN }}`.
4. The token propagation pattern for the checkout step and hash check step **SHOULD** remain consistent with the propagation pattern for the reaction, comment, and label-removal steps.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24376710842) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
