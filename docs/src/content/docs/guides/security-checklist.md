---
title: Security Checklist
description: A quick reference checklist for securing your GitHub Agentic Workflows
sidebar:
  order: 4
---

Use this checklist to ensure your agentic workflows follow security best practices. Each item links to detailed documentation for deeper understanding.

## Pre-Deployment Checklist

### Permissions

- [ ] Use read-only `permissions:` for the main workflow job
- [ ] Use [safe outputs](/gh-aw/reference/safe-outputs/) for write operations instead of direct write permissions
- [ ] Configure `roles:` to restrict who can trigger the workflow
- [ ] Avoid `roles: all` in public repositories

```yaml wrap
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
  add-comment:
roles: [admin, maintainer]
```

### Network Security

- [ ] Configure explicit `network:` allowlists instead of using defaults
- [ ] Enable `firewall: true` for Copilot engine workflows
- [ ] Avoid wildcard domains in `network.allowed`
- [ ] Use ecosystem identifiers (`python`, `node`) over individual domains

```yaml wrap
network:
  firewall: true
  allowed:
    - defaults
    - python
```

### Input Sanitization

- [ ] Use `${{ needs.activation.outputs.text }}` instead of raw `github.event` fields
- [ ] Never interpolate untrusted input directly into shell commands
- [ ] Enable [threat detection](/gh-aw/guides/threat-detection/) for workflows that create content

```yaml wrap
# Secure: sanitized context
Analyze: "${{ needs.activation.outputs.text }}"

# Avoid: raw event fields
Analyze: "${{ github.event.issue.body }}"
```

### Supply Chain

- [ ] Pin GitHub Actions to specific commit SHAs, not tags
- [ ] Pin container images to digests (`@sha256:...`)
- [ ] Use `--strict` mode to enforce action pinning
- [ ] Review `.lock.yml` files before committing

```yaml wrap
# Secure: pinned to SHA
uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11

# Avoid: tag reference
uses: actions/checkout@v5
```

### Tool Configuration

- [ ] Use explicit `allowed:` lists for GitHub tools instead of allowing all
- [ ] Configure `bash:` with specific allowed commands, not wildcards
- [ ] Set `read-only: true` for GitHub tools when write access isn't needed

```yaml wrap
tools:
  github:
    read-only: true
    allowed: [issue_read, get_file_contents]
  bash: ["echo", "git status"]
```

### Workflow Limits

- [ ] Set appropriate `timeout-minutes:` to prevent runaway costs
- [ ] Configure `stop-after:` to limit workflow lifetime
- [ ] Set `max-turns:` in engine configuration to limit iterations
- [ ] Use `max:` limits in safe outputs to prevent excessive creation

```yaml wrap
timeout-minutes: 15
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+7d"
engine:
  id: copilot
  max-turns: 10
safe-outputs:
  create-issue:
    max: 3
```

## Production Deployment

### Strict Mode

- [ ] Enable `strict: true` in frontmatter or use `--strict` CLI flag
- [ ] Verify all actions are pinned to commit SHAs
- [ ] Ensure explicit network configuration exists
- [ ] Confirm no direct write permissions in main job

```yaml wrap
strict: true
permissions:
  contents: read
network:
  allowed:
    - defaults
    - "api.example.com"
```

### Security Scanning

- [ ] Run `gh aw compile --zizmor` to scan for vulnerabilities
- [ ] Run `gh aw compile --strict --zizmor` in CI/CD pipelines
- [ ] Address all High and Critical findings
- [ ] Review security scan reports before merging

```bash wrap
gh aw compile --strict --zizmor
```

### Token Management

- [ ] Use fine-grained PATs with minimal scopes
- [ ] Store tokens as repository secrets, never in code
- [ ] Configure token precedence appropriately
- [ ] Implement regular token rotation

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}
  create-issue:
```

### Fork Protection

- [ ] Keep default fork blocking for `pull_request` triggers
- [ ] Only allow specific trusted fork patterns when needed
- [ ] Use repository ID comparison for reliable fork detection

```yaml wrap
on:
  pull_request:
    types: [opened]
    forks: ["trusted-org/*"]
```

## Monitoring and Maintenance

### Ongoing Security

- [ ] Regularly audit workflow execution logs with `gh aw logs`
- [ ] Monitor for unusual patterns in workflow runs
- [ ] Keep dependencies updated
- [ ] Review and update allowlists periodically
- [ ] Document security decisions and exceptions

### Incident Response

- [ ] Know how to disable a workflow quickly
- [ ] Have a process for rotating compromised tokens
- [ ] Understand audit log locations
- [ ] Document escalation procedures

## Quick Reference Commands

```bash wrap
# Validate all workflows with strict mode
gh aw compile --strict

# Security scan with zizmor
gh aw compile --strict --zizmor

# Monitor workflow costs and execution
gh aw logs --start-date -1w

# Audit a specific workflow run
gh aw audit 12345678
```

## Related Documentation

- [Security Best Practices](/gh-aw/guides/security/) - Comprehensive security guide
- [Threat Detection](/gh-aw/guides/threat-detection/) - Configure threat detection
- [Strict Mode](/gh-aw/reference/frontmatter/#strict-mode-strict) - Strict mode reference
- [Network Access](/gh-aw/reference/network/) - Network isolation configuration
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure write operations
