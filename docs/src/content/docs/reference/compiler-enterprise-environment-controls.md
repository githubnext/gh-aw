---
title: Compiler Enterprise Environment Controls
description: Enterprise environment variables injected and managed by the compiler for default guardrails and model overrides
sidebar:
  order: 655
---

Use these variables to set organization- or repository-wide defaults without editing individual workflow frontmatter files.

## Enterprise Control Variables

| Variable | Purpose | Applies when |
| --- | --- | --- |
| `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS` | Default AWF `apiProxy.maxEffectiveTokens` budget | `max-effective-tokens` is not set in frontmatter |
| `GH_AW_DEFAULT_MODEL_COPILOT` | Default fallback model for Copilot | `GH_AW_MODEL_AGENT_COPILOT` / `GH_AW_MODEL_DETECTION_COPILOT` is unset |
| `GH_AW_DEFAULT_MODEL_CLAUDE` | Default fallback model for Claude | `GH_AW_MODEL_AGENT_CLAUDE` / `GH_AW_MODEL_DETECTION_CLAUDE` is unset |
| `GH_AW_DEFAULT_MODEL_CODEX` | Default fallback model for Codex | `GH_AW_MODEL_AGENT_CODEX` / `GH_AW_MODEL_DETECTION_CODEX` is unset |

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

## Example

Set an org-wide Codex model fallback:

```bash
gh variable set GH_AW_DEFAULT_MODEL_CODEX --org my-org --body "gpt-5.5"
```

Set an org-wide default max-effective-tokens guardrail:

```bash
gh variable set GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS --org my-org --body "15000000"
```
