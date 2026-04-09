---
# Base configuration for AI-powered security analysis workflows.
# Provides: code security tools, bash access, and security-events permissions.

permissions:
  security-events: read

tools:
  github:
    toolsets: [repos, code_security]
  bash: true

features:
  copilot-requests: true
---

## Security Analysis Configuration

Standard tooling for security scanning agents is configured.

### Available Tools

- **GitHub code security toolset** — access code scanning alerts, secret scanning, dependabot alerts
- **GitHub repos toolset** — read repository files and commits
- **`bash: true`** — full bash access for running scanners, examining files, and scripting

### Security Scanning Guidelines

1. **Scope carefully** — analyze only the declared time window (e.g., last 24h commits)
2. **Report findings** — use `create-code-scanning-alert` or `create-issue` as appropriate
3. **Avoid false positives** — cross-reference findings with git blame and PR context before alerting
4. **Cache results** — store scan state in `cache-memory` to enable incremental analysis
