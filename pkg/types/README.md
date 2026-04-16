# types Package

The `types` package provides shared type definitions used across multiple `gh-aw` packages to avoid circular dependencies.

## Overview

This package defines common data structures that are shared between the `parser` and `workflow` packages. Centralizing these types here allows both packages to reference the same definitions without creating import cycles.

## Types

### `BaseMCPServerConfig`

The foundational configuration structure for MCP (Model Context Protocol) servers. This type is embedded by both `parser.MCPServerConfig` and `workflow.MCPServerConfig`.

MCP servers can run as:
- **stdio processes**: `Command` + `Args`, launched as a child process.
- **HTTP endpoints**: `URL` + optional `Headers` and `Auth`, reached over HTTP/HTTPS.
- **Container services**: `Container` image + optional `Mounts`, run inside a container.

```go
import "github.com/github/gh-aw/pkg/types"

// Stdio MCP server
cfg := types.BaseMCPServerConfig{
    Type:    "stdio",
    Command: "npx",
    Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
    Env: map[string]string{
        "ALLOWED_PATHS": "/workspace",
    },
}

// HTTP MCP server with OIDC auth
cfg := types.BaseMCPServerConfig{
    Type: "http",
    URL:  "https://my-mcp-server.example.com",
    Auth: &types.MCPAuthConfig{
        Type:     "github-oidc",
        Audience: "https://my-mcp-server.example.com",
    },
}
```

#### Fields

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Server type: `"stdio"`, `"http"`, `"local"`, or `"remote"` |
| `Command` | `string` | Executable to launch (stdio mode) |
| `Args` | `[]string` | Arguments passed to the command |
| `Env` | `map[string]string` | Environment variables injected into the process |
| `Version` | `string` | Optional version or tag |
| `URL` | `string` | HTTP endpoint URL (HTTP mode) |
| `Headers` | `map[string]string` | Additional HTTP headers (HTTP mode) |
| `Auth` | `*MCPAuthConfig` | Upstream authentication (HTTP mode only) |
| `Container` | `string` | Container image (container mode) |
| `Entrypoint` | `string` | Optional entrypoint override for the container |
| `EntrypointArgs` | `[]string` | Arguments passed to the container entrypoint |
| `Mounts` | `[]string` | Volume mounts in `"source:dest:mode"` format |

### `MCPAuthConfig`

Authentication configuration for HTTP MCP servers. When configured, the MCP gateway dynamically acquires tokens and injects them as `Authorization` headers on each outgoing request.

```go
auth := &types.MCPAuthConfig{
    Type:     "github-oidc", // Currently the only supported type
    Audience: "https://my-service.example.com",
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Auth type; currently only `"github-oidc"` is supported |
| `Audience` | `string` | OIDC token audience (`aud` claim); defaults to the server URL if omitted |

### `TokenWeights`

Defines custom model cost information for effective token computation. Specified under `engine.token-weights` in workflow frontmatter and stored in `aw_info.json` at runtime.

```go
weights := types.TokenWeights{
    Multipliers: map[string]float64{
        "gpt-4o": 2.5,
    },
    TokenClassWeights: &types.TokenClassWeights{
        Input:  1.0,
        Output: 3.0,
    },
}
```

### `TokenClassWeights`

Per-token-class weights for effective token computation. Each field corresponds to one token class; a zero value means "use the default weight".

| Field | Token class |
|-------|-------------|
| `Input` | Standard input tokens |
| `CachedInput` | Cache-hit input tokens |
| `Output` | Generated output tokens |
| `Reasoning` | Internal reasoning tokens |
| `CacheWrite` | Cache-write tokens |

## Design Notes

- This package has no dependencies on other `gh-aw` packages, making it safe to import from anywhere.
- All struct fields use both `json` and `yaml` struct tags so they can be round-tripped through both serialization formats.
- `BaseMCPServerConfig` is designed to be embedded — packages add domain-specific fields and validation on top of the shared base.
