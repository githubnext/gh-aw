---
title: gh-aw vs. engine-specific actions vs. hand-rolled GitHub Actions
description: Compare gh-aw with engine-specific GitHub Actions and custom Actions implementations for AI automation workflows.
---

This guide compares three ways to run AI automation in GitHub Actions: gh-aw, an engine-specific action such as Claude's, and a custom hand-rolled workflow. Use it to decide how much workflow structure, engine flexibility, and safety control the repository needs.

| Approach | Authoring format | Supported engines | Sandboxing/safe outputs | Trigger model | Human-in-the-loop controls | Cost model |
|---|---|---|---|---|---|---|
| gh-aw | Markdown workflows compiled to Actions | Copilot, Claude, Codex, Gemini | Built-in sandboxing and safe outputs | GitHub events and schedules | Pull request review and other GitHub review gates | AI provider billing plus GitHub Actions minutes |
| [`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action) | GitHub Actions YAML | Claude | No built-in gh-aw sandboxing or safe outputs | PR events and manual invocation | PR conversation and `@`-mention driven | Anthropic API plus GitHub Actions minutes |
| Hand-rolled GitHub Actions | GitHub Actions YAML plus custom scripts | Any engine the workflow integrates manually | Manual | All GitHub Actions triggers | Manual | Full provider and runtime flexibility plus GitHub Actions minutes |

## Recommendations

### Choose gh-aw when...

Choose gh-aw when the workflow should be authored in Markdown, run across multiple engines, and keep GitHub writes behind validated safe outputs. It also fits scheduled automation and event-driven repository tasks that should produce reviewable artifacts such as issues, comments, or pull requests.

### Choose anthropics/claude-code-action when...

Choose [`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action) when the repository specifically wants Claude-first pull request assistance and conversation-driven usage around comments and mentions, without adopting gh-aw's workflow model.

### Choose hand-rolled Actions when...

Choose hand-rolled [GitHub Actions](https://docs.github.com/en/actions) when the workflow needs full control over scripts, external systems, and execution flow, and the repository is willing to implement its own safety, review, and maintenance model.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
