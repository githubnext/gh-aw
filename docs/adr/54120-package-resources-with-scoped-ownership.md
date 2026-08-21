# ADR-54120: Package Resources with Scoped Ownership

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Repository packages (`aw.yml` manifests) could install workflows, skills, and agents, but had no mechanism to bundle supplementary repository assets such as Issue Forms (`.github/ISSUE_TEMPLATE/*.yml`), `CODEOWNERS`, or policy files under `.github/aw/`. Consumers who needed these files had to copy them manually alongside `gh aw add`, breaking the self-contained package installation model. The gap was tracked in issue #52769.

### Decision

We will introduce a `resources:` field in the `aw.yml` package manifest. Each resource entry declares a package-relative `source` and a repository-root-relative `destination`. Destinations are restricted to an explicit allowlist (`.github/ISSUE_TEMPLATE/*.yml|*.yaml`, `.github/CODEOWNERS`, `.github/aw/**`). Resources are copied as inert content (no compilation, no secret injection). For every package installation, SHA-256-based ownership records are written under `.github/aw/packages/*.json`, and updates refuse to overwrite locally drifted files unless `--force` is passed. Stale resource files are removed on `gh aw update` when they are dropped from the manifest and unchanged since installation.

### Alternatives Considered

#### Alternative 1: Extend the existing `includes` / `files` field

The `includes` field already supports explicit source-to-destination mappings for installable workflow files. Resources could be added there with a special flag or naming convention distinguishing inert-copy from compiled-workflow semantics.

Rejected because mixing the two installation modes in one field creates ambiguity: `includes` entries go through compilation and `.md`-to-workflow translation steps that are inappropriate for raw YAML or JSON assets. Adding a discriminant flag would complicate the schema and parser without a natural extension point.

#### Alternative 2: Document-only / manual copy instructions

Packages could document supplementary files in their README and expect users to copy them manually. This preserves simplicity in the CLI.

Rejected because it breaks the single-command (`gh aw add`) installation promise and requires package consumers to know which files to copy, defeating the purpose of a manifest-driven package system.

### Consequences

#### Positive
- Packages can ship a complete, self-contained repository setup — workflows, skills, agents, issue templates, CODEOWNERS, and policy files — in one `gh aw add` invocation.
- SHA-256 ownership records prevent silent overwrites of locally modified files, making update safety explicit and auditable.
- The allowlist of valid destinations prevents packages from writing to arbitrary repository paths, limiting the blast radius of a malicious or misconfigured package.

#### Negative
- The destination allowlist (ISSUE_TEMPLATE, CODEOWNERS, `.github/aw/**`) must be maintained as product needs evolve; adding new allowed namespaces requires a code change and specification update.
- Packages that declare only `resources:` (no workflows/skills/agents) are now valid, which changes the emptiness check in `resolveRepositoryPackage` and may surface unexpected edge cases in tooling that assumes at least one workflow is present.

#### Neutral
- A new `IsPackageResourceFile` discriminant is added to `WorkflowSpec` and `ResolvedWorkflow`; bootstrap profile helpers skip resource files when inferring Copilot Auth and GitHub App permission requirements, consistent with how skill and agent files are handled.
- The `gh aw remove` command gains cleanup logic that removes package-owned resource files when the last workflow from a package is removed and the files are still unchanged since installation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
