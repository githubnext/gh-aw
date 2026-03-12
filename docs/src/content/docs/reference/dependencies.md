---
title: APM Dependencies
description: Install and manage APM (Agent Package Manager) packages in your agentic workflows, including skills, prompts, instructions, agents, hooks, and plugins.
sidebar:
  order: 330
---

The `dependencies:` frontmatter field installs [APM (Agent Package Manager)](https://microsoft.github.io/apm/) packages before workflow execution. When present, the compiler packs dependencies in the activation job and unpacks them in the agent job for faster, deterministic startup.

APM manages AI agent primitives such as skills, prompts, instructions, agents, hooks, and plugins (including the Claude `plugin.json` specification). Packages can depend on other packages and APM resolves the full dependency tree.

## Reproducibility and governance

APM lock files (`apm.lock`) pin every package to an exact commit SHA, so the same versions are installed on every run. Lock file diffs appear in pull requests and are reviewable before merge, giving teams and enterprises a clear audit trail and the ability to govern which agent context is in use. See the [APM governance guide](https://microsoft.github.io/apm/enterprise/governance/) for details on policy enforcement and access controls.

## Format

### Simple array format

```yaml wrap
dependencies:
  - microsoft/apm-sample-package
  - github/awesome-copilot/skills/review-and-refactor
  - anthropics/skills/skills/frontend-design
```

### Object format with options

```yaml wrap
dependencies:
  packages:
    - microsoft/apm-sample-package
    - github/awesome-copilot/skills/review-and-refactor
  isolated: true   # clear repo primitives before unpack (default: false)
```

## Package reference formats

Each entry is an APM package reference. Supported formats:

| Format | Description |
|--------|-------------|
| `owner/repo` | Full APM package |
| `owner/repo/path/to/primitive` | Individual primitive (skill, instruction, plugin, etc.) from a repository |
| `owner/repo#ref` | Package pinned to a tag, branch, or commit SHA |

### Examples

```yaml wrap
dependencies:
  # Full APM package
  - microsoft/apm-sample-package
  # Individual primitive from any repository
  - github/awesome-copilot/skills/review-and-refactor
  # Plugin (Claude plugin.json format)
  - github/awesome-copilot/plugins/context-engineering
  # Version-pinned to a tag
  - microsoft/apm-sample-package#v2.0
  # Version-pinned to a branch
  - microsoft/apm-sample-package#main
  # Git URL with sub-path and ref (object format)
  - git: https://github.com/acme/coding-standards.git
    path: instructions/security
    ref: v2.0
```

## Compilation behavior

The compiler emits an `apm pack` step in the activation job and an `apm unpack` step in the agent job. The APM target is automatically inferred from the configured engine (`copilot`, `claude`, or `all` for other engines). The `isolated` flag controls whether existing `.github/` primitive directories are cleared before the bundle is unpacked in the agent job.

To reproduce or debug the pack/unpack flow locally, run `apm pack` and `apm unpack` directly. See the [pack and distribute guide](https://microsoft.github.io/apm/guides/pack-distribute/) for instructions.

## Reference

| Resource | URL |
|----------|-----|
| APM documentation | https://microsoft.github.io/apm/ |
| APM governance guide | https://microsoft.github.io/apm/enterprise/governance/ |
| Pack and distribute guide | https://microsoft.github.io/apm/guides/pack-distribute/ |
| gh-aw integration (APM docs) | https://microsoft.github.io/apm/integrations/gh-aw/ |
| apm-action (GitHub) | https://github.com/microsoft/apm-action |
| microsoft/apm (GitHub) | https://github.com/microsoft/apm |
