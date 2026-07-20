# ADR-46855: Refactor Bootstrap Profile Runner into Focused Modules

**Date**: 2026-07-20
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/bootstrap_profile_runner.go` had grown past a healthy size threshold and mixed four distinct concerns in a single file: top-level orchestration, repository variable/secret/Copilot-auth mutations, GitHub App manifest-registration flow, git commit-and-push operations, and shared parsing/prompt/network helpers. This made the file hard to navigate, review, and test in isolation. Individual logical units (e.g., the GitHub App browser callback server, the git push helper) could not be exercised by focused unit tests without pulling in all adjacent concerns. The existing monolithic test file mirrored this problem, exercising only the top-level control flow rather than the extracted logic paths.

### Decision

We will decompose `bootstrap_profile_runner.go` into four focused files within the same `cli` package, each responsible for one concern:

- `bootstrap_profile_actions_repo.go` — repository variable, secret, and Copilot-auth action handlers
- `bootstrap_profile_github_app.go` — GitHub App manifest registration flow, installation polling, and credential handling
- `bootstrap_profile_git.go` — commit-and-push bootstrap action and git command helpers
- `bootstrap_profile_helpers.go` — parsing, prompt/env resolution, naming, HTML/browser/network utilities, and Copilot permission detection

The top-level runner retains orchestration only (`executeBootstrapProfile`, `applyBootstrapAction`, `bootstrapProfileState`, `bootstrapActionNeedsMutation`). Shared types and injection points remain with the runner to preserve the external CLI surface unchanged. Focused test files are added alongside each new module.

### Alternatives Considered

#### Alternative 1: Keep the Monolithic File, Improve Organization with Comments

Retain `bootstrap_profile_runner.go` as a single file and introduce section comments or `//region` markers to delineate concerns internally. This avoids any file-restructuring risk and requires no changes to test organization. However, it does not improve testability (extracted helpers cannot be tested independently), does not reduce merge-conflict surface on the single file, and does not enforce separation of concerns — the file continues to grow unboundedly as new action types are added.

#### Alternative 2: Extract a Separate Package (`pkg/bootstrap/`)

Move bootstrap logic out of the `cli` package entirely into a dedicated `pkg/bootstrap/` package with an exported API. This would create a hard package boundary that Go's toolchain enforces, preventing inadvertent recoupling. The cost is a substantially more invasive change: exported type names, cross-package visibility decisions, and import rewiring across the `cli` package are all required. This was not chosen because the scope exceeded the stated goal of improving file organization without changing behavior, and the intra-package approach achieves most of the testability benefit with far less churn.

### Consequences

#### Positive
- Each concern is independently navigable and reviewable — a contributor looking at the GitHub App flow reads one ~500-line file rather than scanning a much larger mixed file
- Focused test files can exercise extracted helpers directly, improving test coverage granularity and making test failures easier to attribute to specific subsystems
- Merge-conflict surface per logical concern is reduced: changes to, e.g., the git helper no longer touch the same file as changes to GitHub App registration

#### Negative
- The bootstrap domain now spans five files rather than one; contributors unfamiliar with the decomposition must discover the module map before making changes
- Intra-package boundaries are not enforced by the Go compiler — all symbols remain visible within `cli`, so discipline (code review, naming conventions) is the only guard against recoupling over time

#### Neutral
- The external CLI surface and all observable behavior are preserved unchanged; no callers outside `pkg/cli` are affected
- Shared injection points (function variables for side-effecting operations like `bootstrapUpsertVariable`, `bootstrapSetSecret`) are retained in the runner or helpers file and remain accessible to all test files in the package via the `package cli` test build tag

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
