---
title: Model Aliases & Multipliers
description: Reference tables for the built-in model alias map and per-model Effective Token multipliers used by GitHub Agentic Workflows.
sidebar:
  order: 297
---

This page lists the built-in model aliases and the per-model Effective Token (ET) multipliers used by GitHub Agentic Workflows.

> [!CAUTION]
> The multiplier values shown on this page are **approximations**. They are used solely for the purpose of normalizing token usage across models into a single comparable metric (Effective Tokens) and do **not** represent precise cost ratios. Values may be inaccurate for specific model versions and may become out of date as providers update their offerings. Do not use these numbers for billing or financial calculations.

## Model Aliases

Model aliases let you write `engine: copilot` with a human-friendly model name such as `sonnet` or `mini`, and gh-aw resolves it to the best available concrete model at compile time. Each alias holds an ordered list of patterns; the first pattern that matches an available model wins.

For details on the alias syntax, fallback resolution algorithm, and how to define your own aliases in workflow frontmatter, see the [Model Alias Format Specification](/gh-aw/reference/model-alias-specification/).

### Vendor Aliases

Vendor aliases map a short name to one or more provider-scoped glob patterns. The Copilot gateway is always tried first.

| Alias | Fallback patterns (tried in order) |
|-------|-------------------------------------|
| `sonnet` | `copilot/*sonnet*`, `anthropic/*sonnet*` |
| `haiku` | `copilot/*haiku*`, `anthropic/*haiku*` |
| `opus` | `copilot/*opus*`, `anthropic/*opus*` |
| `gpt-4.1` | `copilot/gpt-4.1*`, `openai/gpt-4.1*` |
| `gpt-5` | `copilot/gpt-5*`, `openai/gpt-5*` |
| `gpt-5-mini` | `copilot/gpt-5*mini*`, `openai/gpt-5*mini*` |
| `gpt-5-nano` | `copilot/gpt-5*nano*`, `openai/gpt-5*nano*` |
| `gpt-5-codex` | `copilot/gpt-5*codex*`, `openai/gpt-5*codex*` |
| `gpt-5-pro` | `copilot/gpt-5*pro*`, `openai/gpt-5*pro*` |
| `reasoning` | `copilot/o1*`, `copilot/o3*`, `copilot/o4*`, `openai/o1*`, `openai/o3*`, `openai/o4*` |
| `gemini-flash` | `copilot/gemini-*flash*`, `google/gemini-*flash*` |
| `gemini-pro` | `copilot/gemini-*pro*`, `google/gemini-*pro*` |
| `deep-research` | `copilot/deep-research*`, `google/deep-research*` |

### Meta-Aliases

Meta-aliases reference other aliases by name. They are resolved recursively until a concrete pattern is reached.

| Meta-alias | Expands to |
|------------|------------|
| `small` | `mini` |
| `mini` | `haiku` → `gpt-5-mini` → `gpt-5-nano` → `gemini-flash` |
| `large` | `sonnet` → `gpt-5-pro` → `gpt-5` → `gemini-pro` |
| `auto` | `large` |

## Model Multipliers

Effective Token multipliers scale the weighted token total for each model relative to the reference model (`claude-sonnet-4.5`, multiplier = 1.0). A multiplier of 5.0 means that a run on that model counts as five times as many Effective Tokens as the same run on the reference model.

See the [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) for the full formula.

### Token Class Weights

Before per-model multipliers are applied, raw token counts are weighted by token class:

| Token class | Default weight |
|-------------|---------------|
| Input | 1 |
| Cached Input | 0.1 |
| Output | 4 |
| Reasoning | 4 |
| Cache Write | 1 |

### Per-Model Multipliers

### Anthropic

| Model | Multiplier |
|-------|-----------|
| `claude-haiku-4-5` | 0.1 |
| `claude-haiku-4.5` | 0.1 |
| `claude-3-5-haiku` | 0.1 |
| `claude-3-haiku` | 0.1 |
| `claude-sonnet-4` | 1 |
| `claude-sonnet-4-5` | 1 |
| `claude-sonnet-4.5` | 1 |
| `claude-sonnet-4.6` | 1 |
| `claude-3-5-sonnet` | 1 |
| `claude-3-7-sonnet` | 1 |
| `claude-3-sonnet` | 1 |
| `claude-opus-4` | 5 |
| `claude-opus-4-1` | 5 |
| `claude-opus-4-5` | 5 |
| `claude-opus-4-6` | 5 |
| `claude-opus-4-7` | 5 |
| `claude-opus-4.5` | 5 |
| `claude-opus-4.6` | 5 |
| `claude-3-5-opus` | 5 |
| `claude-3-opus` | 5 |

### OpenAI

| Model | Multiplier |
|-------|-----------|
| `gpt-4o` | 1 |
| `gpt-4o-mini` | 0.1 |
| `gpt-4.1` | 1 |
| `gpt-4.1-2025-04-14` | 1 |
| `gpt-41-copilot` | 1 |
| `gpt-4.1-mini` | 0.1 |
| `gpt-4.1-nano` | 0.05 |
| `gpt-4-turbo` | 1 |
| `gpt-4` | 1 |
| `gpt-5` | 1 |
| `gpt-5-mini` | 0.33 |
| `gpt-5-nano` | 0.05 |
| `gpt-5-pro` | 2 |
| `gpt-5.1` | 1 |
| `gpt-5.1-codex` | 1 |
| `gpt-5.1-codex-mini` | 0.33 |
| `gpt-5.1-codex-max` | 1 |
| `gpt-5.2` | 1 |
| `gpt-5.2-codex` | 1 |
| `gpt-5.2-pro` | 2 |
| `gpt-5.3-codex` | 1 |
| `gpt-5.4` | 1 |
| `gpt-5.4-mini` | 0.33 |
| `gpt-5.4-pro` | 2 |
| `gpt-5.5` | 1 |
| `gpt-5.5-pro` | 2 |

### OpenAI Reasoning

| Model | Multiplier |
|-------|-----------|
| `o1` | 3 |
| `o1-mini` | 0.5 |
| `o1-pro` | 10 |
| `o3` | 3 |
| `o3-mini` | 0.5 |
| `o3-pro` | 10 |
| `o4-mini` | 0.5 |

### Google

| Model | Multiplier |
|-------|-----------|
| `gemini-2.5-pro` | 1 |
| `gemini-2.5-flash` | 0.2 |
| `gemini-2.5-flash-image` | 0.2 |
| `gemini-2.5-flash-lite` | 0.1 |
| `gemini-2.0-flash` | 0.1 |
| `gemini-1.5-pro` | 1 |
| `gemini-1.5-flash` | 0.1 |
| `gemini-3-flash-preview` | 0.33 |
| `gemini-3-pro-preview` | 6 |
| `gemini-3-pro-image-preview` | 6 |
| `gemini-3.1-pro-preview` | 6 |
| `gemini-3.1-flash-lite-preview` | 0.1 |
| `gemini-3.1-flash-image-preview` | 0.33 |
