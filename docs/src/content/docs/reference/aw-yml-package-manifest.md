---
title: Package Manifest (aw.yml)
description: Reference for the aw.yml package manifest used by gh aw add and gh aw compile.
sidebar:
  order: 320
---

Use `aw.yml` to describe an installable agentic workflow package.
`gh aw add` uses this manifest when installing packages, and
`gh aw compile` validates repository-root manifests before compilation.

For the normative file-format definition, see the
[Package Management (Spec)](/gh-aw/reference/repository-package-manifest-specification/).

## Package reference formats

Repository references support two forms:

- `OWNER/REPO`
- `OWNER/REPO/PATH/TO/PACKAGE`

The package root is the folder that contains `aw.yml`.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `manifest-version` | string | No | Current supported value: `"1"`. Defaults to `"1"` when omitted. |
| `min-version` | string | No | Minimum compatible `gh aw` version in `vMAJOR.minor.patch` form, such as `v0.38.0`. |
| `name` | string | Yes | Human-readable package name. Must be non-empty after trimming whitespace. |
| `emoji` | string | No | Optional package emoji for display in package metadata. |
| `description` | string | No | Optional package description. `gh aw add` warns when it exceeds 255 characters. |
| `files` | array of strings | No | Package-root-relative paths. Agentic markdown workflows under `workflows/` or `.github/workflows/`; raw GitHub Actions YAML (`.yml`) is also accepted as direct children of `.github/workflows/`. |

## Installable workflows

If `files` is present, valid entries become the install bundle. Two entry kinds are supported:

- **Agentic workflow markdown** — paths ending in `.md` under `workflows/` or `.github/workflows/`. `gh aw add` compiles these to lock files and fetches their dependencies.
- **Raw GitHub Actions YAML** — paths ending in `.yml` (but not `.lock.yml`) that are direct children of `.github/workflows/`. `gh aw add` copies these verbatim to `.github/workflows/<name>.yml` with no frontmatter processing, no dependency fetch, and no compilation. Nested subdirectories under `.github/workflows/` and `.yml` files under `workflows/` are not accepted.

If `files` is omitted, or no valid entries remain after filtering,
`gh aw add` discovers installable markdown files under:

- `workflows/`
- `.github/workflows/`

If no installable workflow files are resolved, validation fails.

## Package documentation

Package documentation must be `README.md` at the package root.
The manifest does not support a `docs` field.

Missing `README.md` causes package validation to fail.

## Example

```yaml
manifest-version: "1"
min-version: v0.38.0
name: Repo Assist
emoji: 🤖
description: Friendly repository automation for review and issue triage
files:
  - workflows/review.md                # agentic workflow — compiled on install
  - .github/workflows/nightly-review.md
  - .github/workflows/ci.yml           # raw Actions YAML — copied verbatim
```
