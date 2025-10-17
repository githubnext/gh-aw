---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1300
---

Control network access for AI engines using the top-level `network` field to specify which domains and services your agentic workflows can access during execution.

> **Note**: Network permissions are currently only supported by the Claude engine.

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


## Best Practices

Follow the principle of least privilege by only allowing access to domains and ecosystems actually needed. Prefer ecosystem identifiers over broad wildcard patterns. Avoid overly permissive patterns like `"*"` or `"*.com"`.

## Troubleshooting

If you encounter network access denied errors, verify that required domains or ecosystems are included in the `allowed` list. Start with `network: defaults` and add specific requirements incrementally. Network access violations are logged in workflow execution logs.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
