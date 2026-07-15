# ADR-45758: Extend Bootstrap Config Execution to `add` and `add-wizard` Commands

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw` CLI has three workflow-installation commands: `bootstrap` (dedicated setup runner), `add` (non-interactive batch install), and `add-wizard` (interactive install wizard). Package manifests (`aw.yml`) can declare a `config` section listing post-installation steps (set a repository variable, configure a secret, install a GitHub App, etc.). Before this change, only the `bootstrap` command read and acted on those config steps. Users running `gh aw add` or `gh aw add-wizard` received no indication that post-installation setup was required, creating a silent gap: workflows were installed but the configuration steps needed for them to work were never surfaced or executed.

### Decision

We will surface and execute bootstrap config steps from all three install entry points. The `add` command will print a checklist of required manual steps after installation; the `add-wizard` command will execute the steps interactively via the existing `executeBootstrapProfile` runner. Shared helpers (`printBootstrapConfigTODO`, `executeBootstrapConfigForAdd`) are extracted into a new `bootstrap_config.go` file and called from both commands. Concurrently, the `aw.yml` manifest schema is simplified from the nested `bootstrap: { actions: [...] }` structure to a flat top-level `config: [...]` array with strict per-action-type `anyOf` validation, replacing the former `additionalProperties: true` permissiveness.

### Alternatives Considered

#### Alternative 1: Document `gh aw bootstrap` as a required follow-up step

Users would be told in README/docs to run `gh aw bootstrap --repo OWNER/REPO` after `gh aw add`. The existing machinery would remain siloed in the `bootstrap` command.

Not chosen because it relies on users reading external documentation; in practice many users will skip this step, leaving their workflows misconfigured. A nudge within the install flow (a TODO checklist or interactive prompt) is more reliable.

#### Alternative 2: Duplicate bootstrap execution logic into each command

Rather than extracting shared helpers, each command (add, add-wizard) would contain its own copy of the profile-running logic.

Not chosen because duplication creates divergence risk: future changes to bootstrap behavior (new action types, error handling) would need to be applied in three places. Extracting to `bootstrap_config.go` keeps the three commands converged on a single execution path.

### Consequences

#### Positive
- Users installing workflows via `gh aw add` immediately see a TODO checklist of required post-install steps; they no longer need to know about the `bootstrap` subcommand.
- Users running `gh aw add-wizard` get interactive config setup as part of the wizard flow, reducing manual steps after PR creation.
- The new strict `anyOf` schema per action type rejects unknown fields at schema-validation time rather than silently accepting malformed manifests.
- All three commands converge on `executeBootstrapProfile`, so future changes to bootstrap execution propagate automatically.

#### Negative
- **Breaking schema change**: any existing package manifests using the old `bootstrap: { actions: [...] }` structure must be updated to `config: [...]`. There is no migration shim or backward compatibility path.
- The `add` and `add-wizard` commands now carry a secondary responsibility (bootstrap config surfacing) beyond installing workflows, increasing cognitive load for maintainers of those code paths.
- When multiple packages are installed simultaneously and more than one declares a `config` section, bootstrap config is silently skipped for all with only a warning log — the user must install packages separately to apply their config.

#### Neutral
- `ResolveWorkflows` now returns a `BootstrapProfile` field on `ResolvedWorkflows`, widening the surface of the resolution result type.
- The `add_command.go` flow is split from a single `AddWorkflows` call into explicit `ResolveWorkflows` + `AddResolvedWorkflows` stages to allow access to the resolved bootstrap profile before the add result is returned.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
