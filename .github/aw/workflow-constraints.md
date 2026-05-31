---
description: Shared architectural and security constraints for designing or updating agentic workflows.
---

# Agentic Workflow Constraints

## Execution Model

Agentic workflows run as a **single GitHub Actions job** with one agent execution.

## What They Can Do

- read GitHub data, APIs, web pages, and local repository files
- run tools inside the single job
- use MCP servers and safe outputs
- create GitHub resources through `safe-outputs:`
- persist lightweight state with `cache-memory` or other approved mechanisms

## What They Cannot Do

- pause and resume for external events
- orchestrate multi-stage pipelines with job dependencies
- pass state between multiple AI jobs in one workflow run
- implement built-in rollback across external systems
- wait for another workflow or deployment to finish inside the same agent run

## When to Recommend Traditional GitHub Actions Instead

Use traditional Actions when the request needs:

- multi-stage deployment pipelines
- fan-out or fan-in job orchestration
- long waits for approvals or external systems
- rollback logic across several steps or systems
- cross-job state transfer

Suggested response pattern:

> This requires capabilities that agentic workflows do not support in their single-job model. Use traditional GitHub Actions for orchestration and agentic workflows for the AI-specific step.

## Security Posture

- Keep the main agent job read-only.
- Do not add GitHub write permissions to the agent job.
- Route GitHub writes through `safe-outputs:`.
- Prefer `tools.github.mode: gh-proxy` and `toolsets:` over ad hoc shell access.
- Constrain `network.allowed:` to the minimum required ecosystems or domains.
- Use `${{ steps.sanitized.outputs.text }}` for untrusted user content.

## Safer Alternatives First

When a requested feature increases risk:

1. explain the risk clearly
2. propose the safer pattern first
3. require explicit confirmation before relaxing safeguards

## Common Risk Areas

- direct write permissions instead of safe outputs
- auto-merge or bypassing review
- overly broad network access
- unbounded bash allowlists for untrusted input
- using `post-steps:` for agent-driven write actions

## Shared Reminder

Reference this file from creator, updater, and debugger prompts instead of repeating the same architectural explanation.
