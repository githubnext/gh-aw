---
title: Enterprise Configuration
description: Configure GitHub Agentic Workflows for GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC), including GHES compatibility mode and CLI setup.
sidebar:
  order: 51
---

# Enterprise Configuration

This page covers configuration options specific to GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC) deployments.

## Central Admission Control

The optional `admission` setting in `.github/workflows/aw.json` gates agent execution through a trusted service for centralized cost controls. The service is implemented separately in gh-cao; gh-aw provides the client and request contract:

```json title=".github/workflows/aw.json"
{
  "admission": {
    "url": "https://cao.example.com/api/admission",
    "on_outage": "ignore"
  }
}
```

The URL must be a literal, absolute HTTPS endpoint without credentials, query parameters, fragments, or expressions. Omit `admission` to disable the gate. Recompile workflows with `gh aw compile` after changing this configuration; the endpoint is embedded in each compiled workflow.

The activation job makes a REST `GET` request before feedback, secret validation, and agent execution. Every trigger, reusable workflow invocation, and run attempt is checked, including manual and dispatched runs. HTTP `200` or `204` admits execution. Explicit denials and other non-transient statuses always fail activation and prevent the agent from starting. Existing local AI Credit guardrails still apply independently.

### Outage Policy and Retries

Network failures, timeouts, HTTP `408`, HTTP `429`, and HTTP `5xx` are retried up to three total attempts, each with a ten-second timeout. Retries use exponential backoff (250 ms, then 500 ms), with up to 250 ms of additional jitter per delay. Response headers do not control retry timing. Each attempt sends the same invocation identity. Redirects are never followed and block execution rather than use the outage policy.

After retries are exhausted, `on_outage` determines the outcome:

| Value | Behavior |
|-------|----------|
| `ignore` (default) | Emit a generic warning and continue AW execution without admission |
| `block` | Fail activation with the same opaque admission error used for denials |

Explicit denial responses are never retried or ignored, including when a denial follows a transient failure. Invalid configuration, missing Actions credentials, and missing execution identity also block rather than use the outage policy.

### Service Request

The request authenticates with `Authorization: Bearer <github.token>`. This is **only the ephemeral GitHub Actions token** for the activation job. PATs, custom activation tokens, GitHub App tokens, Copilot tokens, and other secrets are never selected or forwarded. Configure only an endpoint trusted to receive this token. HTTPS redirects are rejected, and the token is never included in the URL.

The query string contains exactly these fields, matching the current execution's AW identity:

| Parameter | Value |
|-----------|-------|
| `repo` | Repository slug, such as `owner/repository` |
| `workflow_id` | Compiled AW workflow ID, matching `aw_info.workflow_id` |
| `workflow_call_id` | Current invocation ID, matching `aw_info.workflow_call_id`: `<run_id>-<run_attempt>:<workflow_id>` |
| `run_id` | GitHub Actions run ID |
| `run_attempt` | GitHub Actions run attempt |
| `ref` | Current GitHub Actions ref |
| `event_type` | Current GitHub Actions event name |

Values are URL-encoded. No event payload, arbitrary `aw_context` fields, actor identity, or usage information is sent. Caller-supplied `aw_context` cannot override the current execution's identity or event type.

### Privacy-Preserving Responses

The gh-cao service must validate the Actions token and authorize the claimed repository and run; query parameters are not proof of identity. The token is repository-scoped and does not attest the exact workflow invocation. Admission GETs are not a reservation protocol and must not charge or reserve credits; the service owns consistent cost-control decisions and duplicate handling. The [downstream service specification](https://github.com/github/gh-aw/blob/main/specs/admission-service-spec.md) defines authentication limits, accounting behavior, and acceptance criteria.

The service contract uses `200` or `204` for admission and a uniform `403` for policy denial. Denials must not distinguish budget exhaustion, spend thresholds, account state, or policy reasons through status codes, bodies, headers, response sizes, or reason-dependent timing. Return `Cache-Control: no-store` and no usage or reset information. The client also requests no caching.

The client discards response bodies and exposes **only `ERR_ADMISSION`**. Blocking failures use the same generic message for denials and exhausted outages. Fail-open outages produce a generic warning without diagnostics. The client never logs service status codes, response headers, bodies, tokens, or transport error details. This prevents error diagnostics from revealing AI Credit spend. The observable admit/deny decision itself cannot be hidden; the service must authenticate callers, constrain probing, and avoid offering a spend-inspection oracle.

## GitHub Enterprise Server (GHES) Compatibility

### GHES Compatibility Mode

GHES instances running versions that predate `@actions/artifact` v2.0.0 support cannot use `actions/upload-artifact@v4+` or `actions/download-artifact@v4+`. Attempting to run compiled workflows on these instances produces a `GHESNotSupportedError`.

gh-aw includes a GHES compatibility mode toggle (`aw.json` `ghes` or `gh aw compile --ghes`) for GHES releases that require the v3 artifact backend. Compatibility mode emits `upload-artifact@v3.2.2` and `download-artifact@v3.1.0`; default GitHub.com compilation continues to use the latest artifact actions.

This compatibility path supports GHES 3.21.x and earlier when the workflow runs on Actions Runner 2.327.1 or later, which is required by the Node.js 24 runtime used by these pinned artifact actions. For later GHES releases, keep compatibility mode enabled until your instance supports the v4 artifact backend.

#### Enable via `aw.json` (recommended)

Set `ghes: true` in `.github/workflows/aw.json` to apply GHES compatibility to every workflow compiled in the repository:

```json
{
  "ghes": true
}
```

#### Auto-detection with `gh aw init`

Running `gh aw init` inside a GHES repository automatically detects the deployment and writes `ghes: true` to `.github/workflows/aw.json`. No manual configuration is required.

#### Enable via CLI flag

Pass `--ghes` to `gh aw compile` for a one-off compilation without modifying `aw.json`:

```bash
gh aw compile --ghes my-workflow.md
```

> [!NOTE]
> The `--ghes` flag only affects the current compilation. Use `aw.json` to apply GHES compatibility permanently across all workflows in the repository.

## GitHub Enterprise Server CLI Setup

For `gh` CLI configuration, host authentication, and `GH_HOST` setup on GHES, see [GitHub Enterprise Server Support](/gh-aw/setup/cli/#github-enterprise-server-support) in the CLI reference.

## Copilot Engine on GHES

For Copilot-specific prerequisites, licensing requirements, and firewall configuration on GHES, see [Copilot Engine Prerequisites on GHES](/gh-aw/troubleshooting/common-issues/#copilot-engine-prerequisites-on-ghes).
