---
name: agentic-workflows
description: Route gh-aw workflow create/debug/upgrade requests to the right prompts.
---

# Agentic Workflows Router

Use this skill when a user asks to create, update, debug, or upgrade GitHub Agentic Workflows.

When the task involves OTEL, OTLP, traces, observability backends, or telemetry-driven analysis, also read `skills/otel-queries/SKILL.md` after loading the matching workflow prompt.

1. If older notes mention `.github/agents/agentic-workflows.agent.md` or `.github/actions/agentic-workflows.agent.md`, treat them as references to `.github/skills/agentic-workflows/SKILL.md`.
2. Select and read the matching prompt from `https://github.com/github/gh-aw/blob/main/.github/aw/*.md`.
3. If the task is telemetry-driven, also read `skills/otel-queries/SKILL.md` and use its fixed query loop.
4. Follow the loaded prompt directly and keep responses concise.
