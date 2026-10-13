# ADR-68547: Enforce an Explicit Positional-Argument Boundary and Workflow Destination Containment

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown (drafted from PR #68547 evidence)

---

### Context

The `gh aw` MCP server exposes workflow-management tools (`logs`, `audit`, `audit diff`, `add`, …) that translate JSON tool arguments into `gh aw` subprocess invocations. Before this change, caller-supplied values such as `workflow_name`, `run_ids_or_urls`, and `base_run_id` were appended into the argument slice interleaved with CLI flags, so a value beginning with `-` could be parsed by the CLI as an option rather than as data. Separately, the `add` tool's `name` option flowed through to `filepath.Join(githubWorkflowsDir, name+ext)` without validation, so values containing `/`, `\`, `..`, or an absolute path — or an existing symlink at the destination — could cause files to be written outside the selected workflow directory. Both paths are reachable from untrusted MCP clients, so an input-shape contract was needed at the boundary rather than deep inside each command.

### Decision

We will treat MCP tool inputs as data, never as options, by (1) rejecting positional values that start with `-` via a shared `validateMCPPositionals` helper and (2) emitting all positional values after an explicit `--` separator via `appendMCPPositionals`, with every CLI flag — including server-owned flags such as `-o` and `--json` — placed before that boundary. We will also validate `add` destinations up front: `validateAddedWorkflowName` rejects names containing path separators, `..`, or NUL, and `validateAddedWorkflowDestination` ensures the resolved `.md`, `.lock.yml`, and `.yml` destinations stay inside the workflow directory, rejecting symlinked and dangling-symlink destinations. Validation happens before any subprocess executes or any file is written, so failures roll back cleanly.

### Alternatives Considered

#### Alternative 1: Rely on the `--` separator alone, without rejecting `-`-prefixed values

Appending `--` before positionals is sufficient for the Go flag parser in most cases, and it avoids rejecting otherwise-legal names. It was rejected because the separator only protects the subprocess invocation built in this process; downstream commands and future call sites may reconstruct arguments, and an explicit rejection gives MCP callers an immediate, actionable `InvalidParams` error instead of a confusing downstream CLI failure. The documented escape hatch (`./-name`) keeps genuinely legitimate inputs expressible.

#### Alternative 2: Sanitize the `add` name by rewriting it (e.g., stripping separators and `..`)

Normalizing a hostile name into a safe one would never fail the user's request. It was rejected because silent rewriting hides intent, can collide with an unrelated existing workflow, and makes behaviour hard to reason about; strict rejection with a clear `invalid workflow name` error is easier to test and to audit. It also would not have addressed the symlink-escape case, which requires destination-level checks regardless of name shape.

#### Alternative 3: Per-tool, inline validation in each handler

Each MCP handler could keep its own checks next to its existing argument assembly. Rejected because the same mistake had already been made in several handlers; a shared helper pair makes the invariant greppable and makes new tools fail in an obvious, consistent way.

### Consequences

#### Positive
- Caller-controlled values can no longer be reinterpreted as CLI options, closing an argument-injection class across `logs`, `audit`, and `audit diff`.
- `add` destinations are provably contained in the selected workflow directory, including against pre-planted symlinks, with partial writes rolled back.
- The invariant is centralized in `pkg/cli/mcp_tool_arguments.go`, so new MCP tools have an obvious, reusable pattern.
- Regression coverage now exists at both unit (`mcp_tool_arguments_test.go`, `add_destination_security_test.go`) and stdio-integration (`mcp_argument_boundary_integration_test.go`) levels.

#### Negative
- Workflow and run identifiers that legitimately begin with `-` are now rejected and require the documented `./` prefix workaround, a small usability and compatibility cost.
- Every new MCP tool must remember to call both helpers; the contract is convention-enforced rather than type-enforced, so drift is possible.
- Argument assembly order is now load-bearing (all flags before `--`), which is an implicit requirement that a future refactor could silently break.
- Destination validation adds filesystem `Lstat` work on each add path and changes one existing test's expected error from "failed to write destination file" to "invalid workflow name".

#### Neutral
- `buildAuditCommandArgs` no longer embeds run items; callers append them after flags, shifting responsibility to the call site.
- `newLogsToolHandler` gained a `//nolint:largefunc` annotation rather than being decomposed, deferring that refactor.
- The MCP reference documentation (`docs/src/content/docs/reference/gh-aw-as-mcp-server.md`) now states the accepted input shapes, making the contract user-visible.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
