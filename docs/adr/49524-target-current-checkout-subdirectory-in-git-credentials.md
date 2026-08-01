# ADR-49524: Make Configure Git Credentials Steps Checkout-Context-Aware

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `Configure Git credentials` steps emitted by the workflow compiler always execute from `GITHUB_WORKSPACE` (the workspace root). When a workflow uses `checkout: current: true` to select a subdirectory repository, a pre-agent step may remove the workspace-root `.git` directory as part of cleanup. In this scenario, `configure_git_credentials.sh` — which calls `git remote set-url origin` — fails with `fatal: not a git repository` because it runs from a directory that no longer contains a `.git` directory. The credentials step must target the correct `.git` directory that corresponds to the current checkout.

### Decision

We will introduce `generateGitConfigurationStepsForData(*WorkflowData)` in the workflow compiler. This function uses `CheckoutManager` to inspect the workflow's checkout configuration. When a `current: true` checkout is in a subdirectory (has a non-empty `path`), the compiler emits the credentials step with a `working-directory` pointing to that subdirectory and sets `GITHUB_REPOSITORY` to the checkout's repository slug. For all other configurations (no checkout data, current checkout at workspace root) the function falls back to the existing workspace-root behavior. Both pre-agent and post-agent `Configure Git credentials` calls are updated to use the data-aware variant.

### Alternatives Considered

#### Alternative 1: Make `configure_git_credentials.sh` discover the git directory at runtime

The shell script could walk up the directory tree (using `git -C` with a candidate path, or traversing parent directories) to find the nearest `.git` directory, instead of relying on the compile-time `working-directory`.

This was not chosen because it would make the script's behavior depend on the runner filesystem state at execution time, which is harder to test and reason about. The compiler already has full knowledge of the checkout topology, so expressing the target path as a compile-time `working-directory` keeps the runtime script simple and its behavior predictable.

#### Alternative 2: Preserve the workspace-root `.git` by changing pre-agent step generation

An alternative is to ensure the workspace root always retains a `.git` directory (or a `.git` file pointing to the correct repo) by modifying the pre-agent steps that currently remove the private control-plane checkout.

This was not chosen because it conflates two concerns: the private control-plane checkout removal is a deliberate isolation step. Keeping a stale or stub `.git` at the workspace root to satisfy the credentials script would be fragile and misleading. Targeting the real git directory is the more correct fix.

### Consequences

#### Positive
- Eliminates the `fatal: not a git repository` error for `checkout: current: true` workflows whose pre-agent steps remove the workspace-root repository.
- Backward-compatible: all existing workflows without a subdirectory `current: true` checkout continue to use the original workspace-root behavior unchanged.
- The fix is tested with comprehensive unit tests covering nil data, empty configs, root-level current checkout, subdirectory with and without an explicit repository, and nested subdirectory path normalization.

#### Negative
- `generateGitConfigurationSteps()` (workspace-root variant) and `generateGitConfigurationStepsForData()` (data-aware variant) now coexist; future authors may reach for the simpler function in new call sites and reintroduce the bug for subdirectory checkouts.
- The new function emits YAML lines directly as a string slice rather than delegating to `generateGitConfigurationStepsWithToken`, introducing partial duplication of the step structure for the subdirectory case.

#### Neutral
- The `GITHUB_REPOSITORY` environment variable is now set to the current checkout's repository slug (a literal string) rather than the `${{ github.repository }}` expression when a custom repository is configured. This is semantically correct but changes the generated YAML structure for affected workflows.
- The `CheckoutManager` abstraction is reused from the existing codebase; no new dependencies are introduced.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
