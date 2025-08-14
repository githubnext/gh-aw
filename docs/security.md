# Security Best Practices

This guide provides security recommendations for Agentic Workflows (AW). The fundamental principle: treat all user-supplied inputs as untrusted, enforcing guardrails through code and configuration rather than prompts alone.

## Before You Begin

When working with agentic workflows, thorough review is essential:

1. **Review workflow contents** before installation, particularly third-party workflows that may contain unexpected automation. Treat prompt templates and rule files as code.
2. **Assess compiled workflows** (`.lock.yml` files) to understand the actual permissions and operations being performed
3. **Understand GitHub's security model** - GitHub Actions provides built-in protections like read-only defaults for fork PRs and restricted secret access. These apply to agentic workflows as well. See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use) and [permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions)
4. **Remember permission defaults** - when you specify any permission explicitly, all unspecified permissions default to `none`

## Threat Model

Understanding the security risks in agentic workflows helps inform protective measures:

### Primary Threats

- **Prompt injection and malicious inputs**: Attackers can craft inputs that poison an agent. Agentic workflows often pull data from many sources, including GitHub Issues, PRs, comments, code, and external APIs, so any of those inputs could carry a hidden trigger for AI.
- **Automated execution without review**: Unlike IDEs, agentic workflows may execute code and call tools automatically. If not tightly controlled, an attacker might make the agent fetch and run malicious code.
- **Tool exposure**: Unconstrained MCP tools (filesystem, network) can enable data exfiltration or privilege escalation
- **Supply chain attacks**: Unpinned Actions, npm packages and container images are vulnerable to tampering

### Core Security Principles

Apply these principles consistently across all workflow components:

1. **Least privilege by default** - elevate permissions only when required, scoped to specific jobs or steps
2. **Default-deny approach** - explicitly allowlist tools
3. **Separation of concerns** - implement "plan" and "apply" phases with approval gates for risky operations
4. **Supply chain integrity** - pin all dependencies (Actions, containers) to immutable SHAs

## Implementation Guidelines

### Workflow Permissions and Triggers

Configure GitHub Actions with defense in depth:

#### Permission Configuration

Set minimal top-level permissions and elevate only where necessary:

```yaml
permissions:
  contents: read  # Minimal baseline
  # All others default to none

jobs:
  comment:
    permissions:
      issues: write  # Job-scoped elevation
```

### MCP Tool Hardening

Model Context Protocol tools require strict containment:

#### Sandboxing and Isolation

Run MCP servers in explicit sandboxes to constrain blast radius:

- Container isolation: Prefer running each MCP server in its own container with no shared state between workflows, repos, or users.
- Non-root, least-capability: Use non-root UIDs, drop Linux capabilities, and apply seccomp/AppArmor where supported. Disable privilege escalation.
- Supply-chain sanity: Use pinned images/binaries (digest/SHAs), run vulnerability scans, and track SBOMs for MCP containers.

Example (pinned container with minimal allowances):

```yaml
tools:
  web:
    mcp:
      container: "ghcr.io/example/web-mcp@sha256:abc123..."  # Pinned image digest
    allowed: [fetch]
```

#### Access Control Strategy

Start with empty allowlists and grant permissions incrementally:

```yaml
tools:
  github:
    allowed: [add_issue_comment]  # Minimal required verbs
  web:
    mcp:
      allowed: [fetch]
  container: my-registry/my-image@sha256:abc123...
```

#### Egress Filtering

A critical guardrail is strict control over outbound network connections. Consider using network proxies to enforce allowlists for outbound hosts.

### Agent Security and Prompt Injection Defense

Protect against model manipulation through layered defenses:

#### Policy Enforcement

- **Input sanitization**: Minimize untrusted content exposure; strip embedded commands when not required for functionality
- **Action validation**: Implement a plan-validate-execute flow where policy layers check each tool call against risk thresholds

## See also

- [Tools Configuration](tools.md)
- [MCPs](mcps.md)
- [Secrets Management](secrets.md)

## References

- Model Context Protocol: Security Best Practices (2025-06-18) — <https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices>
