---
title: Model Aliases
description: Reference tables for the built-in model alias map used by GitHub Agentic Workflows.
sidebar:
  order: 297
---

This page lists the built-in model aliases used by GitHub Agentic Workflows.

## Model Aliases

Model aliases let you write `engine: copilot` with a human-friendly name such as `sonnet` or `mini`. gh-aw resolves each alias at compile time by trying its ordered fallback patterns until one matches an available model.

For alias syntax, fallback resolution, and custom aliases in workflow frontmatter, see the [Model Alias Format Specification](/gh-aw/specs/model-alias-specification/).

### Vendor Aliases

Vendor aliases map a short name to one or more provider-scoped glob patterns, with the Copilot gateway tried first.

| Alias | Fallback patterns (tried in order) |
|-------|-------------------------------------|
| `sonnet` | `copilot/*sonnet*`, `anthropic/*sonnet*` |
| `sonnet-6x` | `copilot/*sonnet-4.5*`, `copilot/*sonnet-4.6*`, `copilot/*sonnet-4-5-*`, `anthropic/*sonnet-4-5-*`, `copilot/*sonnet-4-6*`, `anthropic/*sonnet-4-6*` |
| `haiku` | `copilot/*haiku*`, `anthropic/*haiku*` |
| `opus` | `copilot/*opus*`, `anthropic/*opus*` |
| `gpt-5` | `copilot/gpt-5*`, `openai/gpt-5*` |
| `gpt-5.5` | `copilot/gpt-5.5*`, `openai/gpt-5.5*` |
| `gpt-5.4` | `copilot/gpt-5.4*`, `openai/gpt-5.4*` |
| `gpt-5.3` | `copilot/gpt-5.3*`, `openai/gpt-5.3*` |
| `gpt-5.2` | `copilot/gpt-5.2*`, `openai/gpt-5.2*` |
| `gpt-5.1` | `copilot/gpt-5.1*`, `openai/gpt-5.1*` |
| `gpt-5-mini` | `copilot/gpt-5*mini*`, `openai/gpt-5*mini*` |
| `gpt-5-nano` | `copilot/gpt-5*nano*`, `openai/gpt-5*nano*` |
| `gpt-5-codex` | `copilot/gpt-5*codex*`, `openai/gpt-5*codex*` |
| `coding` | `copilot/gpt-5*codex*`, `openai/gpt-5*codex*`, `gpt-5-codex` |
| `mai-code` | `copilot/MAI-Code*`, `copilot/mai-code*`, `openai/MAI-Code*` |
| `gpt-5-pro` | `copilot/gpt-5*pro*`, `openai/gpt-5*pro*` |
| `reasoning` | `copilot/o1*`, `copilot/o3*`, `copilot/o4*`, `openai/o1*`, `openai/o3*`, `openai/o4*` |
| `gemini-flash` | `copilot/gemini-*flash*`, `google/gemini-*flash*`, `gemini/gemini-*flash*` |
| `gemini-flash-lite` | `copilot/gemini-*flash*lite*`, `google/gemini-*flash*lite*`, `gemini/gemini-*flash*lite*` |
| `gemini-pro` | `copilot/gemini-*pro*`, `google/gemini-*pro*`, `gemini/gemini-*pro*` |
| `vision` | `copilot/gemini-*image*`, `google/gemini-*image*`, `gemini/gemini-*image*`, `copilot/gemini-*flash*`, `google/gemini-*flash*`, `gemini/gemini-*flash*` |
| `image-generation` | `copilot/gpt-image*`, `openai/gpt-image*`, `openai/chatgpt-image*`, `copilot/gemini-*image*`, `google/gemini-*image*`, `gemini/gemini-*image*`, `google/imagen*` |
| `gemma` | `copilot/gemma*`, `google/gemma*`, `gemini/gemma*` |
| `deep-research` | `copilot/deep-research*`, `copilot/o3-deep-research*`, `copilot/o4-mini-deep-research*`, `google/deep-research*`, `gemini/deep-research*`, `openai/o3-deep-research*`, `openai/o4-mini-deep-research*` |
| `any` | `copilot/*`, `anthropic/*`, `openai/*`, `google/*`, `gemini/*` |
| `gemini-3-pro` | `copilot/gemini-3*pro*`, `google/gemini-3*pro*`, `google/nano-banana*`, `gemini/gemini-3*pro*` |
| `gemini-3-flash` | `copilot/gemini-3*flash*`, `google/gemini-3*flash*`, `gemini/gemini-3*flash*` |
| `gemini-3.1-pro` | `copilot/gemini-3.1*pro*`, `google/gemini-3.1*pro*`, `gemini/gemini-3.1*pro*` |
| `gemini-3.1-flash` | `copilot/gemini-3.1*flash*`, `google/gemini-3.1*flash*`, `gemini/gemini-3.1*flash*` |
| `gemini-3.5-flash` | `copilot/gemini-3.5*flash*`, `google/gemini-3.5*flash*`, `gemini/gemini-3.5*flash*` |
| `antigravity` | `copilot/antigravity*`, `google/antigravity*`, `gemini/antigravity*` |
| `nano-banana` | `copilot/nano-banana*`, `google/nano-banana*`, `gemini/nano-banana*` |
| `computer-use` | `copilot/*computer-use*`, `google/*computer-use*`, `gemini/*computer-use*`, `openai/*computer-use*` |
| `robotics` | `copilot/*robotics*`, `google/*robotics*`, `gemini/*robotics*` |

### Meta-Aliases

Meta-aliases expand to other aliases and resolve recursively until they reach a concrete pattern.

| Meta-alias | Expands to |
|------------|------------|
| `opusplan` | `opus?effort=high` |
| `small` | `mini` |
| `mini` | `haiku` → `gpt-5-mini` → `gpt-5-nano` → `gemini-flash-lite` |
| `large` | `sonnet` → `gpt-5-pro` → `gpt-5` → `gemini-pro` |
| `agent` | `sonnet-6x` → `gpt-5.5` → `gpt-5.4` → `gpt-5.3` → `gemini-pro` → `any` |
| `small-agent` | `haiku` → `gpt-5-mini` → `gemini-flash` |
| `copilot` | `agent` |
| `claude` | `agent` |
| `codex` | `agent` |
| `gemini` | `agent` |
| `summarization` | `haiku` → `gpt-5-mini` → `gemini-flash-lite` → `mini` |
