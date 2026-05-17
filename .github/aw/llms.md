---
description: Discover LLM API endpoints, ports, and available model names inside the AWF agent container using the api-proxy /reflect endpoint.
---

# LLM API Endpoint Discovery

The AWF api-proxy sidecar exposes a `/reflect` endpoint listing every configured LLM provider, its port, and available models. Use it to configure any tool that needs OpenAI/Anthropic access inside the agent container.

> ⚠️ Only reachable from **inside the AWF agent container** — not from the runner host.

## Quick Start

```bash
# Discover all configured providers and their models
curl -sf http://api-proxy:10000/reflect | jq '.endpoints[] | select(.configured)'
```

## Provider Ports

| Provider | Port | Base URL | Credentials env var |
|---|---|---|---|
| `openai` / `codex` | 10000 | `http://api-proxy:10000/v1` | `OPENAI_API_KEY` |
| `anthropic` | 10001 | `http://api-proxy:10001/v1` (or no `/v1` for native SDK) | `ANTHROPIC_API_KEY` |
| `copilot` | 10002 | `http://api-proxy:10002/v1` | `COPILOT_GITHUB_TOKEN` |
| `gemini` | 10003 | `http://api-proxy:10003/v1` | `GEMINI_API_KEY` |

All ports use the OpenAI-compatible API format. The api-proxy injects auth headers automatically — **do not pass raw API keys** to these URLs.

## /reflect Response

```json
{
  "endpoints": [
    { "provider": "openai",    "port": 10000, "configured": true,  "models": ["gpt-4o", "o1-mini"], "models_url": "http://api-proxy:10000/v1/models" },
    { "provider": "anthropic", "port": 10001, "configured": true,  "models": ["claude-sonnet-4-5"],  "models_url": "http://api-proxy:10001/v1/models" },
    { "provider": "copilot",   "port": 10002, "configured": true,  "models": null,                   "models_url": "http://api-proxy:10002/models" },
    { "provider": "gemini",    "port": 10003, "configured": false, "models": null,                   "models_url": null }
  ],
  "models_fetch_complete": true
}
```

Only use endpoints where `configured: true`. `models` may be `null` when the proxy hasn't finished fetching; use `models_url` to fetch on demand.

## Configure Tools

### OpenAI-compatible SDK (any provider)

```bash
# Use Copilot as the OpenAI backend
export OPENAI_BASE_URL="http://api-proxy:10002/v1"
export OPENAI_API_KEY="$COPILOT_GITHUB_TOKEN"
```

### Anthropic SDK

```bash
export ANTHROPIC_BASE_URL="http://api-proxy:10001"
export ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"
```

### Dynamic resolution from /reflect

```bash
PROVIDER=anthropic
PORT=$(curl -sf http://api-proxy:10000/reflect \
  | jq -r --arg p "$PROVIDER" '.endpoints[] | select(.provider == $p and .configured) | .port')
export ANTHROPIC_BASE_URL="http://api-proxy:${PORT}"
```

## List Available Models

```bash
# OpenAI / Anthropic / Copilot format → { data: [{id}] }
curl -sf http://api-proxy:10000/reflect \
  | jq -r '.endpoints[] | select(.provider == "openai" and .configured) | .models_url' \
  | xargs curl -sf | jq '[.data[].id]'

# Gemini format → { models: [{name: "models/gemini-..."}] }
curl -sf http://api-proxy:10003/v1/models | jq '[.models[].name | ltrimstr("models/")]'
```

## See Also

- [network.md](network.md) — egress domain configuration
- [syntax.md](syntax.md) — `engine:` and `engine.model` frontmatter
