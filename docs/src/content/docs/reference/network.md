---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1300
---

Control network access for AI engines using the top-level `network` field to specify which domains and services your agentic workflows can access during execution.

> **Note**: Network permissions are currently supported by the Claude engine and the Copilot engine (when using the [firewall feature](/gh-aw/reference/engines/#network-permissions)).

If no `network:` permission is specified, it defaults to `network: defaults` which allows access to basic infrastructure domains (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources).

## Configuration

```yaml wrap
# Default: basic infrastructure only
engine:
  id: copilot
network: defaults

# Ecosystems + custom domains
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Custom domain

# Custom domains (automatically includes subdomains)
network:
  allowed:
    - "api.example.com"      # Exact domain
    - "trusted.com"          # Includes all *.trusted.com subdomains

# No network access
network: {}
```

## Security Model

Network permissions follow the principle of least privilege with four access levels:

1. **Default Allow List** (`network: defaults`): Basic infrastructure only
2. **Selective Access** (`network: { allowed: [...] }`): Only listed domains/ecosystems are accessible
3. **No Access** (`network: {}`): All network access denied
4. **Automatic Subdomain Matching**: AWF automatically matches all subdomains of allowed domains (e.g., `github.com` allows `api.github.com`, `raw.githubusercontent.com`, etc.)

:::note
AWF does not support wildcard syntax like `*.example.com`. Instead, listing a domain automatically includes all its subdomains. Use `example.com` to allow access to `example.com`, `api.example.com`, `sub.api.example.com`, etc.
:::

## Content Sanitization

The `network:` configuration also controls which domains are allowed in sanitized content. URLs from domains not in the allowed list are replaced with `(redacted)` to prevent potential data exfiltration through untrusted links.

:::tip
If you see `(redacted)` in workflow outputs, add the domain to your `network.allowed` list. This applies the same domain allowlist to both network egress (when firewall is enabled) and content sanitization.
:::

GitHub domains (`github.com`, `githubusercontent.com`, etc.) are always allowed by default.


## Ecosystem Identifiers

Mix ecosystem identifiers with specific domains for fine-grained control:

| Identifier | Includes |
|------------|----------|
| `defaults` | Basic infrastructure (certificates, JSON schema, Ubuntu, package mirrors) |
| `github` | GitHub domains |
| `containers` | Docker Hub, GitHub Container Registry, Quay |
| `linux-distros` | Debian, Alpine, and other Linux package repositories |
| `dotnet`, `dart`, `go`, `haskell`, `java`, `node`, `perl`, `php`, `python`, `ruby`, `rust`, `swift` | Language-specific package managers and registries |
| `terraform` | HashiCorp and Terraform domains |
| `playwright` | Playwright testing framework domains |


## Implementation

Network permissions are enforced differently depending on the AI engine:

### Copilot Engine

The Copilot engine supports network permissions through AWF (Agent Workflow Firewall). AWF is a network firewall wrapper sourced from [github.com/githubnext/gh-aw-firewall](https://github.com/githubnext/gh-aw-firewall) that wraps Copilot CLI execution and enforces domain-based access controls.

Enable network permissions in your workflow:

```yaml wrap
engine: copilot

network:
  firewall: true           # Enable AWF enforcement
  allowed:
    - defaults             # Basic infrastructure
    - python              # Python ecosystem
    - "api.example.com"   # Custom domain
```

When enabled, AWF:
- Wraps the Copilot CLI execution command
- Enforces domain allowlisting using the `--allow-domains` flag
- Automatically includes all subdomains (e.g., `github.com` allows `api.github.com`)
- Logs all network activity for audit purposes
- Blocks access to domains not explicitly allowed

:::caution
AWF does not support wildcard syntax. Do not use patterns like `*.example.com`. Instead, list the base domain (e.g., `example.com`) which automatically includes all subdomains.
:::

### Firewall Log Level

Control the verbosity of AWF firewall logs using the `log-level` field:

```yaml wrap
network:
  firewall:
    log-level: info      # Options: debug, info, warn, error
  allowed:
    - defaults
    - python
```

Available log levels:
- `debug`: Detailed diagnostic information for troubleshooting
- `info`: General informational messages (default)
- `warn`: Warning messages for potential issues
- `error`: Error messages only

The default log level is `info`, which provides a balance between visibility and log volume. Use `debug` for troubleshooting network access issues or `error` to minimize log output.

See the [Copilot Engine - Network Permissions](/gh-aw/reference/engines/#network-permissions) documentation for detailed AWF configuration options.

### Disabling the Firewall

To disable the firewall for any engine that supports it while using network domain allowlists, set both `firewall: false` and `strict: false`:

```yaml wrap
strict: false
network:
  allowed:
    - defaults
    - python
    - "api.example.com"
  firewall: false
```

When the firewall is disabled with specific `allowed` domains:
- **Normal mode**: Compilation succeeds with a warning
- **Strict mode**: Compilation fails with an error

This configuration is useful during development or when the firewall is incompatible with your workflow requirements. For production workflows, enabling the firewall is recommended for better network security.

## Best Practices

Follow the principle of least privilege by only allowing access to domains and ecosystems actually needed. Prefer ecosystem identifiers over listing individual domains. When adding custom domains, use the base domain (e.g., `trusted.com`) which automatically includes all subdomains—do not use wildcard syntax like `*.trusted.com`.

## Troubleshooting

If you encounter network access denied errors, verify that required domains or ecosystems are included in the `allowed` list. Start with `network: defaults` and add specific requirements incrementally. Network access violations are logged in workflow execution logs.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
