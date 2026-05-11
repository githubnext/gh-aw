---
name: agentic-workflows
description: Router skill for gh-aw workflow tasks. Load `.github/agents/agentic-workflows.agent.md` plus relevant prompts from `.github/aw/*.md`.
---

# Agentic Workflows Router

Use this skill when a user asks to create, update, debug, or upgrade GitHub Agentic Workflows.

When the task involves OTEL, OTLP, traces, observability backends, or telemetry-driven analysis, also read `skills/otel-queries/SKILL.md` after loading the matching workflow prompt.

1. Read `.github/agents/agentic-workflows.agent.md` (also referred to as `.github/actions/agentic-workflows.agent.md` in older notes).
2. Select and read the matching prompt from `.github/aw/*.md`.
3. If the task is telemetry-driven, also read `skills/otel-queries/SKILL.md` and use its fixed query loop.
4. Follow the loaded prompt directly and keep responses concise.
