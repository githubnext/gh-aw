# ADR-53698: Source-to-Destination Mappings in `aw.yml` `includes`

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `aw.yml` package manifest's `includes` field previously accepted only path strings. String entries beginning with `.github/` are resolved relative to the consuming repository root (a rule preserved for backward compatibility). This made it impossible for a distribution repository to keep workflow assets outside its own `.github/workflows/` directory — because any path declared as `.github/workflows/foo.md` pointed to the consumer's repository root — while also preventing assets stored outside `.github/` in the distribution repository from being installed into the consumer's `.github/workflows/`. Package maintainers had no mechanism to declare separate source and destination paths, so assets that should not run in the distribution repository were forced into `.github/workflows/`, where the GitHub Actions runtime would execute them. Issue #52770 filed this limitation explicitly.

### Decision

We will extend the `includes` array to accept object entries (source-to-destination mappings) with required `source` (package-relative path) and `destination` (consuming-repository-root-relative path) fields, plus an optional `kind` discriminator (`agentic-workflow` | `action-workflow`). Mapping sources are always resolved relative to the package root regardless of their prefix; the `.github/` special case that applies to string entries does not apply. Destinations are restricted to direct children of `.github/workflows/`. All three install commands (`gh aw add`, `gh aw add-wizard`, `gh aw update`) share the same mapping semantics through a unified `resolvedPackageInstallable` struct. Validation failures for mapping entries are hard errors that abort before any file is written.

### Alternatives Considered

#### Alternative 1: Change resolution rules for `.github/**` string entries in nested packages

Modify the existing special case so that `.github/**` string entries in a nested package are resolved package-relative rather than repository-root-relative. This would allow `factory/aw.yml` to declare `.github/workflows/foo.md` and have it resolve to `factory/.github/workflows/foo.md`. This was rejected because it breaks backward compatibility: existing manifests depend on the repository-root-relative behavior, and that behavior is explicitly documented (and recorded in ADR-41790).

#### Alternative 2: Add a separate top-level `mappings` field to `aw.yml`

Introduce a new `mappings` key alongside `includes` and `files`. This was rejected because it fragments the authoring surface further — `files` is already deprecated in favor of `includes` — and it prevents authors from mixing string entries and mappings in a single ordered list. Extending `includes` with a union type (string | object) keeps the manifest surface minimal and preserves entry ordering.

### Consequences

#### Positive
- Distribution repositories can now keep workflow assets outside `.github/workflows/`, preventing accidental execution in the distribution context, while still installing those assets into the consumer's `.github/workflows/`.
- Source file names are decoupled from install names: a file at `payload/workflows/reviewer.md` can be installed as `.github/workflows/code-reviewer.md`.
- All existing string-entry semantics are unchanged; existing manifests continue to work without modification.
- Semantic validation (absolute paths, path traversal, symlinks, extension mismatches, unsupported extensions, duplicate destinations) is applied eagerly and fails before any file is written.

#### Negative
- The internal model evolves from plain `[]string` to `[]resolvedPackageInstallable` structs, requiring updates across all callers including tests — a broad but mechanical change.
- `extractManifestIncludes` now returns an error in addition to warnings (compared to the previous warnings-only return), changing the call signature at all call sites.
- Validation failures for mapping entries are hard errors (not soft skips), which is a stricter behavior than the existing warning-and-continue approach for unsupported string entries.

#### Neutral
- The `aw_manifest_schema.json` JSON Schema gains a `oneOf` branch to accommodate object entries alongside string entries.
- `gh aw update` keys installed workflows by destination name and maps them back to source paths, so manifest-scoped source tracking is preserved without additional changes.
- `files` is now marked deprecated in the reference documentation and specification (§4.8); string-form `files` entries continue to work via conversion to `repositoryPackageInclude{Source: p}`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
