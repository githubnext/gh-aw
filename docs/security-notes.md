# Security Notes

> [!CAUTION]
> GitHub Agentic Workflows is a research demonstrator, and Agentic Workflows are not for production use.

Security is foundational -- Agentic Workflows inherits GitHub Actions' sandboxing model, scoped permissions, and auditable execution. The attack surface of agentic automation can be subtle (prompt injection, tool invocation side‑effects, data exfiltration), so we bias toward explicit constraints over implicit trust: least‑privilege tokens, allow‑listed tools, and execution paths that always leave human‑visible artifacts (comments, PRs, logs) instead of silent mutation.

A core reason for building Agentic Workflows as a research demonstrator is to closely track emerging security controls in agentic engines under near‑identical inputs, so differences in behavior and guardrails are comparable. Alongside engine evolution, we are working on our own mechanisms:
highly restricted substitutions, MCP proxy filtering, and hooks‑based security checks that can veto or require review before effectful steps run.

We aim for strong, declarative guardrails -- clear policies the workflow author can review and version -- rather than opaque heuristics. Lock files are fully reviewable so teams can see exactly what was resolved and executed. This will keep evolving; we would love to hear ideas and critique from the community on additional controls, evaluation methods, and red‑team patterns.

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

- **Command execution**: Agentic workflows are, executed in the partially-sandboxed environment of GitHub Actions. By default, they are configured to disallow the execution of arbitrary shell commands. However, they may optionally be manually configured to allow specific commands, and if so they will not ask for confirmation before executing these specific commands as part of the GitHub Actions workflow run. If these configuration options are used inappropriately, or on sensitive code, an attacker might use this capability to make the coding agent fetch and run malicious code to exfiltrate data or perform unauthorized execution within this environment.
- **Malicious inputs**: Attackers can craft inputs that poison an coding agent. Agentic workflows often pull data from many sources, including GitHub Issues, PRs, comments and code. If considered untrusted, e.g. in an open source setting, any of those inputs could carry a hidden payload for AI. Agentic workflows are designed to minimize the risk of malicious inputs by restricting the expressions that can be used in workflow markdown content. This means inputs such as GitHub Issues and Pull Requests must be accessed via the GitHub MCP, however the returned data can, in principle, be used to manipulate the AI's behavior if not properly assessed and sanitized.
- **Tool exposure**: By default, Agentic Workflows are configured to have no access to MCPs except the GitHub MCP in read-only mode. However unconstrained use of 3rd-party MCP tools can enable data exfiltration or privilege escalation.
- **Supply chain attacks and other generic GitHub Actions threats**: Unpinned Actions, npm packages and container images are vulnerable to tampering. These threats are generic to all GitHub Actions workflows, and Agentic Workflows are no exception.

### Core Security Principles

The fundamental principle of security for Agentic Workflows is that they are GitHub Actions workflows and should be reviewed with the same rigour and rules that are applied to all GitHub Actions. See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use).

This means they inherit the security model of GitHub Actions, which includes:

- **Isolated copy of the repository** - each workflow runs in a separate copy of the repository, so it cannot access other repositories or workflows
- **Read-only defaults** for forked PRs
- **Restricted secret access** - secrets are not available in forked PRs by default
- **Explicit permissions** - all permissions default to `none` unless explicitly set

In addition, the compilation step of Agentic Workflows enforces additional security measures:

- **Expression restrictions** - only a limited set of expressions are allowed in the workflow frontmatter, preventing arbitrary code execution
- **Highly restricted commands** - by default, no commands are allowed to be executed, and any commands that are allowed must be explicitly specified in the workflow
- **Explicit tool allowlisting** - only tools explicitly allowed in the workflow can be used
- **Engine network restrictions** - control network access for AI engines using domain allowlists
- **Limit workflow longevity** - workflows can be configured to stop triggering after a certain time period
- **Limit chat iterations** - workflows can be configured to limit the number of chat iterations per run, preventing runaway loops and excessive resource consumption

Apply these principles consistently across all workflow components:

1. **Least privilege by default** - elevate permissions only when required, scoped to specific jobs or steps
2. **Default-deny approach** - explicitly allowlist tools
3. **Separation of concerns** - implement "plan" and "apply" phases with approval gates for risky operations
4. **Supply chain integrity** - pin all dependencies (Actions, containers) to immutable SHAs

## Implementation Guidelines

### Workflow Permissions and Triggers

Configure GitHub Actions with defense in depth:

#### Permission Configuration

Set minimal permissions for the agentic processing:

```yaml
# Applies to the agentic processing
permissions:
  issues: write
  contents: read
```

### Human in the Loop

GitHub Actions workflows are designed to be steps within a larger process. Some critical operations should always involve human review:

- **Approval gates**: Use manual approval steps for high-risk operations like deployments, secret management, or external tool invocations
- **Pull requests require humans**: GitHub Actions cannot approve or merge pull requests. This means a human will always be involved in reviewing and merging pull requests that contain agentic workflows.
- **Plan-apply separation**: Implement a "plan" phase that generates a preview of actions before execution. This allows human reviewers to assess the impact of changes. This is usually done via an output issue or pull request.
- **Review and audit**: Regularly review workflow history, permissions, and tool usage to ensure compliance with security policies.

### Limit operations

#### Limit workflow longevity by `stop-after:`

Use `stop-after:` in the `on:` section to limit the time of operation of an agentic workflow. For example, using

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+7d"
```

will mean the agentic workflow no longer operates 7 days after time of compilation.

#### Limit workflow runs by engine `max-turns:`

Use `max-turns:` in the engine configuration to limit the number of chat iterations per run. This prevents runaway loops and excessive resource consumption. For example:

```yaml
engine:
  id: claude
  max-turns: 5
```

This limits the workflow to a maximum of 5 interactions with the AI engine per run.

#### Monitor costs by `gh aw logs`

Use `gh aw logs` to monitor the costs of running agentic workflows. This command provides insights into the number of turns, tokens used, and other metrics that can help you understand the cost implications of your workflows. Reported information may differ based on the AI engine used (e.g., Claude vs. Codex).

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

#### Tool Allow/Disallow Examples

Configure explicit allow-lists for tools. See also `docs/tools.md` for full options.

- Minimal GitHub tool set (read + specific writes):

```yaml
tools:
  github:
    allowed: [get_issue, add_issue_comment]
```

- Restricted Claude bash and editing:

```yaml
engine: claude
tools:
  claude:
    allowed:
      Edit:
      Write:
      Bash: ["echo", "git status"]   # keep tight; avoid wildcards
```

- Patterns to avoid:

```yaml
tools:
  github:
    allowed: ["*"]            # Too broad
  claude:
    allowed:
      Bash: [":*"]           # Unrestricted shell access
```

#### Egress Filtering

A critical guardrail is strict control over outbound network connections. Agentic Workflows now supports declarative network allowlists for containerized MCP servers.

Example (domain allowlist):

```yaml
tools:
  fetch:
    mcp:
      type: stdio
      container: mcp/fetch
      permissions:
        network:
          allowed:
            - "example.com"
    allowed: ["fetch"]
```

Enforcement details:

- Compiler generates a per‑tool Squid proxy and Docker network; MCP egress is forced through the proxy via iptables.
- Only listed domains are reachable; all others are denied at the network layer.
- Applies to `mcp.container` stdio servers. Non‑container stdio and `type: http` servers are not supported and will cause compilation errors.

Operational guidance:

- Use bare domains (no scheme). Explicitly list each domain you intend to permit.
- Prefer minimal allowlists; review the compiled `.lock.yml` to verify proxy setup and rules.

### Agent Security and Prompt Injection Defense

Protect against model manipulation through layered defenses:

#### Policy Enforcement

- **Input sanitization**: Minimize untrusted content exposure; strip embedded commands when not required for functionality
- **Action validation**: Implement a plan-validate-execute flow where policy layers check each tool call against risk thresholds

## Engine Network Permissions

### Overview

Engine network permissions provide fine-grained control over network access for AI engines themselves, separate from MCP tool network permissions. This feature uses Claude Code's hook system to enforce domain-based access controls.

### Security Benefits

1. **Defense in Depth**: Additional layer beyond MCP tool restrictions
2. **Compliance**: Meet organizational security requirements for AI network access
3. **Audit Trail**: Network access attempts are logged through Claude Code hooks
4. **Principle of Least Privilege**: Only grant network access to required domains

### Implementation Details

- **Hook-Based Enforcement**: Uses Claude Code's PreToolUse hooks to intercept network requests
- **Runtime Validation**: Domain checking happens at request time, not compilation time
- **Error Handling**: Blocked requests receive clear error messages with allowed domains
- **Performance Impact**: Minimal overhead (~10ms per network request)

### Best Practices

1. **Always Specify Permissions**: When using network features, explicitly list allowed domains
2. **Use Defaults When Appropriate**: Use `"defaults"` in the allowed list to include common development domains, then add custom ones
3. **Use Wildcards Carefully**: `*.example.com` matches any subdomain including nested ones (e.g., `api.example.com`, `nested.api.example.com`) - ensure this broad access is intended
4. **Test Thoroughly**: Verify that all required domains are included in allowlist
5. **Monitor Usage**: Review workflow logs to identify any blocked legitimate requests
6. **Document Reasoning**: Comment why specific domains are required for maintenance

### Permission Modes

1. **No network permissions**: Unrestricted access (backwards compatible)
   ```yaml
   engine:
     id: claude
     # No network block - full network access
   ```

2. **Empty allowed list**: Complete network access denial
   ```yaml
   engine:
     id: claude

   network:
     allowed: []  # Deny all network access
   ```

3. **Specific domains**: Granular access control to listed domains only
   ```yaml
   engine:
     id: claude

   network:
     allowed:
       - "api.github.com"
       - "*.company-internal.com"
   ```

## Engine Security Notes

Different agentic engines have distinct defaults and operational surfaces.

#### `engine: claude`

- Restrict `claude.allowed` to only the needed capabilities (Edit/Write/WebFetch/Bash with a short list)
- Keep `allowed_tools` minimal in the compiled step; review `.lock.yml` outputs
- Use engine network permissions to restrict WebFetch and WebSearch to required domains only

#### Security posture differences with Codex

Claude exposes richer default tools and optional Bash; codex relies more on CLI behaviors. In both cases, tool allow-lists, network restrictions, and pinned dependencies are your primary controls.

## See also

- [Tools Configuration](tools.md)
- [MCPs](mcps.md)
- [Secrets Management](secrets.md)
- [Workflow Structure](workflow-structure.md)

## References

- Model Context Protocol: Security Best Practices (2025-06-18) — <https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices>
