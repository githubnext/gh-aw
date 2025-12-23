---
title: Network Configuration Guide
description: Common network configurations for package registries, CDNs, and development tools
sidebar:
  order: 450
---

This guide provides practical examples and best practices for configuring network access in GitHub Agentic Workflows. Use these patterns to reduce firewall denials while maintaining security.

## Quick Start

If you're experiencing firewall denials for legitimate package installations or dependency resolution, start with these common configurations:

### Python Projects

```yaml
network:
  allowed:
    - defaults       # Basic infrastructure
    - python        # PyPI and conda registries
```

The `python` ecosystem includes:
- `pypi.org` - Python Package Index
- `files.pythonhosted.org` - PyPI file hosting
- `pip.pypa.io` - pip installer
- `*.pythonhosted.org` - PyPI CDN domains
- `anaconda.org` - Conda packages
- Additional Python ecosystem domains

### Node.js Projects

```yaml
network:
  allowed:
    - defaults       # Basic infrastructure
    - node          # npm, yarn, pnpm, and Node.js
```

The `node` ecosystem includes:
- `registry.npmjs.org` - npm registry
- `npmjs.com` - npm website
- `nodejs.org` - Node.js downloads
- `yarnpkg.com` - Yarn package manager
- `get.pnpm.io` - pnpm installer
- `bun.sh` - Bun runtime
- `deno.land` - Deno runtime
- Additional Node.js ecosystem domains

### Go Projects

```yaml
network:
  allowed:
    - defaults       # Basic infrastructure
    - go            # Go module proxy and registries
```

The `go` ecosystem includes:
- `proxy.golang.org` - Go module proxy
- `sum.golang.org` - Go checksum database
- `go.dev` - Go documentation
- `golang.org` - Go language site
- `pkg.go.dev` - Package documentation

### Docker/Container Projects

```yaml
network:
  allowed:
    - defaults       # Basic infrastructure
    - containers    # Docker Hub, GHCR, Quay, etc.
```

The `containers` ecosystem includes:
- `registry.hub.docker.com` - Docker Hub
- `*.docker.io` - Docker domains
- `ghcr.io` - GitHub Container Registry
- `quay.io` - Quay container registry
- `gcr.io` - Google Container Registry
- `mcr.microsoft.com` - Microsoft Container Registry
- Additional container registry domains

## Multi-Language Projects

For projects using multiple languages or tools, combine ecosystem identifiers:

```yaml
network:
  allowed:
    - defaults       # Basic infrastructure
    - python        # Python dependencies
    - node          # JavaScript dependencies
    - containers    # Docker images
```

## Full-Stack Development

A comprehensive configuration for full-stack development with multiple tools:

```yaml
network:
  allowed:
    - defaults          # Basic infrastructure
    - github           # GitHub API and resources
    - node             # npm, yarn, pnpm
    - python           # PyPI, conda
    - containers       # Docker registries
    - playwright       # Browser testing
```

## Language-Specific Configurations

### .NET Projects

```yaml
network:
  allowed:
    - defaults
    - dotnet        # NuGet and .NET ecosystem
```

Includes: `nuget.org`, `dotnet.microsoft.com`, `api.nuget.org`, and related domains.

### Java Projects

```yaml
network:
  allowed:
    - defaults
    - java          # Maven, Gradle, and Java registries
```

Includes: `repo.maven.apache.org`, `gradle.org`, `repo1.maven.org`, and related domains.

### Ruby Projects

```yaml
network:
  allowed:
    - defaults
    - ruby          # RubyGems and Bundler
```

Includes: `rubygems.org`, `api.rubygems.org`, `bundler.rubygems.org`, and related domains.

### Rust Projects

```yaml
network:
  allowed:
    - defaults
    - rust          # Crates.io and Cargo
```

Includes: `crates.io`, `static.crates.io`, `static.rust-lang.org`, and related domains.

## Infrastructure and Platform Tools

### Terraform/HashiCorp

```yaml
network:
  allowed:
    - defaults
    - terraform     # Terraform registry and HashiCorp
```

### Linux Package Managers

```yaml
network:
  allowed:
    - defaults
    - linux-distros  # Debian, Ubuntu, Alpine, etc.
```

## Custom Domains

Add specific domains alongside ecosystem identifiers:

```yaml
network:
  allowed:
    - defaults
    - python
    - "api.example.com"      # Your API
    - "cdn.example.com"      # Your CDN
```

:::tip
Domain matching is automatic for subdomains. Listing `example.com` allows access to all subdomains like `api.example.com`, `cdn.example.com`, etc.
:::

## Testing and Browser Automation

For workflows using Playwright or web testing:

```yaml
network:
  allowed:
    - defaults
    - playwright    # Playwright browser downloads
    - node          # npm dependencies
    - "*.example.com"  # Sites to test
```

## Security Best Practices

### Start Minimal

Begin with only the ecosystems you need:

```yaml
# ✅ Good: Only what's needed
network:
  allowed:
    - defaults
    - python
```

```yaml
# ❌ Avoid: Too permissive
network:
  allowed:
    - defaults
    - python
    - node
    - java
    - dotnet
    - ruby
    - rust
    # ... when you only use Python
```

### Use Ecosystem Identifiers

Prefer ecosystem identifiers over individual domains:

```yaml
# ✅ Good: Use ecosystem identifier
network:
  allowed:
    - python
```

```yaml
# ❌ Avoid: Listing individual domains
network:
  allowed:
    - "pypi.org"
    - "files.pythonhosted.org"
    - "pip.pypa.io"
    - "bootstrap.pypa.io"
    # ... missing many others
```

### Incremental Addition

If you encounter firewall denials, add ecosystems incrementally:

1. Start with `network: defaults`
2. Add specific ecosystem: `network: { allowed: [defaults, python] }`
3. Test and verify
4. Add more ecosystems only as needed

## Troubleshooting Firewall Denials

### Identifying Blocked Domains

Use `gh aw logs` or `gh aw audit` to see firewall activity:

```bash
gh aw logs --run-id <run-id>
```

Look for denied requests in the firewall log section:

```
🔥 Firewall Log Analysis
Denied Domains:
  ✗ registry.npmjs.org:443 (3 requests)
  ✗ pypi.org:443 (2 requests)
```

### Adding Missing Ecosystems

If you see denied domains for package registries:

1. **npm/Node.js domains** → Add `node` ecosystem
2. **PyPI/Python domains** → Add `python` ecosystem
3. **Docker/container domains** → Add `containers` ecosystem
4. **Go module proxy** → Add `go` ecosystem

### Complete Example

Before (experiencing denials):
```yaml
network: defaults
```

After (adding required ecosystems):
```yaml
network:
  allowed:
    - defaults
    - python       # Fixed PyPI denials
    - node         # Fixed npm denials
    - containers   # Fixed Docker Hub denials
```

## Common Patterns

### CI/CD Pipeline

```yaml
network:
  allowed:
    - defaults
    - github       # GitHub API
    - containers   # Docker images
    - python       # Python packages
```

### Data Science Project

```yaml
network:
  allowed:
    - defaults
    - python       # PyPI, conda
    - github       # GitHub datasets
```

### Web Development

```yaml
network:
  allowed:
    - defaults
    - node         # npm packages
    - playwright   # Browser testing
    - github       # GitHub resources
```

### DevOps Automation

```yaml
network:
  allowed:
    - defaults
    - terraform    # Infrastructure as code
    - containers   # Container registries
    - github       # GitHub API
```

## No Network Access

For workflows that don't need external network access:

```yaml
network: {}
```

This denies all network access except what's required by the AI engine itself.

## Ecosystem Contents Reference

To see all domains in an ecosystem, check the [ecosystem domains source](https://github.com/githubnext/gh-aw/blob/main/pkg/workflow/data/ecosystem_domains.json).

### Quick Reference

| Ecosystem | Primary Domains | Purpose |
|-----------|----------------|---------|
| `defaults` | certificates, JSON schema, Ubuntu mirrors | Basic infrastructure |
| `python` | pypi.org, files.pythonhosted.org | Python packages |
| `node` | registry.npmjs.org, yarnpkg.com | Node.js packages |
| `go` | proxy.golang.org, sum.golang.org | Go modules |
| `containers` | registry.hub.docker.com, ghcr.io | Container images |
| `java` | repo.maven.apache.org, gradle.org | Java dependencies |
| `dotnet` | nuget.org, dotnet.microsoft.com | .NET packages |
| `ruby` | rubygems.org | Ruby gems |
| `rust` | crates.io | Rust crates |
| `github` | githubusercontent.com, github.com | GitHub resources |

## Related Documentation

- [Network Permissions Reference](/gh-aw/reference/network/) - Complete network configuration reference
- [Security Guide](/gh-aw/guides/security/) - Security best practices
- [Troubleshooting](/gh-aw/troubleshooting/common-issues/) - Common issues and solutions
