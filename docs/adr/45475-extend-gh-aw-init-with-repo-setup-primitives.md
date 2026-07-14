# ADR-45475: Extend `gh aw init` with Repo Setup Primitives via Planner/Executor Pattern

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: Unknown

---

### Context

`gh aw init` only handled in-directory initialization (applying markers, skills, MCP config) but had no mechanism to attach to a remote repository, clone one locally, or create a new one. Users who wanted a complete first-run setup had to run multiple commands manually — `gh repo create`, `git clone`, then `gh aw init` — with no guidance and no idempotency guarantee. This fragmented flow blocked automation and agent-driven onboarding. The need for a single entry-point command that can resolve remote state, manage the local checkout, and apply initialization markers drove this extension.

### Decision

We will extend the existing `gh aw init` command with a new set of flags (`--repo`, `--dir`, `--create`, `--private`, `--plan`, `--yes`, `--json`, `--require-owner-type`) backed by a dedicated `PlanAndExecuteRepoSetup` function that implements a **planner/executor split**: `buildPlan()` resolves remote and local state without side effects, and `executePlan()` runs mutations only after an optional interactive confirmation. The `--plan` flag enables a dry-run preview with no mutations. Stable status constants (`attached`, `created`, `cloned`, `initialized`, `blocked`, `noop`) are returned in a `RepoSetupResult` struct to support downstream composition by other `gh aw` subcommands.

### Alternatives Considered

#### Alternative 1: New Dedicated Subcommands (`gh aw clone`, `gh aw create`)

Add separate top-level subcommands (`gh aw clone OWNER/REPO` and `gh aw create OWNER/REPO`) instead of extending `init`. Each command would have a single, narrow responsibility.

This was not chosen because it would leave the initialization step disconnected from the lifecycle steps (create → clone → init). Users would still need to chain commands manually, and the tools would have no shared contract for status reporting or downstream composition. A single `gh aw init --repo` invocation that handles the entire lifecycle is more discoverable and automation-friendly.

#### Alternative 2: Inline State Resolution and Mutations Directly in the `init` Command Handler

Skip the planner/executor abstraction. Resolve remote state, classify the local directory, and run mutations all within a single procedural function in the `RunE` handler.

This was not chosen because inline logic makes dry-run mode (`--plan`) impossible without duplicating the resolution logic into a separate preview path. The planner/executor split also makes unit testing straightforward: `buildPlan` can be tested with controlled inputs without triggering any mutations or requiring network access.

#### Alternative 3: `--repo` as a Separate `gh aw setup` Command

Add `gh aw setup` as a new top-level command rather than extending `init`, keeping `init` focused on in-directory marker application.

Not chosen because it would duplicate the flag surface and add a cognitive burden for users who expect one command (`init`) to fully prepare a repository. The PR description explicitly identifies first-run fragmentation as the problem, and a single command solves it most directly.

### Consequences

#### Positive
- Users have a single idempotent command that handles the complete first-run lifecycle: create remote → clone → apply markers.
- Dry-run mode (`--plan`) is first-class, enabled by the planner/executor split without duplicating logic.
- Stable `RepoSetupStatus` constants provide a machine-readable contract for downstream composition by `gh aw add`, `gh aw auth`, and scripted workflows.
- The `--json` flag allows CI pipelines to parse results without screen-scraping.

#### Negative
- The `init` command now has dual responsibility: it manages both the repository lifecycle (remote + local checkout) and the initialization marker application, increasing its conceptual surface area.
- The planner/executor pattern requires careful state threading (`dirStateType`, `initStateType`, `RepoSetupResult`) through multiple function boundaries, raising the cost of future changes to the flow.

#### Neutral
- The `--require-owner-type` flag adds an org/user policy enforcement hook that is silently skipped when the GitHub API is unreachable, making it a best-effort guardrail rather than a hard gate.
- The `createRemoteRepo` function uses `--source .` when calling `gh repo create`, which assumes the current directory is a valid git repo; this is a constraint future callers must be aware of.
- The init marker detection logic (`.gitattributes` entry + secondary marker) is now formalized as a stable internal API (`detectInitMarkers`) that other code may depend on.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
