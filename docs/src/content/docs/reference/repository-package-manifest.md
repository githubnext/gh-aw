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

For the normative file-format definition, see the [aw.yml repository package manifest specification](/gh-aw/reference/repository-package-manifest-specification/).

## Example

```yaml
min-version: v0.38.0
name: Repo Assist
description: Friendly repository automation for review and issue triage
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
```

## Quick reference

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `manifest-version` | string | No | Current supported value: `"1"`. Defaults to `"1"` when omitted. |
| `min-version` | string | No | Minimum compatible `gh aw` version. Must use the exact `vMAJOR.minor.patch` form, such as `v0.38.0`. |
| `name` | string | Yes | Human-readable package name. Must be non-empty after trimming whitespace. |
| `description` | string | No | Optional package description. `gh aw add` warns when it exceeds 255 characters. |
| `files` | array of strings | No | Package-root-relative markdown files under `workflows/` or `.github/workflows/`. |

## Documentation

Package documentation is `README.md` in the package root.

- Repository-root package docs: `README.md`
- Nested package docs: `path/to/package/README.md`

The manifest does not support a `docs` field.
Missing `README.md` causes package validation to fail.

## Installable workflows

If `files` is present, valid entries are used as the install bundle.

If `files` is omitted or contains no valid installable paths, `gh aw add` scans:

- `workflows/`
- `.github/workflows/`

For nested packages, those paths are resolved relative to the package root.

The embedded JSON schema source of truth lives in `pkg/parser/schemas/aw_manifest_schema.json`.
