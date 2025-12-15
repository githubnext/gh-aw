---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1300
---

Control network access for AI engines using the top-level `network` field to specify which domains and services your agentic workflows can access during execution. Currently supported by the Claude engine and the Copilot engine (when using the [firewall feature](/gh-aw/reference/engines/#network-permissions)).

Defaults to `network: defaults` (basic infrastructure: certificates, JSON schema, Ubuntu, package mirrors, Microsoft sources).

## Configuration

```yaml wrap
# Default (basic infrastructure only)
network: defaults

# Ecosystems + custom domains (subdomains auto-included)
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Includes all subdomains
    - "trusted.com"        # Includes *.trusted.com

# No network access
network: {}
```

## Security Model

Network permissions follow least privilege with three access levels:

1. **Default Allow List** (`network: defaults`): Basic infrastructure only
2. **Selective Access** (`network: { allowed: [...] }`): Only listed domains/ecosystems
3. **No Access** (`network: {}`): All network access denied

Domains automatically include all subdomains (e.g., `github.com` allows `api.github.com`, `raw.githubusercontent.com`). No wildcard syntax like `*.example.com` is supported.

## Content Sanitization

URLs from non-allowed domains are replaced with `(redacted)` in workflow outputs to prevent data exfiltration. If you see `(redacted)`, add the domain to your `network.allowed` list. GitHub domains are always allowed.


## Ecosystem Identifiers

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

When enabled, AWF wraps Copilot CLI execution, enforces domain allowlisting, logs network activity, and blocks non-allowed domains.

### Firewall Log Level

```yaml wrap
network:
  firewall:
    log-level: info      # debug | info (default) | warn | error
  allowed:
    - defaults
    - python
```

Use `debug` for troubleshooting, `error` to minimize output.

See the [Copilot Engine - Network Permissions](/gh-aw/reference/engines/#network-permissions) documentation for detailed AWF configuration options.

### Disabling the Firewall

Disable the firewall using `sandbox.agent: false`:

```yaml wrap
engine: copilot
network:
  allowed:
    - defaults
    - python
sandbox:
  agent: false
```

When disabled, network permissions still apply for content sanitization but not for firewall enforcement. Recommended for production: enable the firewall.

## Best Practices

Follow least privilege: only allow needed domains/ecosystems. Prefer ecosystem identifiers over individual domains.

## Troubleshooting

Network access denied errors: verify domains/ecosystems are in the `allowed` list. Start with `network: defaults` and add requirements incrementally. Check workflow logs for violations.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
