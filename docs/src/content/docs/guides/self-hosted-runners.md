---
title: Self-Hosted Runners
description: How to configure the runs-on field to target self-hosted runners in agentic workflows.
---

Use the `runs-on` frontmatter field to target a self-hosted runner instead of the default `ubuntu-latest`.

> [!NOTE]
> Runners must be Linux with Docker support. macOS and Windows are not supported — agentic workflows require container jobs for the [sandbox](/gh-aw/reference/sandbox/).

## runs-on formats

**String** — single runner label:

```aw
---
on: issues
runs-on: self-hosted
---
```

**Array** — runner must have *all* listed labels (logical AND):

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
---
```

**Object** — named runner group, optionally filtered by labels:

```aw
---
on: issues
runs-on:
  group: my-runner-group
  labels: [linux, x64]
---
```

## Sharing configuration via imports

`runs-on` must be set in each workflow — it is not merged from imports. Other settings like `network` and `tools` can be shared:

```aw title=".github/workflows/shared/runner-config.md"
---
network:
  allowed:
    - defaults
    - private-registry.example.com
tools:
  bash: {}
---
```

```aw
---
on: issues
imports:
  - shared/runner-config.md
runs-on: [self-hosted, linux, x64]
---

Triage this issue.
```

## Configuring the detection job runner

When [threat detection](/gh-aw/reference/threat-detection/) is enabled, the detection job runs on the agent job's runner by default. Override it with `safe-outputs.threat-detection.runs-on`:

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
safe-outputs:
  create-issue: {}
  threat-detection:
    runs-on: ubuntu-latest
---
```

This is useful when your self-hosted runner lacks outbound internet access for AI detection, or when you want to run the detection job on a cheaper runner.

## Related documentation

- [Frontmatter](/gh-aw/reference/frontmatter/#run-configuration-run-name-runs-on-timeout-minutes) — `runs-on` syntax reference
- [Imports](/gh-aw/reference/imports/) — importable fields and merge semantics
- [Threat Detection](/gh-aw/reference/threat-detection/) — detection job configuration
- [Network Access](/gh-aw/reference/network/) — configuring outbound network permissions
- [Sandbox](/gh-aw/reference/sandbox/) — container and Docker requirements
