---
title: Security Guide
description: Important security considerations for GitHub Agentic Workflows, including sandboxing, permissions, and best practices for safe agentic automation.
sidebar:
  order: 100
---

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.

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
5. **Check repository access restrictions** - By default, agentic workflows restrict execution to users with `admin` or `maintainer` repository permissions. Use `roles: all` carefully, especially in public repositories where any user can potentially trigger workflows

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

**Production workflows**: Consider using strict mode to enforce additional security constraints:

```yaml
# Enable strict mode for production workflows
strict: true
permissions:
  contents: read  # Write permissions are blocked in strict mode
timeout_minutes: 10
network:
  allowed:
    - "api.example.com"
```

Strict mode prevents write permissions (`contents:write`, `issues:write`, `pull-requests:write`) and requires explicit network configuration. Use `safe-outputs` configuration instead for controlled GitHub API interactions. See [Strict Mode Validation](#strict-mode-validation) for details.

### Human in the Loop

GitHub Actions workflows are designed to be steps within a larger process. Some critical operations should always involve human review:

- **Approval gates**: Use manual approval steps for high-risk operations like deployments, secret management, or external tool invocations
- **Pull requests require humans**: GitHub Actions cannot approve or merge pull requests. This means a human will always be involved in reviewing and merging pull requests that contain agentic workflows.
- **Plan-apply separation**: Implement a "plan" phase that generates a preview of actions before execution. This allows human reviewers to assess the impact of changes. This is usually done via an output issue or pull request.
- **Review and audit**: Regularly review workflow history, permissions, and tool usage to ensure compliance with security policies.

### Limit operations

#### Strict Mode Validation

For production workflows, use strict mode to enforce enhanced security and reliability constraints:

```yaml
# Enable strict mode declaratively in frontmatter
strict: true
permissions:
  contents: read
network:
  allowed:
    - "api.example.com"
```

Or enable for all workflows during compilation:

```bash
gh aw compile --strict
```

**Strict mode enforces:**

1. **Write Permissions Blocked**: Refuses `contents:write`, `issues:write`, and `pull-requests:write` (use `safe-outputs` instead)
2. **Network Configuration Required**: Must explicitly configure network access (cannot rely on defaults)
3. **No Network Wildcards**: Cannot use wildcard `*` in `network.allowed` domains
4. **MCP Network Configuration**: Custom MCP servers with containers must have network configuration

**Benefits:**
- **Security**: Minimizes attack surface by blocking write permissions and requiring explicit network access
- **Compliance**: Ensures workflows meet organizational security standards
- **Auditability**: Clear security requirements make workflows easier to review

**Behavior:**
- CLI `--strict` flag applies to all workflows during compilation
- Frontmatter `strict: true` enables strict mode for individual workflows
- CLI flag takes precedence over frontmatter settings
- Default is non-strict mode for backward compatibility

See the [Frontmatter Reference](/gh-aw/reference/frontmatter/#strict-mode-strict) for complete strict mode documentation.

#### Limit workflow longevity by `stop-after:`

Use `stop-after:` in the `on:` section to limit the time of operation of an agentic workflow. For example, using

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+7d"
```

will mean the agentic workflow no longer operates 7 days after time of compilation.

For complete documentation on `stop-after:` configuration and supported formats, see [Trigger Events](/gh-aw/reference/triggers/#stop-after-configuration-stop-after).

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

### Repository Access Control

Agentic workflows include built-in access control to prevent unauthorized execution:

By default, workflows restrict execution to users with administrative privileges:

- **Default roles**: `admin` and `maintainer` repository permissions are required
- **Automatic enforcement**: Permission checks are automatically added to workflows with potentially unsafe triggers (`push`, `issues`, `pull_request`, etc.)
- **Safe trigger exceptions**: Workflows that only use "safe" triggers (`schedule`, `workflow_run`) skip permission checks by default
- **workflow_dispatch** is treated as safe only when `write` is in the allowed roles, since workflow_dispatch can be triggered by users with write access

Use the `roles:` frontmatter field to customize who can trigger workflows:

```yaml
# Default (recommended for most workflows)
roles: [admin, maintainer]

# Allow contributors with write access (use carefully)
roles: [admin, maintainer, write]

# Disable restrictions entirely (high risk in public repos)
roles: all
```

**workflow_dispatch Examples:**

```yaml
# Permission check REQUIRED - write role not allowed
on:
  workflow_dispatch:
roles: [admin, maintainer]  # Users with write access will be denied

# Permission check SKIPPED - write role allowed
on:
  workflow_dispatch:
roles: [admin, maintainer, write]  # Users with write access allowed

# Permission check SKIPPED - all users allowed
on:
  workflow_dispatch:
roles: all
```

#### Security Behavior

- Permission checks happen at workflow runtime, not when the workflow is installed
- Failed permission checks automatically cancel the workflow with a logged warning
- Users see the workflow start but then immediately stop if they lack permissions
- All permission check results are visible in the Actions tab for debugging

**Important:** Use `roles: all` with extreme caution, especially in public repositories where any authenticated user can potentially trigger workflows through issues, comments, or pull requests.

### Authorization and Token Management

GitHub Agentic Workflows support flexible token configuration for different execution contexts and security requirements.

#### GitHub Token Precedence

By default, workflows use GitHub's standard `GITHUB_TOKEN` for authentication. However, you can override this behavior using environment variables with the following precedence:

1. **`GH_AW_GITHUB_TOKEN`** - Primary override token (highest priority)
2. **`GITHUB_TOKEN`** - Standard GitHub Actions token (fallback)

#### Token Configuration Examples

**Basic override for enhanced permissions:**

```yaml
# Set via repository secrets
env:
  GH_AW_GITHUB_TOKEN: ${{ secrets.CUSTOM_PAT }}
```

**Per-job token configuration:**

```yaml
jobs:
  agentic-task:
    runs-on: ubuntu-latest
    env:
      GH_AW_GITHUB_TOKEN: ${{ secrets.ENHANCED_PAT }}
    steps:
      # Workflow steps use the enhanced token
```

**Safe outputs with custom tokens:**

```yaml
safe-outputs:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
  create-issue:
  create-pull-request:
```

#### Security Considerations

When using custom tokens:

- **Principle of least privilege**: Grant only the minimum permissions required
- **Token rotation**: Regularly rotate Personal Access Tokens
- **Scope limitation**: Use fine-grained PATs when possible to limit repository access
- **Audit logging**: Monitor token usage through GitHub's audit logs
- **Secret management**: Store tokens as GitHub repository or organization secrets, never in code

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
  edit:
  bash: ["echo", "git status"]   # keep tight; avoid wildcards
```

- Patterns to avoid:

```yaml
tools:
  github:
    allowed: ["*"]            # Too broad
  bash: [":*"]           # Unrestricted shell access
```

#### Egress Filtering

A critical guardrail is strict control over outbound network connections. Agentic Workflows now supports declarative network allowlists for containerized MCP servers.

Example (domain allowlist):

```yaml
mcp-servers:
  fetch:
    container: mcp/fetch
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

#### Sanitized Context Text Usage

**CRITICAL**: Always use `${{ needs.activation.outputs.text }}` instead of raw `github.event` fields when accessing user-controlled content.

Raw context fields like `${{ github.event.issue.title }}`, `${{ github.event.issue.body }}`, and `${{ github.event.comment.body }}` contain unsanitized user input that can:
- Inject malicious prompts to manipulate AI behavior
- Trigger unintended @mentions and bot commands  
- Include XML/HTML content that could affect output processing
- Contain excessive content leading to resource exhaustion

The `needs.activation.outputs.text` provides the same content but with security protections:
- @mentions are neutralized: `@user` becomes `` `@user` ``
- Bot triggers are escaped: `fixes #123` becomes `` `fixes #123` ``
- XML tags converted to safe parentheses format
- Only HTTPS URIs from trusted domains allowed
- Content size limited to 0.5MB and 65k lines maximum
- Control characters and ANSI sequences removed

```aw wrap
# SECURE: Use sanitized context
Analyze this content: "${{ needs.activation.outputs.text }}"

# INSECURE: Raw user input (vulnerable to injection)
Title: "${{ github.event.issue.title }}"
Body: "${{ github.event.issue.body }}"
```

#### Policy Enforcement

- **Input sanitization**: Always use sanitized context text for user-controlled content
- **Action validation**: Implement a plan-validate-execute flow where policy layers check each tool call against risk thresholds

### Safe Outputs Security Model

Safe outputs provide a security-first approach to GitHub API interactions by separating AI processing from write operations. The agentic portion of workflows runs with minimal read-only permissions, while separate jobs handle validated GitHub API operations like creating issues, comments, or pull requests.

This architecture ensures the AI never has direct write access to your repository, preventing unauthorized changes while still enabling automated actions. All agent output is automatically sanitized and validated before processing.

See the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete configuration details and available output types.

### Threat Detection

GitHub Agentic Workflows includes automatic threat detection to analyze agent output and code changes for potential security issues before they are applied. When safe outputs are configured, a threat detection job automatically runs to identify prompt injection attempts, secret leaks, and malicious code patches.

The system uses AI-powered analysis with workflow source context to distinguish between legitimate actions and threats, helping reduce false positives while maintaining strong security controls.

See the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for threat detection configuration options.

### Network Isolation

Network isolation in GitHub Agentic Workflows operates at two layers to prevent unauthorized network access:

1. **MCP Tool Network Controls**: Containerized tools with network-level domain allowlisting
2. **AI Engine Network Permissions**: Configurable network access controls for AI engines

See the [Network Reference](/gh-aw/reference/network/) for detailed configuration options and the [Engine Network Permissions](#engine-network-permissions) section below for engine-specific controls.

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

1. **Start with Minimal Access**: Begin with `defaults` and add only needed ecosystems
2. **Use Ecosystem Identifiers**: Prefer `python`, `node`, etc. over listing individual domains
3. **Use Wildcards Carefully**: `*.example.com` matches any subdomain including nested ones (e.g., `api.example.com`, `nested.api.example.com`) - ensure this broad access is intended
4. **Test Thoroughly**: Verify that all required domains/ecosystems are included in allowlist
5. **Monitor Usage**: Review workflow logs to identify any blocked legitimate requests
6. **Document Reasoning**: Comment why specific domains/ecosystems are required for maintenance

### Permission Modes

1. **No network permissions**: Defaults to basic infrastructure only (backwards compatible)
   ```yaml
   engine:
     id: claude
     # No network block - defaults to basic infrastructure
   ```

2. **Basic infrastructure only**: Explicit basic infrastructure access
   ```yaml
   engine:
     id: claude

   network: defaults  # Or use "allowed: [defaults]"
   ```

3. **Ecosystem-based access**: Use ecosystem identifiers for common development tools
   ```yaml
   engine:
     id: claude

   network:
     allowed:
       - defaults         # Basic infrastructure
       - python          # Python/PyPI ecosystem
       - node            # Node.js/NPM ecosystem
       - containers      # Container registries
   ```

4. **Granular domain control**: Specific domains only
   ```yaml
   engine:
     id: claude

   network:
     allowed:
       - "api.github.com"
       - "*.company-internal.com"
   ```

5. **Complete denial**: No network access
   ```yaml
   engine:
     id: claude

   network: {}  # Deny all network access
   ```

## Engine Security Guide

Different agentic engines have distinct defaults and operational surfaces.

#### `engine: claude`

- Restrict `claude.allowed` to only the needed capabilities (Edit/Write/WebFetch/Bash with a short list)
- Keep `allowed_tools` minimal in the compiled step; review `.lock.yml` outputs
- Use engine network permissions with ecosystem identifiers to grant access to only required development tools

#### Security posture differences across engines

Copilot and Claude expose richer default tools and optional Bash; Codex relies more on CLI behaviors. In all cases, tool allow-lists, network restrictions, and pinned dependencies are your primary controls.

## See also

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Network Configuration](/gh-aw/reference/network/)
- [Tools](/gh-aw/reference/tools/)
- [MCPs](/gh-aw/guides/mcps/)
- [Workflow Structure](/gh-aw/reference/workflow-structure/)

## References

- Model Context Protocol: Security Best Practices (2025-06-18) — <https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices>
