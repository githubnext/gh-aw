---
title: Package Management (Spec)
description: Normative specification for the aw.yml repository package manifest format.
sidebar:
  order: 321
---

# aw.yml Repository Package Manifest Specification

**Version**: 0.1.0  
**Status**: Draft

## Abstract

This specification defines the `aw.yml` repository package manifest format used by `gh aw` to identify, validate, and install repository packages.

## 1. Introduction

The `aw.yml` manifest describes an installable Agentic Workflow package located either at a repository root or within a nested package folder.

Package references use one of these forms:

- `owner/repo`
- `owner/repo/path/to/package`

The package root is the directory containing `aw.yml`.

## 2. Conformance

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are to be interpreted as described in RFC 2119.

## 3. Manifest location and naming

The canonical manifest filename is `aw.yml`.

## 4. Manifest format

The manifest document MUST be a YAML mapping. Unknown top-level fields MUST be rejected.

### 4.1 Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `manifest-version` | string | No | Manifest format version. Defaults to `"1"`. |
| `min-version` | string | No | Minimum supported `gh-aw` version. |
| `name` | string | Yes | Human-readable package name. |
| `emoji` | string | No | Optional package emoji for display in package metadata. |
| `description` | string | No | Human-readable package description. |
| `files` | array of strings | No | Explicit installable workflow file list. |

### 4.2 `manifest-version`

If omitted, `manifest-version` defaults to `"1"`.

For this version of the format, the only valid value is `"1"`.

### 4.3 `min-version`

If present, `min-version` MUST use the exact `vMAJOR.minor.patch` form, such as:

- `v1.2.3`

If the running compiler version is lower than `min-version`, validation MUST fail.

### 4.4 `name`

`name` MUST be present and MUST be a non-empty string after trimming surrounding whitespace.

### 4.5 `emoji`

If present, `emoji` MUST be a string.

### 4.6 `description`

If present, `description` MUST be a string.

Implementations SHOULD warn if `description` exceeds 255 characters.

### 4.7 `files`

If present, `files` MUST be an array of strings.

Each entry MUST be resolved relative to the package root and MUST match one of the following kinds:

- **Agentic workflow markdown** — the path MUST end in `.md` (case-insensitive) and MUST begin with either `workflows/` or `.github/workflows/`.
- **Raw GitHub Actions YAML** — the path MUST end in `.yml` (case-insensitive) but MUST NOT end in `.lock.yml`. It MUST be a direct child of `.github/workflows/` (no nested subdirectories) and MUST NOT appear under `workflows/`.

Duplicate entries SHOULD be ignored after normalization.

## 5. Installable file resolution

Supported installable paths are:

- `workflows/<name>.md`
- `.github/workflows/<name>.md`
- `.github/workflows/<name>.yml` (raw GitHub Actions YAML; direct children only, `.lock.yml` excluded)

Nested descendants under the markdown directories are also valid when referenced explicitly in `files`. Raw `.yml` action workflows MUST be direct children of `.github/workflows/`; nested `.yml` files are rejected.

Raw `.yml` action workflows are installed verbatim: `gh aw add` copies the file to `.github/workflows/<name>.yml` and performs no frontmatter parsing, no dependency resolution, and no compilation. No `.lock.yml` is produced.

If `files` is present, valid entries are used as the installable workflow set. Invalid entries MUST be ignored with a warning.

If `files` is omitted, or if no valid entries remain after filtering, the implementation MUST attempt discovery under:

- `workflows/`
- `.github/workflows/`

Auto-discovery considers only agentic workflow markdown (`.md`); raw `.yml` action workflows MUST be referenced explicitly in `files` to be installed.

If no installable workflow files are resolved, package validation MUST fail.

## 6. Documentation

Package documentation is `README.md` in the package root.

Examples:

- Repository-root package: `README.md`
- Nested package: `path/to/package/README.md`

If `README.md` is absent, package validation MUST fail.

## 7. Validation and errors

Validation MUST fail for at least the following conditions:

- manifest file not found at the resolved package root;
- malformed YAML;
- top-level document is not a mapping;
- missing or empty `name`;
- unsupported `manifest-version`;
- invalid `min-version`;
- current compiler version is lower than `min-version`;
- unknown top-level fields, including `docs`; or
- missing required `README.md`; or
- no installable workflow files resolved.

Implementations SHOULD emit warnings for at least the following conditions:

- a `files` entry is ignored because it is not a supported installable path; or
- `description` exceeds 255 characters.

## 8. Compile validation

When `gh aw compile` encounters a repository-root `aw.yml`, it validates that manifest before compiling workflows.

A conforming compiler:

- MUST parse and validate the manifest according to this specification;
- MUST fail compilation on manifest errors;
- SHOULD surface warnings as `manifest_warning`; and
- SHOULD surface errors as `manifest_error`.

If JSON output is requested, manifest validation failure still causes an overall compilation failure result.

## 9. Examples

### 9.1 Repository-root package

```yaml
min-version: v0.38.0
name: Repo Assist
emoji: 🤖
description: Friendly repository automation for review and issue triage
files:
  - workflows/review.md                # agentic workflow — compiled on install
  - .github/workflows/nightly-review.md
  - .github/workflows/ci.yml           # raw Actions YAML — copied verbatim
```

### 9.2 Nested package folder

Package reference:

```text
owner/repo/packages/repo-assist
```

Manifest location:

```text
packages/repo-assist/aw.yml
```

Manifest:

```yaml
name: Repo Assist
files:
  - workflows/review.md
```

Documentation file:

```text
packages/repo-assist/README.md
```
