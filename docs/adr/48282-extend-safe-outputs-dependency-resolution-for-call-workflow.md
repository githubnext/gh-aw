# ADR-48282: Extend safe-outputs Dependency Resolution to Support call-workflow Workers

**Date**: 2026-07-27
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent

---

### Context

The `gh aw add` command resolves and downloads remote workflow dependencies during installation. Before this change, only `safe-outputs.dispatch-workflow` workers were automatically fetched and compiled; `safe-outputs.call-workflow` workers were silently ignored. Orchestrators that declared call-workflow workers would fail at runtime because the required `.lock.yml` files never existed on disk. A second, unrelated bug caused the bootstrap wizard to split `config` steps into pre-install and post-install phases based on action type, breaking the declared ordering from `aw.yml` manifests.

### Decision

We will extend the `fetchAllRemoteDependencies` pipeline to fetch and compile `call-workflow` workers using the same two-phase approach already applied to `dispatch-workflow`: (1) `fetchAndSaveRemoteCallWorkflows` fetches workers declared directly in frontmatter during remote `gh aw add`, and (2) `fetchAndSaveCallWorkflowsFromParsedFile` fetches import-derived workers discovered only after the compiler merges all imports. We will extract shared helpers (`extractWorkflowNamesFromSafeOutputs`, `compileSafeOutputsWorkflowDependencies`, `safeOutputsWorkflowNamesForCompilation`, `extractSafeOutputsNamesFromFrontmatter`) to eliminate duplication between the two dependency types. We will also remove the pre-install/post-install config phase split so all config steps execute post-install in exact declared order.

### Alternatives Considered

#### Alternative 1: Duplicate dispatch-workflow logic without shared abstractions

Implement `fetchAndSaveRemoteCallWorkflows` and `compileCallWorkflowDependencies` as full copies of their dispatch-workflow counterparts, with only the key name changed. This avoids abstraction overhead and keeps each type self-contained.

Not chosen because the two codepaths are structurally identical; maintaining two divergent copies would cause drift (e.g., a bug fix applied to one but not the other). The existing dispatch-workflow code was already showing signs of this duplication.

#### Alternative 2: Add call-workflow fetching outside the existing dependency pipeline

Register a separate hook or post-processing step outside `fetchAllRemoteDependencies`, invoked independently from the orchestrator add path.

Not chosen because `fetchAllRemoteDependencies` is the canonical, ordered integration point for all dependency types. Adding a separate path would bypass existing source-conflict detection, tracker integration, and the verbose-logging conventions already present in that function, and would create a second ordering assumption about when workers must be present.

#### Alternative 3: Require users to pre-declare call-workflow workers as explicit `resources:` entries

Document that call-workflow workers must be listed under the `resources:` frontmatter key, which is already fetched. No code change needed.

Not chosen because it breaks the encapsulation guarantee of the `safe-outputs.call-workflow` config block, which is supposed to be self-describing. It would require all existing and future orchestrator authors to add redundant `resources:` declarations for every worker.

### Consequences

#### Positive
- `call-workflow` workers are now fetched and compiled automatically on `gh aw add`, matching the existing `dispatch-workflow` behavior and eliminating a class of silent runtime failures.
- Shared helper functions (`compileSafeOutputsWorkflowDependencies`, `safeOutputsWorkflowNamesForCompilation`, `extractSafeOutputsNamesFromFrontmatter`) reduce duplication between the two dependency types and make adding future `safe-outputs` dependency kinds straightforward.
- Config bootstrap steps now run in the exact declared order from the manifest, fixing a user-visible sequencing bug where pre-install type filtering silently dropped steps of unexpected types.
- The compiler-parse fallback to raw frontmatter extraction is now shared and consistently handles both array and map config forms for all safe-outputs dependency types.

#### Negative
- All config steps — including steps that previously ran pre-install (e.g., `require-owner-type`, `github-app`) — now execute post-install. If any of these steps is truly order-sensitive relative to the installation, that constraint is no longer enforced; failures will surface later in the wizard flow rather than early.
- `fetchAndSaveCallWorkflowsFromParsedFile` is best-effort and silently skips workers when the repo slug is missing or the file cannot be parsed; there is no user-visible warning in non-verbose mode when workers are skipped this way.
- The shared `compileSafeOutputsWorkflowDependencies` abstraction uses a function pointer (`namesFunc`) to select the dependency list, which adds a layer of indirection that makes the call graph less obvious when reading individual callers.

#### Neutral
- The `extractDispatchWorkflowNames` function is preserved as a thin wrapper over the new shared `extractWorkflowNamesFromSafeOutputs`, maintaining backward compatibility for any callers that reference it by name.
- Tests for `bootstrapProfileAddWizardPreInstall` / `bootstrapProfileAddWizardPostInstall` are replaced with a more general `TestBootstrapProfileFilterActions` that tests the underlying `filterBootstrapProfileActions` directly, since the wizard-phase helpers are removed.
- Dead-code removal of `bootstrapProfileAddWizardPreInstall` and `bootstrapProfileAddWizardPostInstall` reduces the public surface of `bootstrap_config.go`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
