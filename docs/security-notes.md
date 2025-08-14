# Security Notes

> [!CAUTION]
> GitHub Agentic Workflows is a research demonstrator, and Agentic Workflows are not for production use.

This material documents some notes on the security of using partially-automated agentic workflows.

## Before You Begin

When working with agentic workflows, thorough review is essential:

1. **Review workflow contents** before installation, particularly third-party workflows that may contain unexpected automation. Treat prompt templates and rule files as code.
2. **Assess compiled workflows** (`.lock.yml` files) to understand the actual permissions and operations being performed
3. **Understand GitHub's security model** - GitHub Actions provides built-in protections like read-only defaults for fork PRs and restricted secret access. These apply to agentic workflows as well. See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use) and [permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions)
4. **Remember permission defaults** - when you specify any permission explicitly, all unspecified permissions default to `none`

## Threat Model

Understanding the security risks in agentic workflows helps inform protective measures:

### Primary Threats

- **Command execution**: Agentic workflows are, executed in the partially-sandboxed environment of GitHub Actions. By default, they are configured disallow the execution of arbitrary shell commands. However, they may optionally be manually configured to allow specific commands, and if so they will not ask for confirmation before executing these specific commands as part of the GitHub Actions workflow run. If these configuration options are used inappropriately, or on sensitive code, an attacker might use this capability to make the agent fetch and run malicious code to exfiltrate data or perform unauthorized execution within this environment.
- **Malicious inputs**: Attackers can craft inputs that poison an agent. Agentic workflows often pull data from many sources, including GitHub Issues, PRs, comments and code. If considered untrusted, e.g. in an open source setting, any of those inputs could carry a hidden payload for AI. Agentic workflows are designed to minimize the risk of malicious inputs by restricting the expressions that can be used in workflow markdown content. This means inputs such as GitHub Issues and Pull Requests must be accessed via the GitHub MCP, however the returned data can, in principle, be used to manipulate the AI's behavior if not properly assessed and sanitized.
- **Tool exposure**: By default, Agentic Workflows are configured to have no access to MCPs except the GitHub MCP in read-only mode. However unconstrained use of 3rd-party MCP tools can enable data exfiltration or privilege escalation.
- **Supply chain attacks and other generic GitHub Actions threats**: Unpinned Actions, npm packages and container images are vulnerable to tampering

### Core Security Principles

The fundamental principle of security for Agentic Workflows is that they are GitHub Actions workflows and should be reviewed with the same rigour and rules that are applied to all GitHub Actions. See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use).

This means they inherit the security model of GitHub Actions, which includes:
- **Isolated copy of the repository** - each workflow runs in a separate copy of the repository, so it cannot access other repositories or workflows
- **Read-only defaults** for forked PRs
- **Restricted secret access** - secrets are not available in forked PRs by default
- **Explicit permissions** - all permissions default to `none` unless explicitly set

In addition, the compilation step of Agentic Workflows enforces additional security measures: 
- **Expression restrictions** - only a limited set of expressions are allowed in the workflow frontmatter, preventing arbitrary code execution
- **Tool allowlisting** - only explicitly allowed tools can be used in the workflow
- **Highly restricted commands** - by default, no commands are allowed to be executed, and any commands that are allowed must be explicitly specified in the workflow
- **Explicit tool allowlisting** - only tools explicitly allowed in the workflow can be used

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

### Human in the Loop

GitHub Actions workflows are not designed to be fully autonomous. Some critical operations should always involve human review:
- **Approval gates**: Use manual approval steps for high-risk operations like deployments, secret management, or external tool invocations
- **Pull requests require humans**: GitHub Actions cannot approve or merge pull requests. This means a human will always be involved in reviewing and merging pull requests that contain agentic workflows.
- **Plan-apply separation**: Implement a "plan" phase that generates a preview of actions before execution. This allows human reviewers to assess the impact of changes. This is usually done via a pull request.
- **Review and audit**: Regularly review workflow history, permissions, and tool usage to ensure compliance with security policies.

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
