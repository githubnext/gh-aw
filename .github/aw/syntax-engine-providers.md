---
description: Inline engine definitions for routing a runtime adapter to a custom LLM provider (BYOK) in GitHub Agentic Workflows.
---

# Engine: Inline Provider Routing (BYOK)

Use an inline engine definition to point a runtime adapter at a custom or self-hosted LLM provider with your own key, auth flow, and request shaping. This is the declarative alternative to `engine.auth` keyless WIF (see [syntax-agentic.md](syntax-agentic.md)); reach for it when you bring your own provider endpoint.

Shape: replace the engine `id` with a `runtime` block and an optional `provider` block.

```yaml
engine:
  runtime:
    id: codex                 # Required: runtime adapter (codex, claude, copilot, gemini, opencode, crush, pi)
    version: "0.105.0"        # Optional: pin the adapter version
  provider:
    id: openai                # Provider backend: openai, anthropic, github, google
    model: gpt-5              # Optional: specific model
    auth:
      strategy: api-key       # api-key (default when secret set) | oauth-client-credentials | bearer
      secret: OPENAI_API_KEY  # Secret name holding the API key/token (api-key, bearer)
      header-name: api-key    # Header to inject the key into; required unless strategy is bearer
  bare: false                 # Optional: disable automatic context loading (default false)
```

`runtime` is required; `provider` is optional and only needed for non-default backends.

## Auth strategies (`provider.auth`)

| Field | Applies to | Meaning |
|---|---|---|
| `strategy` | all | `api-key` (default when `secret` set), `oauth-client-credentials`, or `bearer` |
| `secret` | api-key, bearer | Secret name holding the raw API key or token |
| `header-name` | non-bearer | HTTP header to inject the key into (e.g. `api-key`, `x-api-key`); bearer always uses `Authorization` |
| `token-url` | oauth-client-credentials | OAuth 2.0 token endpoint |
| `client-id` | oauth-client-credentials | Secret name holding the OAuth client ID |
| `client-secret` | oauth-client-credentials | Secret name holding the OAuth client secret |
| `token-field` | oauth-client-credentials | JSON field in the token response (default `access_token`) |

All `secret`/`client-id`/`client-secret` values are GitHub Actions secret **names**, not literal values.

## Request shaping (`provider.request`)

Reshape the outgoing call for non-standard backends (e.g. Azure OpenAI):

| Field | Meaning |
|---|---|
| `path-template` | URL path with `{model}` and other placeholders, e.g. `/openai/deployments/{model}/chat/completions` |
| `query` | Static/template query params appended to every request, e.g. `api-version: "2024-10-01-preview"` |
| `body-inject` | Key/value pairs merged into the JSON request body before sending |

## Example: Azure OpenAI via OAuth client credentials

```yaml
engine:
  runtime:
    id: codex
  provider:
    id: azure-openai
    model: gpt-4o
    auth:
      strategy: oauth-client-credentials
      token-url: https://auth.example.com/oauth/token
      client-id: AZURE_CLIENT_ID
      client-secret: AZURE_CLIENT_SECRET
      header-name: api-key
    request:
      path-template: /openai/deployments/{model}/chat/completions
      query:
        api-version: "2024-10-01-preview"
```

## Notes

- For the `opencode` and `crush` adapters you can instead set `engine.model` in `provider/model` form (e.g. `openai/gpt-5`, `anthropic/claude-sonnet-4`); supported providers there are `copilot`, `anthropic`, `openai`, `codex`.
- BYOK Azure OpenAI with the AWF firewall: set `sandbox.agent.model-fallback: false` to prevent deployment-name rewriting (see [syntax-agentic.md](syntax-agentic.md)).
- Prefer the simple string/`id` engine form unless you genuinely need a custom provider endpoint.
