---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1200
---

Control network access for AI engines using the top-level `network` field. Network permissions provide fine-grained control over which domains and services your agentic workflows can access during execution.

> **Note**: Network permissions are currently only supported by the Claude engine.

If no `network:` permission is specified, it defaults to `network: defaults` which uses a curated allow-list of common development and package manager domains for basic infrastructure needs.

### Basic Configuration

```yaml
# Default allow-list (basic infrastructure only)
engine:
  id: claude

network: defaults
```

### Ecosystem-Based Configuration

```yaml
# Use ecosystem identifiers + custom domains
engine:
  id: claude

network:
  allowed:
    - defaults              # Basic infrastructure (certs, JSON schema, Ubuntu, etc.)
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Custom domain
```

### Domain-Only Configuration

```yaml
# Allow specific domains only (no ecosystems)
engine:
  id: claude

network:
  allowed:
    - "api.example.com"      # Exact domain match
    - "*.trusted.com"        # Wildcard matches any subdomain (including nested subdomains)
```

### Combined Configuration

```yaml
# Combine defaults with additional domains
engine:
  id: claude

network:
  allowed:
    - "defaults"             # Expands to the full default whitelist
    - "good.com"             # Add custom domain
    - "api.example.org"      # Add another custom domain
```

### No Network Access

```yaml
# Deny all network access (empty object)
engine:
  id: claude

network: {}
```

## Security Model

The network permissions system provides multiple layers of security control:

- **Default Allow List**: When no network permissions are specified or `network: defaults` is used, access is restricted to basic infrastructure domains only (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources)
- **Ecosystem Access**: Use ecosystem identifiers like `python`, `node`, `containers` to enable access to specific development ecosystems
- **Selective Access**: When `network: { allowed: [...] }` is specified, only listed domains/ecosystems are accessible
- **No Access**: When `network: {}` is specified, all network access is denied
- **Domain Validation**: Supports exact matches and wildcard patterns (`*` matches any characters including dots, allowing nested subdomains)

## Configuration Examples

### Development Environment Examples

```yaml
# Default infrastructure only (basic certificates, JSON schema, Ubuntu, etc.)
network: defaults

# Python development environment
network:
  allowed:
    - defaults             # Basic infrastructure
    - python              # Python/PyPI ecosystem
    - github              # GitHub domains

# Full-stack development with multiple ecosystems
network:
  allowed:
    - defaults
    - python
    - node
    - containers
    - dotnet
    - "api.custom.com"    # Custom domain
```

### Domain Pattern Examples

```yaml
# Allow all subdomains of a trusted domain
# Note: "*.github.com" matches api.github.com, subdomain.github.com, and even nested.api.github.com
network:
  allowed:
    - "*.company-internal.com"
    - "public-api.service.com"

# Specific ecosystems only (no basic infrastructure)
network:
  allowed:
    - "defaults"                    # Expands to full default whitelist
    - java
    - rust
    - "api.mycompany.com"           # Add custom API
    - "*.internal.mycompany.com"    # Add internal services
```

## Available Ecosystem Identifiers

The `network: { allowed: [...] }` format supports these ecosystem identifiers:

### Core Infrastructure
- **`defaults`**: Basic infrastructure (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources)
- **`github`**: GitHub domains (api.github.com, github.com, etc.)
- **`containers`**: Container registries (Docker Hub, GitHub Container Registry, Quay, etc.)
- **`linux-distros`**: Linux distribution package repositories (Debian, Alpine, etc.)

### Programming Language Ecosystems
- **`dotnet`**: .NET and NuGet ecosystem
- **`dart`**: Dart and Flutter ecosystem  
- **`go`**: Go ecosystem (golang.org, proxy.golang.org, etc.)
- **`haskell`**: Haskell ecosystem (hackage.haskell.org, etc.)
- **`java`**: Java ecosystem (Maven Central, Gradle, etc.)
- **`node`**: Node.js and NPM ecosystem (npmjs.org, nodejs.org, etc.)
- **`perl`**: Perl and CPAN ecosystem
- **`php`**: PHP and Composer ecosystem
- **`python`**: Python ecosystem (PyPI, Conda, etc.)
- **`ruby`**: Ruby and RubyGems ecosystem
- **`rust`**: Rust and Cargo ecosystem (crates.io, etc.)
- **`swift`**: Swift and CocoaPods ecosystem

### Specialized Tools
- **`terraform`**: HashiCorp and Terraform ecosystem
- **`playwright`**: Playwright testing framework domains

### Fine-Grained Control

You can mix ecosystem identifiers with specific domain names for fine-grained control:

```yaml
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python ecosystem
    - "api.custom.com"     # Custom domain
    - "*.internal.corp"    # Wildcard domain
```

## Domain Patterns

Network permissions support flexible domain matching patterns:

### Exact Matches
```yaml
network:
  allowed:
    - "api.example.com"     # Matches exactly api.example.com
    - "service.internal"    # Matches exactly service.internal
```

### Wildcard Patterns
```yaml
network:
  allowed:
    - "*.example.com"       # Matches any subdomain of example.com
    - "*.internal.corp"     # Matches any subdomain of internal.corp
```

**Important**: Wildcard patterns (`*`) match any characters including dots, allowing nested subdomains. For example, `*.github.com` matches:
- `api.github.com`
- `subdomain.github.com` 
- `nested.api.github.com`

## Security Considerations

### Best Practices
- **Principle of Least Privilege**: Only allow access to domains and ecosystems actually needed
- **Use Ecosystem Identifiers**: Prefer ecosystem identifiers over broad wildcard patterns
- **Validate Custom Domains**: Ensure custom domains are trusted and necessary
- **Regular Review**: Periodically review network permissions to remove unused access

### Common Patterns
```yaml
# Recommended: Specific ecosystems for your stack
network:
  allowed:
    - defaults
    - python              # Only if using Python
    - node                # Only if using Node.js
    - "api.myservice.com" # Only specific APIs needed

# Avoid: Overly broad patterns
network:
  allowed:
    - "*"                 # Too permissive
    - "*.com"             # Too broad
```

## Troubleshooting

### Common Issues

**Network access denied errors**: Check that required domains or ecosystems are included in the `allowed` list.

**Wildcard not matching**: Ensure wildcard patterns use `*` correctly and understand that they match nested subdomains.

**Ecosystem not working**: Verify the ecosystem identifier is spelled correctly and supported.

### Debugging Tips

1. **Start with defaults**: Begin with `network: defaults` and add specific requirements
2. **Use specific domains**: Test with exact domain names before using wildcards
3. **Check logs**: Network access violations are logged in workflow execution logs
4. **Incremental permissions**: Add permissions incrementally rather than all at once

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools Configuration](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
