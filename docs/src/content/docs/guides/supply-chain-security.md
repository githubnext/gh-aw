---
title: Supply Chain Security
description: Protect your agentic workflows from supply chain attacks through dependency pinning, container image verification, and trusted action selection
sidebar:
  order: 5
---

Supply chain attacks target the dependencies and tools your workflows rely on. GitHub Agentic Workflows provides mechanisms to protect against these threats through dependency pinning, strict mode enforcement, and verification practices.

## Understanding Supply Chain Risks

Agentic workflows have several dependency types that can be targeted:

1. **GitHub Actions**: Third-party actions referenced in generated `.lock.yml` files
2. **Container Images**: Docker images used by MCP servers
3. **NPM Packages**: Runtime packages installed during workflow execution
4. **Python Packages**: pip/uv packages for Python-based tools

Each dependency type presents opportunities for attackers to inject malicious code through compromised repositories, typosquatting, or dependency confusion.

## Pinning GitHub Actions

### The Risk

Tag-based action references like `actions/checkout@v5` can be updated by maintainers at any time. An attacker who compromises a repository can move a tag to point to malicious code.

```yaml wrap
# Vulnerable: Tag can be redirected
uses: actions/checkout@v5

# Secure: Immutable SHA reference
uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11
```

### Enforcing SHA Pinning

Enable strict mode to require all actions are pinned to commit SHAs:

```yaml wrap
strict: true
```

Or use the CLI flag during compilation:

```bash wrap
gh aw compile --strict
```

Strict mode causes compilation to fail if any action uses tag or branch references instead of commit SHAs.

### Finding Action SHAs

To find the commit SHA for a specific action version:

```bash wrap
# Using git
git ls-remote https://github.com/actions/checkout v5

# Using GitHub API
gh api repos/actions/checkout/git/refs/tags/v5 --jq '.object.sha'
```

Add a comment with the version for maintainability:

```yaml wrap
uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v5.0.0
```

## Pinning Container Images

### The Risk

Container tags like `latest` or version tags can be overwritten. Even "immutable" version tags can be deleted and recreated with different content.

```yaml wrap
# Vulnerable: Tag can be overwritten
mcp-servers:
  fetch:
    container: mcp/fetch:latest

# Secure: Digest reference
mcp-servers:
  fetch:
    container: mcp/fetch@sha256:abc123def456...
```

### Using Image Digests

Pin container images to their digest (SHA256 hash) instead of tags:

```yaml wrap
tools:
  web:
    mcp:
      container: "ghcr.io/example/web-mcp@sha256:abc123def456789..."
    allowed: [fetch]
```

To find an image digest:

```bash wrap
# Docker
docker pull mcp/fetch:v1.0.0
docker images --digests mcp/fetch

# Skopeo (doesn't require pulling)
skopeo inspect docker://mcp/fetch:v1.0.0 | jq -r '.Digest'
```

### Container Verification

For critical workflows, verify container images before use:

1. **Use trusted registries**: Prefer `ghcr.io` for GitHub-hosted images
2. **Scan for vulnerabilities**: Integrate tools like Trivy or Grype
3. **Track SBOMs**: Maintain software bill of materials for container contents
4. **Monitor for updates**: Set up Dependabot or Renovate for container dependencies

## NPM Package Security

### Compilation-Time Validation

The gh-aw compiler validates NPM packages during workflow compilation. Packages are checked against the npm registry for existence and version availability.

### Version Pinning

Specify exact versions for NPM packages:

```yaml wrap
runtimes:
  node:
    version: "22"
```

### Private Registries

For sensitive environments, consider using a private npm registry or npm proxy that only allows pre-approved packages.

## Python Package Security

### Compilation-Time Validation

Python packages specified in workflows are validated against PyPI during compilation. Invalid packages or versions cause compilation failure.

### Pinning Python Packages

When workflows require Python packages, use exact version specifications:

```yaml wrap
runtimes:
  python:
    version: "3.12"
```

## Security Scanning

### Automated Scanning

Use zizmor to scan compiled workflows for supply chain issues:

```bash wrap
# Scan all workflows
gh aw compile --zizmor

# Strict mode: fail on any findings
gh aw compile --strict --zizmor
```

zizmor detects:
- Unpinned action references
- Excessive permissions
- Insecure practices
- Known vulnerable patterns

### CI/CD Integration

Add supply chain scanning to your CI/CD pipeline:

```yaml wrap
# Example GitHub Actions job
- name: Compile and Scan Workflows
  run: gh aw compile --strict --zizmor
```

### Vulnerability Monitoring

Enable Dependabot for your repository to receive alerts about vulnerable dependencies:

```bash wrap
gh aw compile --dependabot
```

This generates manifests that Dependabot can monitor for security updates.

## Trusted Sources

### Evaluating Actions

Before using third-party actions:

1. **Check the publisher**: Prefer actions from `actions/`, `github/`, or verified creators
2. **Review the code**: Examine the action's source repository
3. **Check maintenance**: Look for recent commits and issue responses
4. **Review permissions**: Understand what the action requires access to
5. **Search for audits**: Look for security reviews or CVE reports

### Approved Action Lists

For organization-wide security, maintain an approved list of actions and enforce it through:

1. Repository rulesets that restrict action sources
2. Custom compilation validators
3. Pre-commit hooks that check action references

## Lock File Review

### Understanding Lock Files

Compiled `.lock.yml` files contain the actual workflow that runs. Always review these files before committing:

```bash wrap
# Show changes in lock files
git diff --name-only | grep '.lock.yml'

# Review a specific lock file
cat .github/workflows/my-workflow.lock.yml
```

### What to Check

When reviewing lock files:

1. **Action references**: Verify SHAs match expected versions
2. **Permissions**: Confirm minimal required permissions
3. **Network access**: Review allowed domains
4. **Environment variables**: Check for unexpected secrets or tokens
5. **Steps**: Verify no unexpected commands or scripts

## Incident Response

### Compromised Dependency

If a dependency is compromised:

1. **Immediately disable affected workflows**
2. **Review audit logs** for unexpected executions
3. **Pin to last known good version** using SHA/digest
4. **Notify your security team**
5. **Monitor for unauthorized changes** in your repository

### Updating After Security Fixes

When dependencies release security fixes:

1. **Verify the fix** in the upstream repository
2. **Update the SHA/digest** reference
3. **Re-run security scans**
4. **Test in staging** before production deployment

## Related Documentation

- [Security Best Practices](/gh-aw/guides/security/) - Comprehensive security guide
- [Security Checklist](/gh-aw/guides/security-checklist/) - Quick security verification
- [Strict Mode](/gh-aw/reference/frontmatter/#strict-mode-strict) - Strict mode enforcement
- [Network Access](/gh-aw/reference/network/) - Network isolation configuration
