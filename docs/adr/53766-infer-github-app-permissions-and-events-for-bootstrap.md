# ADR-53766: Infer GitHub App Permissions and Events from Resolved Package Workflows in add-wizard Bootstrap

**Date**: 2026-08-19
**Status**: Draft
**Deciders**: Unknown

---

### Context

`gh aw add-wizard` bootstraps a GitHub App for packages that declare a `github-app` action in their `aw.yml` manifest. Previously, the manifest author had to manually list the App's `permissions` and `events` under `github-app`, duplicating information already expressed in each workflow's frontmatter (`permissions:`, `on:`, and `safe-outputs:` handlers). Manifests that omitted these fields silently produced a GitHub App scoped to only `metadata: read` with no subscribed webhook events, so the installed App could not actually perform the actions the package's workflows required (for example, writing issues or reacting to pull requests).

### Decision

We will infer the minimal GitHub App `permissions` and `events` requirements directly from the workflows reachable from a package's `aw.yml`, and merge them with any values still explicitly declared in the manifest. Permission inference reuses the same canonical helpers already used by the `.md` interactive workflow builder (`workflow.ComputeGitHubAppManifestPermissions`, which layers safe-outputs-derived permissions via `ComputePermissionsForSafeOutputs`/`SafeOutputsConfigFromKeys` on top of the raw top-level `permissions` block, keeping the highest scope seen per resource) and normalizes Actions permission keys to GitHub App manifest keys, dropping scopes with no App equivalent. Event inference uses `workflow.NormalizeGitHubAppWebhookEvents` to expand compiler-only triggers (command shorthands, `slash_command`, `label_command`, `reaction`, `status-comment`) to their underlying webhook events, map `pull_request_target` to `pull_request`, and filter out non-webhook triggers (`schedule`, `workflow_dispatch`, `repository_dispatch`). Inference is scoped to only the workflows resolved from the specific package associated with the selected bootstrap profile (`config.Profile.Source`), not the full set of sources passed to add-wizard, so an unrelated standalone workflow installed alongside a package cannot widen that package's App scope. A `--no-config` flag on `gh aw add-wizard` disables this inference entirely, restoring the previous behavior of only applying explicitly declared `permissions`/`events`.

### Alternatives Considered

#### Alternative 1: Require manifest authors to keep declaring permissions/events explicitly

Leave the existing behavior unchanged and instead improve documentation or add a linter warning when `github-app` permissions/events are missing from `aw.yml`. Why not chosen: this still requires every package author to manually keep the manifest's App scopes in sync with the workflows' actual requirements, which is exactly the duplication and silent-drift problem the issue calls out; a warning does not prevent an under-scoped App from being created.

#### Alternative 2: Hand-roll a separate permission/event derivation pass in the CLI package

Implement bespoke frontmatter parsing inside `pkg/cli` to compute permissions and events for `aw.yml` inference, independent of the logic already used for `.md` workflow permission derivation in the interactive builder. Why not chosen: this would create two divergent code paths for deriving GitHub App requirements from workflow frontmatter — one for `.md` workflows and one for `aw.yml` packages — that would need to be kept in sync manually and would be prone to the same normalization bugs (Actions-style keys vs. App manifest keys, compiler-only trigger expansion) independently in each place.

### Consequences

#### Positive
- Package `aw.yml` manifests no longer need to duplicate `permissions`/`events` that are already derivable from their workflows; omitting them no longer silently produces an under-scoped App.
- Permission and event derivation for `aw.yml` packages and `.md` interactive workflow building now share one canonical implementation (`pkg/workflow/github_app_requirements.go`), reducing the risk of divergent or incorrect normalization logic.
- Inference is scoped per-package (via the bootstrap profile's own source), preserving least-privilege when multiple unrelated packages/workflows are installed together.

#### Negative
- Inference adds a workflow-resolution pass (parsing every workflow reachable from the package) to `executeBootstrapProfile`, which is extra work compared to reading a few manifest fields directly, though bounded by the size of the package.
- Authors who want an App scoped more narrowly than what its workflows technically use (e.g., deliberately under-provisioning) must now use `--no-config` and revert to fully manual manifest declarations, since inferred values are additive/merged rather than a ceiling.

#### Neutral
- `aw.yml` manifests may still declare `permissions`/`events` explicitly; declared values are merged with (not replaced by) inferred values, so manifests can supplement inference for cases the inference cannot see (e.g., App requirements unrelated to any workflow trigger).
- The `--no-config` flag preserves an escape hatch for the previous fully-manual behavior without removing any existing manifest fields or schema.

---

*ADR created by adr-writer. Review and finalize before changing status from Draft to Accepted.*
