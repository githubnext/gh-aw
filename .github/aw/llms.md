---
description: How to query the AWF /reflect endpoint to discover the LLM API endpoints, available model names, and configure OpenAI/Anthropic-compatible tools inside the AWF agent container.
---

# LLM API Endpoint Discovery

The AWF API proxy (`api-proxy` sidecar) exposes a `/reflect` management endpoint that describes every configured LLM provider, its port, and the models available for that run. Query this endpoint first when you need to configure a third-party tool that requires LLM access (e.g. an OpenAI-compatible client, an Anthropic SDK, or a local coding assistant).

## The `/reflect` Endpoint

The endpoint is only reachable from **inside the AWF agent container**, not from the GitHub Actions runner host.

```
GET http://api-proxy:10000/reflect
```

### Quick query

```bash
curl -sf http://api-proxy:10000/reflect | jq .
```

### Response shape

```json
{
  "endpoints": [
    {
      "provider": "openai",
      "port": 10000,
      "configured": true,
      "models": ["gpt-4o", "gpt-4o-mini", "o1-mini"],
      "models_url": "http://api-proxy:10000/v1/models"
    },
    {
      "provider": "anthropic",
      "port": 10001,
      "configured": true,
      "models": ["claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-4-5"],
      "models_url": "http://api-proxy:10001/v1/models"
    },
    {
      "provider": "copilot",
      "port": 10002,
      "configured": true,
      "models": null,
      "models_url": "http://api-proxy:10002/models"
    },
    {
      "provider": "gemini",
      "port": 10003,
      "configured": false,
      "models": null,
      "models_url": null
    }
  ],
  "models_fetch_complete": true
}
```

**Field descriptions:**

| Field | Type | Description |
|---|---|---|
| `provider` | string | Provider name: `openai`, `anthropic`, `copilot`, or `gemini` |
| `port` | number | Port on `api-proxy` that serves this provider |
| `configured` | boolean | `true` when credentials are available for this provider |
| `models` | string[] or null | Sorted list of available model IDs; `null` if not yet fetched |
| `models_url` | string or null | URL to call on `api-proxy` to list models for this provider |
| `models_fetch_complete` | boolean | Top-level flag; `true` once the proxy has finished fetching all model lists |

Only endpoints where `configured: true` are usable during the run.

## Default Provider Ports

The api-proxy exposes each provider on a dedicated port. All ports use an OpenAI-compatible API format unless noted otherwise.

| Provider | Default port | OpenAI-compatible base URL | Notes |
|---|---|---|---|
| `openai` / `codex` | 10000 | `http://api-proxy:10000/v1` | OpenAI API format; also used by Claude engine |
| `anthropic` | 10001 | `http://api-proxy:10001/v1` | OpenAI-compatible; native Anthropic SDK: `http://api-proxy:10001` |
| `copilot` | 10002 | `http://api-proxy:10002/v1` | OpenAI-compatible; Copilot token authentication |
| `gemini` | 10003 | `http://api-proxy:10003/v1` | OpenAI-compatible |

> The api-proxy injects authentication headers automatically. Do not pass API keys when routing through these endpoints inside the container.

## Discovering Available Models

Use `models_url` from the reflect response to get the live model list for any configured provider:

```bash
# Read reflect once and extract the OpenAI models_url
MODELS_URL=$(curl -sf http://api-proxy:10000/reflect \
  | jq -r '.endpoints[] | select(.provider == "openai" and .configured) | .models_url')

# Fetch the model list (OpenAI / Anthropic format: { data: [{id, ...}] })
curl -sf "$MODELS_URL" | jq '[.data[].id]'
```

For Gemini, the model list uses the Gemini format (`{ models: [{name: "models/gemini-1.5-pro", ...}] }`):

```bash
curl -sf http://api-proxy:10003/v1/models \
  | jq '[.models[].name | ltrimstr("models/")]'
```

## Configuring Tools That Require LLM Access

### OpenAI SDK / OpenAI-compatible clients

Set `OPENAI_BASE_URL` (or the equivalent `base_url` config option) to the provider's base URL from the reflect response. Use the `models_url` path without the trailing `/models` segment as the base:

```bash
# Copilot provider via OpenAI SDK
export OPENAI_BASE_URL="http://api-proxy:10002/v1"
export OPENAI_API_KEY="$(printenv COPILOT_GITHUB_TOKEN)"  # set by AWF automatically
```

```bash
# OpenAI provider
export OPENAI_BASE_URL="http://api-proxy:10000/v1"
export OPENAI_API_KEY="$(printenv OPENAI_API_KEY)"
```

### Anthropic SDK

Point `ANTHROPIC_BASE_URL` at the Anthropic provider port (no `/v1` suffix needed for the native SDK):

```bash
export ANTHROPIC_BASE_URL="http://api-proxy:10001"
export ANTHROPIC_API_KEY="$(printenv ANTHROPIC_API_KEY)"
```

### Dynamically resolving the base URL from /reflect

Use this pattern when the provider or port may vary at runtime:

```bash
# Resolve the base URL for the first configured Anthropic provider
ANTHROPIC_PORT=$(curl -sf http://api-proxy:10000/reflect \
  | jq -r '.endpoints[] | select(.provider == "anthropic" and .configured) | .port')

export ANTHROPIC_BASE_URL="http://api-proxy:${ANTHROPIC_PORT}"
```

## Shell Helper: Print All Configured Providers

```bash
curl -sf http://api-proxy:10000/reflect | jq '
  .endpoints[]
  | select(.configured)
  | { provider, port, model_count: (.models | if . then length else "unknown" end) }
'
```

Example output:

```json
{ "provider": "openai", "port": 10000, "model_count": 42 }
{ "provider": "anthropic", "port": 10001, "model_count": 8 }
{ "provider": "copilot", "port": 10002, "model_count": "unknown" }
```

## Network Requirements

The `api-proxy` service is part of the AWF Docker network and is always reachable via the `api-proxy` hostname from the agent container. No additional `network.allowed` entries are required.

> ⚠️ Do **not** call `http://api-proxy:10000/reflect` or the provider endpoints from **outside** the AWF container (e.g., in a non-agent job step). Those URLs resolve only within the AWF sandbox network.

## Step Summary

AWF automatically appends a collapsed provider/model table to the GitHub Actions step summary at the end of every agent run. This table is built from the same `/reflect` payload and is useful for post-run diagnostics.

## See Also

- [network.md](network.md) — Configuring egress domains for outbound network access
- [syntax.md](syntax.md) — `engine:` and `engine.model` frontmatter fields
- [github-agentic-workflows.md](github-agentic-workflows.md) — Complete workflow format reference
