# ADR-34875: Generate PR Patches from `current: true` Checkout Path in Multi-Checkout Workflows

**Date**: 2026-05-26
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Multi-repository workflows can check out a target repository into a subdirectory via `checkout: { current: true, path: ./subdir }`, while the workflow itself runs from `GITHUB_WORKSPACE`. The `create_pull_request` and `push_to_pull_request_branch` safe-output handlers previously generated their git patches from `GITHUB_WORKSPACE` unconditionally. When the workspace root happened to contain the *outer* repository as a gitlink/submodule pointer to the checked-out subdirectory, `git diff` produced subproject (gitlink) diffs instead of the file-level changes the agent actually made inside the subdirectory. The result was PRs whose patches did not contain the agent's real edits — only a gitlink change that GitHub could not apply.

### Decision

We will thread the `current: true` checkout subdirectory from the compiler into the runtime handler config and use it as the `cwd` for patch generation when the target repo matches the current checkout. Concretely, `CheckoutManager` exposes `GetCurrentCheckoutPath()` and `GetCurrentRepository()`; `injectCurrentCheckoutPatchWorkspacePath()` injects `patch_workspace_path` and `current_checkout_repo` into the `create_pull_request` and `push_to_pull_request_branch` handler configs when a non-root `current: true` checkout is present and its repo does not conflict with an explicit `target-repo`. The runtime handlers validate the path (must stay under `GITHUB_WORKSPACE`, must exist, must be a directory) and pass it to `generateGitPatch()` via a new `workspacePath` option. This treats the compiler as the authority on *where* the agent worked, eliminating the brittle runtime-only assumption that `GITHUB_WORKSPACE` is the right git root.

### Alternatives Considered

#### Alternative 1: Continue with runtime-only checkout discovery (`findRepoCheckout()`)

The handlers already had a `findRepoCheckout()` path that scanned the workspace for a repo slug at runtime. This was used when `entry.repo` or `prConfig["target-repo"]` was set explicitly, but did not fire for the implicit `current: true` case. Extending it to scan for *any* gitlink-pointing subdirectory was rejected because runtime discovery can be wrong in the same way the workspace-root assumption was wrong (e.g., multiple checkouts of the same repo, nested working trees), and because the compiler already knows the answer unambiguously from `CheckoutConfigs`.

#### Alternative 2: Require users to set `target-repo` explicitly to enable subdirectory patching

We could have told users to always set `safe-outputs.create-pull-request.target-repo` whenever they used `current: true` with a subdirectory path, so the existing `findRepoCheckout()` codepath would be used. This was rejected because `current: true` already declares the intent unambiguously — making users restate it would be a footgun, and existing workflows would silently keep producing broken gitlink patches until someone noticed.

#### Alternative 3: Detect gitlink diffs at runtime and rerun from the gitlink target

The runtime could inspect the generated patch, detect that it consists entirely of subproject (gitlink) changes, and rerun from the resolved gitlink path. Rejected because it is reactive rather than corrective (the first patch is wasted), and a gitlink diff is sometimes legitimate (e.g., true submodule updates), so the heuristic would be ambiguous.

### Consequences

#### Positive
- PR patches in `current: true` multi-checkout workflows now contain the agent's real file edits instead of gitlink diffs.
- The compiler is the single source of truth for the patch working directory; runtime no longer has to infer it from filesystem layout.
- Path is validated against `GITHUB_WORKSPACE` containment, rejecting `..` traversal and missing/non-directory targets before invoking git.

#### Negative
- The handler config surface grows by two fields (`patch_workspace_path`, `current_checkout_repo`), increasing the documented contract between compiler and runtime.
- Path-resolution and validation logic is duplicated between `generate_git_patch.cjs` and `safe_outputs_handlers.cjs` (each enforces the under-`GITHUB_WORKSPACE` check independently).
- The injection logic is conditional on a non-trivial predicate (no `target-repo` conflict, not wildcard, not root path), which future contributors must understand before changing checkout or safe-output behavior.

#### Neutral
- `options.workspacePath` takes precedence over the existing `options.cwd` parameter on `generateGitPatch()`, establishing an order of precedence that should be preserved.
- The compiled handler config now varies based on `CheckoutConfigs`, so snapshot tests that previously ignored checkout state may need refreshing.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Compile-Time Injection

1. When a workflow declares a `checkout` entry with `current: true` and a non-root `path`, the compiler **MUST** inject `patch_workspace_path` (the normalized, workspace-relative path) into the handler configs for `create_pull_request` and `push_to_pull_request_branch`.
2. When the `current: true` checkout has a non-empty `repository` slug, the compiler **MUST** also inject `current_checkout_repo` with that slug into the same handler configs.
3. The compiler **MUST NOT** inject `patch_workspace_path` when the current checkout path normalizes to empty or `.` (i.e., workspace root).
4. The compiler **MUST NOT** inject `patch_workspace_path` when the handler's `target-repo` is set to a value that differs from the `current: true` checkout's `repository` slug.
5. The compiler **MUST NOT** inject `patch_workspace_path` when the handler's `target-repo` is `*` (wildcard).
6. The compiler **SHOULD** normalize the injected path to a forward-slash, workspace-relative form and **MUST** reject paths that resolve outside `GITHUB_WORKSPACE`.

### Runtime Patch Generation

1. When `patch_workspace_path` is present in handler config and the handler's resolved target repo matches `current_checkout_repo` (or `current_checkout_repo` is empty), the runtime **MUST** use that path as the git working directory for patch generation.
2. The runtime **MUST** validate `patch_workspace_path` before use, rejecting it if any of the following hold:
   1. The resolved path escapes `GITHUB_WORKSPACE` (path traversal).
   2. The resolved path does not exist on disk.
   3. The resolved path exists but is not a directory.
3. On validation failure, the runtime **MUST** return an error response from the handler and **MUST NOT** fall back to `GITHUB_WORKSPACE` for that invocation.
4. The runtime `generateGitPatch()` function **MUST** treat `options.workspacePath` as having higher precedence than `options.cwd` when both are supplied.
5. When `patch_workspace_path` is not present, the runtime **MUST** preserve the prior behavior (use `options.cwd`, then `GITHUB_WORKSPACE`, then `process.cwd()`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26434943762) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
