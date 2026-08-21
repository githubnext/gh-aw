---
title: Repository package manifest
description: Reference for the aw.yml manifest used by gh aw add and gh aw compile.
sidebar:
  order: 320
---

Use `aw.yml` to describe an installable workflow package for `gh aw add`.

- Repository root packages use `owner/repo`
- Nested packages use `owner/repo/path/to/package`
- `gh aw compile` validates a repository-root `aw.yml` before compiling workflows

For the normative file-format definition, see the [Package Management (Spec)](/gh-aw/specs/repository-package-manifest-specification/).

## Example

```yaml
min-version: v0.38.0
name: Repo Assist
emoji: 🤖
description: Friendly repository automation for review and issue triage
includes:
  - workflows/review.md                # agentic workflow — compiled on install
  - source: templates/triage.yml       # explicit source/destination mapping
    destination: .github/workflows/triage.yml
  - skills/code-review                 # skill directory (must contain SKILL.md)
  - agents/reviewer.md                 # agent file
  - .github/workflows/ci.yml           # raw Actions YAML — copied verbatim
resources:
  - source: templates/bug.yml
    destination: .github/ISSUE_TEMPLATE/bug.yml
  - source: policy/controls.json
    destination: .github/aw/policy/controls.json
```

## Quick reference

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `manifest-version` | string | No | Current supported value: `"1"`. Defaults to `"1"` when omitted. |
| `min-version` | string | No | Minimum compatible `gh aw` version. Must use the exact `vMAJOR.minor.patch` form, such as `v0.38.0`. |
| `name` | string | Yes | Human-readable package name. Must be non-empty after trimming whitespace. |
| `emoji` | string | No | Optional package emoji for display in package metadata. |
| `description` | string | No | Optional package description. `gh aw add` warns when it exceeds 255 characters. |
| `includes` | array of strings or mappings | No | Explicit install entries. Strings use path conventions; mappings declare package-relative `source` and repository-root-relative `destination` for workflow installs under `.github/workflows/`. |
| `resources` | array of mappings | No | Declarative repository assets copied as-is from package content to allowlisted destinations. |
| `files` | array of strings | No | Deprecated alias. Use `includes` instead. |

## Documentation

Package documentation is `README.md` in the package root.

- Repository-root package docs: `README.md`
- Nested package docs: `path/to/package/README.md`

The manifest does not support a `docs` field.
Missing `README.md` causes package validation to fail.

When `files` is present, `gh aw add` emits a deprecation warning and automatically codemods values to equivalent `includes` entries in memory for resolution.

## Installable workflows

If `includes` is present, valid entries are used as the install bundle. Supported entry kinds:

- **Agentic workflow markdown** — paths ending in `.md` under `workflows/`, `agentic-workflows/`, or `.github/workflows/`. Compiled to lock files on install.
- **Raw GitHub Actions YAML** — paths ending in `.yml` (but not `.lock.yml`) that are direct children of `.github/workflows/`. Copied verbatim with no compilation or dependency fetching. `.yml` files under `workflows/` and nested subdirectories under `.github/workflows/` are not accepted.
- **Skills** — directory paths under `skills/` or `.github/skills/` that contain `SKILL.md`.
- **Agents** — `.md` files under `agents/` or `.github/agents/`.

Mapping entries in `includes` can install workflow files from package-relative `source` paths into explicit repository-root `destination` paths under `.github/workflows/`.

## Resources

`resources` installs inert repository assets from package content:

- `source` is always package-root-relative.
- `destination` is always repository-root-relative.
- Allowed destination namespaces:
  - `.github/ISSUE_TEMPLATE/*.{yml,yaml}` (direct children only; nested subdirectories such as `.github/ISSUE_TEMPLATE/bug_report/bug.yml` fail manifest validation)
  - `.github/CODEOWNERS`
  - `.github/aw/**`

`gh aw add` rejects duplicate (including case-insensitive duplicate) resource destinations, path traversal, and non-regular local resource sources. Installed resources are tracked with package-scoped ownership metadata in `.github/aw/packages/*.json`.

If `includes` is omitted or contains no valid workflow paths, `gh aw add` scans:

- `workflows/`
- `.github/workflows/`

For nested packages, those paths are resolved relative to the package root.

Packages can still be valid without workflows when they declare installable `resources`, skills, or agents.

The embedded JSON schema source of truth lives in `pkg/parser/schemas/aw_manifest_schema.json`.
