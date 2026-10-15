# ADR-69111: Centralize Pull Request URL Reporting at Creation Time

**Date**: 2026-02-23
**Status**: Draft
**Deciders**: pelikhan

---

### Context

`gh aw` has several commands that create pull requests (`gh aw pr`, workflow add/transfer flows, and the shared `CreatePRWithChanges` helper). Each of these call sites printed its own success message, and each did so *after* the command had finished post-creation bookkeeping — most notably switching the local checkout back to the original branch. If that cleanup step failed, the command returned an error and the user never saw the URL of a pull request that had in fact already been created on GitHub. The message text also differed between call sites (`"Created PR: "`, `"Created pull request "`, `"PR created successfully!"` plus a separate `"URL: "` line), so terminal emulators could not reliably surface a single clickable link.

### Decision

We will emit the pull request success message, including the full PR URL, from the single shared creation path (`createPRForRepo` in `pkg/cli/pr_command.go`) immediately after the `gh pr create` call returns, and remove the per-caller success messages in `pkg/cli/add_workflow_pr.go` and `pkg/cli/pr_helpers.go`. The PR transfer path uses the same `Created pull request: <url>` wording. The message is written to stderr unconditionally, so it is visible in quiet and non-verbose modes, and survives a later failure in branch cleanup. The primary driver is correctness of user-facing output: the URL must never be lost because of a failure that happens after the PR already exists.

### Alternatives Considered

#### Alternative 1: Keep per-caller messages but move them before branch restoration

Each call site would keep its own `FormatSuccessMessage` call and simply print it earlier, before switching branches. This is the smallest possible diff and preserves caller-specific wording. It was rejected because it does not remove the duplication: the same ordering bug can be reintroduced by the next command that creates a PR, and the inconsistent message formats (`Created PR:` vs `PR created successfully!` + `URL:`) would persist.

#### Alternative 2: Return the URL and let the top-level command layer print it

Creation helpers would stay silent and the outermost command handler would be responsible for rendering the URL after the whole operation succeeds or fails. This keeps I/O out of helper functions, which is generally the cleaner layering. It was rejected because it still ties printing to the end of the operation — the failure case this PR fixes — and would require touching every command entry point rather than one helper.

### Consequences

#### Positive
- The PR URL is always shown as soon as the PR exists, even when branch restoration or other cleanup subsequently fails.
- A single, consistent `Created pull request: <url>` message across all PR-creating commands, making the URL reliably clickable in terminals.
- New PR-creating commands that route through `createPRForRepo` inherit correct messaging for free.

#### Negative
- `createPRForRepo` now performs user-facing I/O, mixing a side effect into an otherwise data-returning helper; callers that want to suppress or customize the message have no hook to do so.
- Output is unconditional, so there is no way to silence it via a quiet flag — intentional here, but it removes a degree of caller control.

#### Neutral
- Message wording changed (`Created PR:` / `PR created successfully!` → `Created pull request:`); any scripts or tests matching the old strings must be updated.
- The transfer path keeps its separate fork-origin informational line, which now follows the success message rather than preceding a separate `URL:` line.
- Behaviour is documented in `docs/src/content/docs/setup/cli.md` and covered by new tests in `pkg/cli/pr_command_test.go`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
