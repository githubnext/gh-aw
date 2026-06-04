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
- Prefer `tools.github.mode: gh-proxy` with `gh` for GitHub reads.
- Prefer `tools.cli-proxy: true` with mounted `mcp-clis` commands for non-GitHub MCP tools.
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

## Self-Hosted Runner Compatibility

When a workflow targets a self-hosted runner (any `runs-on` value other than GitHub-hosted labels such as `ubuntu-latest`, `ubuntu-slim`, `windows-latest`, or `macos-latest`), keep the generated workflow compatible with self-hosted constraints:

- Set `runs-on` explicitly (it is not inherited from imports) to the runner the user's setup provides; `runs-on` accepts a string, array, or runner-group object. Framework/generated jobs (activation, safe-outputs, unlock, etc.) default to the hosted `ubuntu-slim`, so also set `runs-on-slim` to route them to the self-hosted runner, otherwise they try to run on a hosted runner. `runs-on-slim` takes a single string label, so give it a self-hosted label the runner answers to (it cannot mirror an array or object value).
- Write transient state, tool downloads, and intermediate outputs under `$RUNNER_TEMP`, not `/tmp`, which can persist across jobs on shared runners.
- The agent job's own steps run as the runner user, not root — don't write steps that assume root (for example, installing to system-wide paths). Separately, the egress firewall needs host-level privileges (sudo) on the runner; if the host cannot provide that, the firewall can be disabled, which removes egress filtering. Surface that trade-off to the user rather than encoding it in the workflow.
- Declare every outbound domain the workflow contacts in `network.allowed` (keep `defaults` for the core GitHub/Copilot/registry endpoints). When the egress firewall is enabled (the default once network permissions are set), any domain that is not allow-listed is blocked.
- Do not install to system-wide paths such as `/usr/local` or the toolcache — they may be read-only or shared across runners. Install into job-scoped writable paths instead.
- Do not hardcode `/home/runner` or any literal home path — read `$HOME` from the environment instead, and use `$RUNNER_TEMP` for transient state since it is guaranteed writable.
- For GitHub Enterprise Server, enable GHES compatibility so generated workflows use artifact action versions that work on GHES, and configure the enterprise API endpoint.

For the full set of requirements (Docker socket, ARC / Docker-in-Docker, network egress, GHES specifics), follow the [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/) reference page.

## Shared Reminder

Reference this file from creator, updater, and debugger prompts instead of repeating the same architectural explanation.
