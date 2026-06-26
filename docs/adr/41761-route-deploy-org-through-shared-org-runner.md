# ADR-41761: Route `deploy --org` Through Shared Org Runner

**Date**: 2026-06-26
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw deploy` command previously supported only single-repository targets via `--repo`. Other multi-repo commands (e.g., `lock`, `create`) already route their org-mode execution through a shared pipeline in `org_runner.go` that handles repository discovery, glob filtering, interactive confirmation, per-repo apply, and rate-limit waiting. Adding org-mode support to `deploy` required a decision: implement the org execution path inline in the deploy command, or delegate to the shared runner by providing deploy-specific callbacks. Duplicating the pipeline logic would diverge from the pattern used by other commands and create two code paths to maintain for the same org-execution concerns.

### Decision

We will route `deploy --org` operations through `runCommandForOrg` in `org_runner.go` by introducing `pkg/cli/deploy_org.go` with a thin adapter (`runDeployForOrg`) that supplies deploy-specific callbacks (search, report, apply). The `deploy` command gains `--org`, `--repos`, and `--yes` flags, and `runDeployCommand` dispatches to the org path when `--org` is provided. This approach reuses the shared org execution contract for consistency with all other org-mode commands.

### Alternatives Considered

#### Alternative 1: Inline org logic in `deploy_command.go`

Implement repository discovery, glob filtering, confirmation, and per-repo apply directly inside `runDeployCommand` without introducing a shared-runner dependency. This would be the simplest code path to follow in isolation, but it would duplicate all org-execution concerns already handled by `runCommandForOrg` (CI detection, `--yes` enforcement, rate-limit wait, failure aggregation). Any bug fix or behavioral change to org execution would need to be applied in multiple places.

#### Alternative 2: New standalone `deploy-org` subcommand

Introduce a separate top-level subcommand (e.g., `gh aw deploy-org`) instead of adding `--org` to the existing `deploy` command. This avoids flag-dispatch branching inside `runDeployCommand` but splits the deploy surface across two commands, complicating discovery for users and requiring separate documentation, completion, and validation logic.

### Consequences

#### Positive
- Consistent behavior across all org-mode commands: discovery, confirmation, rate-limit handling, and failure reporting share one implementation.
- Adding org-mode to future commands requires only a small adapter file with deploy-specific callbacks, not re-implementing the full pipeline.
- CI safety (`--yes` required in CI) is enforced by the shared runner automatically.

#### Negative
- `deploy --org` now inherits all future changes to `runCommandForOrg`; behavioral changes to the shared runner (e.g., new confirmation semantics) will affect deploy without an explicit deploy-side change.
- The callback contract (`orgRunCallbacks`, `createPR`/`createIssue` booleans) adds indirection that makes the deploy org flow harder to trace end-to-end compared to a self-contained implementation.

#### Neutral
- The `runDeployFn` and `runDeployForOrgFn` package-level variables introduced for test injection are consistent with the pattern used elsewhere in the `cli` package.
- Repository search reuses `searchOrgLockWorkflowRepos`, coupling deploy org discovery to the lock command's search logic; divergence in search criteria would require splitting this function.

---
