---
title: aw.yml package manifest
description: Reference for the aw.yml package manifest used by gh aw add and gh aw compile.
sidebar:
  order: 320
---

Use `aw.yml` to describe an installable agentic workflow package.
`gh aw add` uses this manifest when installing packages, and
`gh aw compile` validates repository-root manifests before compilation.

For the normative file-format definition, see the
[aw.yml repository package manifest specification](/gh-aw/reference/repository-package-manifest-specification/).

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
| `files` | array of strings | No | Package-root-relative markdown files under `workflows/` or `.github/workflows/`. |

## Installable workflows

If `files` is present, valid entries become the install bundle.

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
  - workflows/review.md
  - .github/workflows/nightly-review.md
```
