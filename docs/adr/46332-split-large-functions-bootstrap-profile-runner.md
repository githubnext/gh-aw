# ADR-46332: Split Large Functions in bootstrap_profile_runner.go to Fix largefunc Lint

**Date**: 2026-07-18
**Status**: Accepted
**Deciders**: copilot-swe-agent (PR author), reviewers of PR #46332

---

### Context

`pkg/cli/bootstrap_profile_runner.go` had accumulated 14 `largefunc` lint findings (limit: 60 lines per function) concentrated in three functions: `executeBootstrapProfile` (78 lines), `runBootstrapGitHubAppAction` (86 lines), and `createBootstrapGitHubApp` (118 lines). The codebase enforces the 60-line maximum via the `largefunc` linter, and violations block CI. These functions grew large because they combine multi-branch dispatch logic, credential resolution, and HTTP handler construction in a single body. The refactoring must resolve all lint findings without changing any observable behavior.

### Decision

We will extract focused helper functions from each oversized function, grouping logically coherent blocks into named helpers: `applyBootstrapAction` (action-type dispatch), `handleBootstrapGitHubAppExistingFlow` and `handleBootstrapGitHubAppCreateOrExistingChoice` (two credential-resolution paths), `setupBootstrapGitHubAppDetails` (app metadata derivation), and `buildBootstrapGitHubAppMux` (HTTP handler construction). A new `bootstrapGitHubAppFlowChannels` struct groups the result/error channels to keep `buildBootstrapGitHubAppMux` under the 8-parameter linter limit. All extracted helpers carry the same behavior as the inlined code they replace.

### Alternatives Considered

#### Alternative 1: Raise or Disable the largefunc Limit

Increase the `largefunc` threshold for these functions, or add per-function suppression comments. This avoids the refactoring work and keeps all logic co-located. Rejected because accepting `largefunc` exceptions for already-large functions encourages further accumulation; the lint limit exists precisely to enforce readability discipline across the codebase.

#### Alternative 2: Restructure Using a Command/Strategy Object Pattern

Replace the `switch action.Type` dispatch with a registry map of handler functions or a formal strategy hierarchy. This is a more architecturally significant approach that would make adding new action types easier. Rejected as disproportionate to a lint compliance fix — the helper-extraction approach resolves all 14 findings with minimal structural churn and zero behavior change, whereas a strategy pattern would require new types, a registration mechanism, and a larger diff to review.

### Consequences

#### Positive
- All 14 `largefunc` CI findings are resolved, unblocking future changes to this file.
- Each extracted function is individually testable with a single, named responsibility.
- The `bootstrapGitHubAppFlowChannels` struct makes channel ownership explicit, improving readability of the browser manifest flow coordinator.

#### Negative
- Following `createBootstrapGitHubApp` end-to-end now requires reading through `setupBootstrapGitHubAppDetails` and `buildBootstrapGitHubAppMux` separately, adding indirection.

#### Neutral
- `io.WriteString` replaces `w.Write([]byte(...))` as an incidental cleanup — functionally identical.
- The `continue` statement in the `copilot-auth` switch branch becomes `return nil` inside the extracted `applyBootstrapAction` function — semantically equivalent, required by the extraction boundary.
- `setupBootstrapGitHubAppDetails` now uses explicit return values (no named returns) to remove fragility around bare `return` statements.
- `buildBootstrapGitHubAppMux` now receives the override-resolved `appOwner`/`appOwnerType` so `exchangeBootstrapGitHubAppCode` stores the correct owner in the created app result when `overrides.Owner` is set. This fixes a pre-existing latent bug exposed during the extraction review.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
