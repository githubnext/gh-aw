# ADR-45660: Bootstrap Profile Loading from Package Manifests

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw bootstrap` command can install workflows, skills, and agents from an Agentic Workflow (AW) package repository, but it has no mechanism for packages to declare and execute their own prerequisite setup steps. Package authors often need consumers to satisfy environment preconditions (e.g., setting repository variables, configuring secrets, verifying organization ownership, installing GitHub Apps) before the installed workflows can function. Without a first-class way to express these steps, authors resort to README instructions that are easy to miss and hard to validate programmatically.

### Decision

We will embed a `bootstrap` section directly in the repository package manifest (`aw.yml`) that declares a typed, ordered list of bootstrap actions. During `gh aw bootstrap`, after workflows are installed and compiled, the CLI will parse this manifest section into a `repositoryPackageBootstrap` struct, resolve it into a `resolvedBootstrapProfile`, build a plan that reflects whether the actions are already satisfied, and execute any outstanding actions via `executeBootstrapProfile`. The supported action types (`require-owner-type`, `repo-variable`, `repo-secret`, `github-app`, `copilot-auth`, `handoff`) are validated at parse time and enumerated in code; only a single bootstrap profile may be active per bootstrap run.

### Alternatives Considered

#### Alternative 1: Separate bootstrap configuration file (`aw-bootstrap.yml`)

A dedicated file alongside `aw.yml` would keep the main manifest schema smaller and allow the bootstrap spec to evolve independently. However, this splits package authorship across two files, requires a new file-discovery step in the CLI, and offers no ergonomic advantage for consumers. Keeping the bootstrap declaration in `aw.yml` makes the package self-describing and ensures it is always present when the manifest is fetched.

#### Alternative 2: Executable scripts shipped with the package

Allowing packages to ship shell scripts (or similar) that run during bootstrap would support arbitrary setup logic. This was not chosen because it introduces serious security and sandboxing concerns (arbitrary code execution on the consumer's machine), makes the setup steps opaque and hard to audit, and would create platform portability issues across operating systems. A declarative, fixed action vocabulary keeps the surface area auditable and the implementation cross-platform.

### Consequences

#### Positive
- Package authors ship configuration and setup instructions in a single manifest file; consumers get a validated, auditable setup sequence without reading documentation.
- Bootstrap actions are validated at parse time (required fields, valid enum values, recognized action types), so malformed manifests fail early with actionable error messages rather than silently skipping setup.
- The plan step checks whether actions are already satisfied (idempotent planning), so re-running `gh aw bootstrap` on a configured repository is safe and informative.
- The new `normalizeBootstrapRuntime` helper makes the bootstrap runtime fully injectable, enabling unit-test coverage of profile resolution and execution paths without live API calls.

#### Negative
- The manifest YAML schema becomes more complex, increasing the learning curve for package authors who must now understand the bootstrap action vocabulary and required fields per action type.
- Only a single bootstrap profile may be active per bootstrap run; workflows from multiple packages that each define a `bootstrap` section cannot be bootstrapped together in one command, requiring the consumer to split into separate runs.
- The set of supported action types is closed — adding a new action type requires a code change and release cycle, meaning package authors cannot extend the vocabulary without upstream contribution.

#### Neutral
- The `resolveLocalRepositoryPackage` function added in `add_workflow_resolution.go` also propagates the `Bootstrap` field for locally sourced packages, keeping local and remote package resolution symmetric.
- Bootstrap profile execution is integrated into the existing `runBootstrapWithRuntime` flow after compilation, meaning it inherits the same `--plan`, `--yes`, and `--verbose` flag semantics without additional wiring.
- The `ProfileNeedsAction` flag participates in `NeedsMutation`, so the CLI correctly prompts for confirmation when only bootstrap profile actions are pending (no workflow installations required).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
