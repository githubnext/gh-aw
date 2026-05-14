---
title: Repository package manifest
description: Schema reference for the repository-root aw.yml manifest used by gh aw add and validated by gh aw compile.
sidebar:
  order: 320
---

Use a repository-root `aw.yml` file to describe an installable workflow package for `gh aw add owner/repo`. When present, `gh aw compile` also validates the manifest before compiling workflows.

The canonical file name is `aw.yml`. The legacy aliases `agents.yml` and `agents.yaml` are still read for compatibility, but `aw.yml` wins when multiple aliases exist.

## Example

```yaml
schema-version: "1"
min-version: v0.38.0
name: Repo Assist
description: Friendly repository automation for review and issue triage
docs: docs/overview.md
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
```

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `schema-version` | string | No | Manifest schema version. Current value: `"1"`. Omitted manifests default to the current schema for backward compatibility. |
| `min-version` | string | No | Minimum compatible `gh aw` version. Must be a semantic version such as `v0.38.0` or `0.38.0`. |
| `name` | string | Yes | Human-readable package name. Must be non-empty after trimming whitespace. |
| `description` | string | No | Package description. Long descriptions are allowed, but `gh aw add` warns when the text exceeds 255 characters. |
| `docs` | string | No | Markdown file shown as the package documentation. Must end in `.md`. |
| `files` | array of strings | No | Explicit installable workflow files. Every entry must be a markdown file under `workflows/` or `.github/workflows/`. |

## Validation rules

- Unknown top-level fields are rejected.
- `schema-version` currently supports only `"1"`.
- `min-version` is checked against the running compiler version with semantic version comparison.
- If `files` is omitted, `gh aw add` falls back to scanning `workflows/` and `.github/workflows/` for installable markdown workflows.
- If `docs` is omitted, `gh aw add` probes `docs/<package-name>.md`, then `docs/<workflow-basename>.md`, then falls back to the first installable workflow file.

The embedded JSON schema source of truth lives in `pkg/parser/schemas/aw_manifest_schema.json` in this repository.
