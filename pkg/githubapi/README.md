# githubapi Package

> Provides a factory function for constructing go-gh REST client options with repository-standard defaults.

## Overview

The `githubapi` package is a thin adapter over the `github.com/cli/go-gh/v2/pkg/api` library. Its sole responsibility is to produce an `api.ClientOptions` value pre-populated with the repository's default HTTP timeout (`constants.DefaultHTTPClientTimeout`), so callers do not have to remember to set this timeout themselves.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `ClientOptions` | `func(host, authToken string) api.ClientOptions` | Returns go-gh REST client options with the repository default timeout applied |

#### `ClientOptions`

```go
func ClientOptions(host, authToken string) api.ClientOptions
```

Returns an `api.ClientOptions` struct suitable for passing to `api.NewRESTClient` or `api.NewGraphQLClient`.

| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | `string` | GitHub hostname (e.g. `"github.com"` or a GHES host) |
| `authToken` | `string` | Personal access token or GitHub App token for authentication |

**Behavioral contract**:
- MUST set `Host` to the provided `host` value.
- MUST set `AuthToken` to the provided `authToken` value.
- MUST set `Timeout` to `constants.DefaultHTTPClientTimeout`.

## Usage Examples

```go
import (
    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/github/gh-aw/pkg/githubapi"
)

opts := githubapi.ClientOptions("github.com", token)
client, err := api.NewRESTClient(opts)
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/constants` — supplies `DefaultHTTPClientTimeout`

**External**:
- `github.com/cli/go-gh/v2/pkg/api` — defines the `ClientOptions` struct

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
