---
title: Compiler Enterprise Environment Controls
description: Enterprise environment variables injected and managed by the compiler for default guardrails and model overrides
sidebar:
  order: 655
---

Use these variables to set organization- or repository-wide defaults without editing individual workflow frontmatter files.

## Enterprise Control Variables

| Variable | Source | Purpose | Applies when |
| --- | --- | --- | --- |
| `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS` | Compiler process environment | Default AWF `apiProxy.maxEffectiveTokens` budget | `max-effective-tokens` is not set in frontmatter |
| `GH_AW_DEFAULT_MAX_TURNS` | Compiler process environment | Default `engine.max-turns` | `engine.max-turns` is not set in frontmatter and the selected engine supports max-turns |
| `GH_AW_DEFAULT_TIMEOUT_MINUTES` | Compiler process environment | Default top-level `timeout-minutes` | `timeout-minutes` is not set in frontmatter |
| `GH_AW_DEFAULT_DETECTION_MODEL` | Compiler process environment | Default threat-detection model | `safe-outputs.threat-detection.engine.model` is not set |
| `GH_AW_DEFAULT_MODEL_COPILOT` | GitHub Actions `vars.*` at runtime | Default fallback model for Copilot | `GH_AW_MODEL_AGENT_COPILOT` / `GH_AW_MODEL_DETECTION_COPILOT` is unset |
| `GH_AW_DEFAULT_MODEL_CLAUDE` | GitHub Actions `vars.*` at runtime | Default fallback model for Claude | `GH_AW_MODEL_AGENT_CLAUDE` / `GH_AW_MODEL_DETECTION_CLAUDE` is unset |
| `GH_AW_DEFAULT_MODEL_CODEX` | GitHub Actions `vars.*` at runtime | Default fallback model for Codex | `GH_AW_MODEL_AGENT_CODEX` / `GH_AW_MODEL_DETECTION_CODEX` is unset |

Use `gh aw env get` and `gh aw env update` to manage these
variables in batch at repo, org, or enterprise scope. The defaults file uses
`default_`-prefixed keys such as `default_max_effective_tokens`, `default_timeout_minutes`, and
`default_model_copilot`.

## Precedence

For model selection, precedence is:

1. `engine.model` in workflow frontmatter
2. `GH_AW_MODEL_AGENT_*` or `GH_AW_MODEL_DETECTION_*`
3. `GH_AW_DEFAULT_MODEL_*`
4. Built-in compiler fallback

For max effective tokens, precedence is:

1. `max-effective-tokens` in workflow frontmatter
2. `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS`
3. Built-in compiler default

A negative `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS` disables AWF token steering and
omits the budget limit when frontmatter does not set `max-effective-tokens`.

For default timeout-minutes, precedence is:

1. `timeout-minutes` in workflow frontmatter
2. `GH_AW_DEFAULT_TIMEOUT_MINUTES`
3. Built-in compiler default

For detection engine selection, precedence is:

1. `safe-outputs.threat-detection.engine` in workflow frontmatter
2. Main workflow engine (`engine`)
3. Built-in compiler default

For detection model selection, precedence is:

1. `safe-outputs.threat-detection.engine.model` in workflow frontmatter
2. `GH_AW_DEFAULT_DETECTION_MODEL`
3. Engine-specific detection defaults

## Example

Set an org-wide Codex model fallback:

```bash
gh variable set GH_AW_DEFAULT_MODEL_CODEX --org my-org --body "gpt-5.5"
```

Set an org-wide default max-effective-tokens guardrail:

```bash
gh variable set GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS --org my-org --body "15000000"
```

Set compiler process defaults for timeout and max-turns:

```bash
export GH_AW_DEFAULT_TIMEOUT_MINUTES=30
export GH_AW_DEFAULT_MAX_TURNS=12
export GH_AW_DEFAULT_DETECTION_MODEL=gpt-5.5-mini
```
