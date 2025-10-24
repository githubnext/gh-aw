---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1300
---

Control network access for AI engines using the top-level `network` field to specify which domains and services your agentic workflows can access during execution.

> **Note**: Network permissions are currently supported by the Claude engine and the Copilot engine (when using the [firewall feature](/gh-aw/reference/engines/#network-firewall-awf)).

If no `network:` permission is specified, it defaults to `network: defaults` which allows access to basic infrastructure domains (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources).

## Configuration

```yaml
# Default: basic infrastructure only
engine:
  id: claude
network: defaults

# Ecosystems + custom domains
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Custom domain

# Domain patterns (exact match or wildcard)
network:
  allowed:
    - "api.example.com"      # Exact domain
    - "*.trusted.com"        # Wildcard (includes nested subdomains)

# No network access
network: {}
```

## Security Model

Network permissions follow the principle of least privilege with four access levels:

1. **Default Allow List** (`network: defaults`): Basic infrastructure only
2. **Selective Access** (`network: { allowed: [...] }`): Only listed domains/ecosystems are accessible
3. **No Access** (`network: {}`): All network access denied
4. **Domain Validation**: Supports exact matches and wildcard patterns (`*` matches nested subdomains)


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

### Claude Engine

The Claude engine uses hook-based enforcement via Claude Code's PreToolUse hooks to intercept network requests. This provides fine-grained control with minimal performance overhead (~10ms per request).

### Copilot Engine with AWF

The Copilot engine supports network permissions through the optional AWF (Agent Workflow Firewall) feature. AWF is a network firewall wrapper sourced from [github.com/githubnext/gh-aw-firewall](https://github.com/githubnext/gh-aw-firewall) that wraps Copilot CLI execution and enforces domain-based access controls.

Enable AWF in your workflow:

```yaml
features:
  firewall: true

engine: copilot

network:
  allowed:
    - defaults
    - "api.example.com"
```

When the firewall feature is enabled, AWF:
- Wraps the Copilot CLI execution command
- Enforces domain allowlisting using the `--allow-domains` flag
- Logs all network activity for audit purposes
- Blocks access to domains not explicitly allowed

See the [Copilot Engine - Network Firewall](/gh-aw/reference/engines/#network-firewall-awf) documentation for detailed AWF configuration options.

## Best Practices

Follow the principle of least privilege by only allowing access to domains and ecosystems actually needed. Prefer ecosystem identifiers over broad wildcard patterns. Avoid overly permissive patterns like `"*"` or `"*.com"`.

## Troubleshooting

If you encounter network access denied errors, verify that required domains or ecosystems are included in the `allowed` list. Start with `network: defaults` and add specific requirements incrementally. Network access violations are logged in workflow execution logs.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
