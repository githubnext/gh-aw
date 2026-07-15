# ADR-45524: Introduce `bootstrap` Command for Idempotent Agentic Workflow Repository Setup

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: mnkiefer

---

### Context

Setting up a repository for agentic workflows required users to run multiple CLI commands in sequence: create or clone the repo, run `gh aw init` for marker files, add workflow sources with `gh aw add`, and compile with `gh aw compile`. This multi-step process was not idempotent, not CI-safe, and required external scripting glue. Agentic CI pipelines especially need a single command that can run unattended (with `--yes`) and safely skip already-completed steps without side effects.

A companion need also emerged: other setup-oriented commands (and future tooling) need access to auth verification and repository state checks as reusable primitives, without being forced to run a full bootstrap. Exposing these as a hidden `doctor` subcommand tree allows both scripted inspection and composition.

### Decision

We will add two CLI commands — `bootstrap` and a hidden `doctor` command tree — backed by a shared `setupRepositoryRuntime` struct. `bootstrap` orchestrates the full repository lifecycle (auth check → optional repo create → clone or attach → init markers → add workflows → compile) as a single idempotent, plan-then-apply operation. `doctor` exposes the auth check (`doctor auth`) and repository state inspection (`doctor repo`) as lightweight standalone subcommands. Both commands reuse the same runtime primitives, injected via struct fields to keep them fully testable.

`bootstrap` follows a hybrid DX contract: guided when needed, automated when possible. In an interactive terminal, it prompts only for values that are missing and required for the selected bootstrap actions. When values already exist or are provided up front, it proceeds without re-prompting. In non-interactive contexts, missing required values are treated as errors unless the action can be safely skipped because the repository is already configured or the selected workflows support an existing auth path. Browser launch is reserved for the GitHub App creation and installation flow; supplying existing GitHub App credentials keeps the flow terminal-only.

### Alternatives Considered

#### Alternative 1: Shell script / external tooling

Users could compose `gh repo create`, `gh repo clone`, `gh aw init`, `gh aw add`, and `gh aw compile` in a shell script. This was considered because it requires no new code. It was rejected because shell scripts are brittle across platforms (Windows, CI images), are not idempotent by default, lack the plan-and-confirm UX, and require every consumer to reimplement the same error-handling and skip logic — defeating the goal of a single authoritative setup path.

#### Alternative 2: Extend `init` with repo-lifecycle flags

Adding `--create-repo`, `--clone`, and `--source` flags to the existing `gh aw init` command would avoid a new top-level command. It was rejected because `init` has a well-defined scope (writing repository marker files), and mixing repository creation/cloning into it would create a single-responsibility violation. It would also make the existing `init` command's interface more confusing for users who only want to reinitialize marker files on an already-cloned repository.

### Consequences

#### Positive
- Single idempotent entry point for bootstrapping agentic workflow repositories from scratch or attaching to existing checkouts.
- CI-safe via `--yes` flag; `--plan` provides a dry-run mode that prints the exact steps without executing them.
- Shared `setupRepositoryRuntime` struct is reusable by future setup-oriented commands without code duplication.
- Full unit and integration test coverage via injected runtime, with a fake `gh` binary for integration tests.

#### Negative
- Adds two new commands (`bootstrap`, hidden `doctor`) to an already large CLI surface, increasing the maintenance and documentation burden.
- The plan-then-apply pattern makes two passes over auth and repository state, adding latency in the common case where no changes are needed.
- The hybrid guided/automatic flow increases branching in bootstrap, especially around TTY detection, missing-input handling, and the GitHub App browser flow.

#### Neutral
- The hidden `doctor` command intentionally does not perform mutations; it is read-only by design. This means users who want a combined check-and-act workflow must use `bootstrap`.
- Engine-specific init marker detection (Copilot vs. other engines) is baked into `expectedBootstrapInitMarkers`, so adding a new engine requires updating that function.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
