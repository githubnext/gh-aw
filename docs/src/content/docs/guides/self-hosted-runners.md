---
title: Self-Hosted Runners
description: How to configure agentic workflows to run on self-hosted runners using the runs-on field, and how to share runner configuration across workflows with a shared import file.
sidebar:
  order: 5
---

By default, agentic workflows run on GitHub-hosted `ubuntu-latest` runners. You can change this using the `runs-on` frontmatter field to target self-hosted runners — for example, to access private networks, use specialized hardware, or manage your own runner pools.

## Prerequisites

Your self-hosted runner must run Linux. macOS and Windows runners are not supported because agentic workflows require container support for the [sandbox](/gh-aw/reference/sandbox/).

## Configuring runs-on

The `runs-on` field accepts three formats.

**Simple label** — the most common form, matching any runner that has the given label:

```aw
---
on: issues
runs-on: self-hosted
---

Triage this issue.
```

**Label array** — GitHub Actions picks the first available runner that matches any label in the list. Useful for high-availability pools with multiple runner groups:

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
---

Triage this issue.
```

**Group and labels object** — targets a named runner group, optionally filtered by labels. Useful when your organization uses runner groups to organize pools by team or environment:

```aw
---
on: issues
runs-on:
  group: my-runner-group
  labels: [linux, x64]
---

Triage this issue.
```

> [!NOTE]
> The `runs-on` field only applies to the main agent job. Custom jobs defined in `jobs:` configure their own `runs-on` field independently.

## Sharing Runner Configuration with Imports

When multiple workflows share the same runner requirements, define the `runs-on` configuration once in a shared file and import it. This avoids repeating runner labels across every workflow.

Create a shared configuration file, for example `.github/workflows/shared/runner-config.md`:

```aw
---
runs-on: [self-hosted, linux, x64]
---
```

> [!NOTE]
> Shared files without an `on:` field are treated as shared workflow components and are not compiled into GitHub Actions workflows on their own.

Then import this file in any workflow that needs it:

```aw
---
on: issues
imports:
  - shared/runner-config.md
permissions:
  issues: write
---

Triage this issue and assign appropriate labels.
```

The compiler merges the `runs-on` field from the imported file into the workflow. The main workflow's `runs-on` always takes precedence over imported values — so you can import a default runner and override it per-workflow when needed:

```aw
---
on: issues
imports:
  - shared/runner-config.md
runs-on: [self-hosted, linux, arm64]  # overrides the imported value
permissions:
  issues: write
---

Triage this issue on the ARM runner.
```

## Runner Requirements

Self-hosted runners must meet the following requirements for agentic workflows:

- **Linux only** — Docker container support is required for the agent sandbox
- **Docker** — the runner must have Docker installed and running
- **Network access** — the runner needs outbound internet access unless you configure a fully air-gapped setup with appropriate [network permissions](/gh-aw/reference/network/)

> [!WARNING]
> macOS (`macos-*`) and Windows (`windows-*`) runners are not supported. Agentic workflows use Docker containers for the agent execution sandbox, which requires Linux.

## Related Documentation

- [Frontmatter Reference](/gh-aw/reference/frontmatter/#run-configuration-run-name-runs-on-timeout-minutes) — complete reference for `runs-on` and other run configuration fields
- [Reusing Workflows](/gh-aw/guides/packaging-imports/) — how to create and use shared import files
- [Imports Reference](/gh-aw/reference/imports/) — import merge semantics and path formats
- [Sandbox](/gh-aw/reference/sandbox/) — how the agent execution sandbox works
- [Network Access](/gh-aw/reference/network/) — configuring network permissions for workflows
