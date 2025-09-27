---
title: Security Guide
description: Important security considerations for GitHub Agentic Workflows, including sandboxing, permissions, and best practices for safe agentic automation.
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

#### Job Separation for Security

**CRITICAL SECURITY FEATURE**: Agentic workflows split work across multiple jobs to minimize attack surface and create clear security boundaries.

**How Job Separation Works**:
1. **Main AI Job**: Runs with minimal read-only permissions (`contents: read`, `actions: read`)
2. **Safe-Output Jobs**: Separate jobs with specific write permissions, generated automatically by the compiler
3. **Job Dependencies**: Safe-output jobs only execute after the main AI job completes successfully
4. **Artifact-Based Communication**: AI output is passed via GitHub Actions artifacts, not environment variables or secrets

**Security Benefits**:
- **Permission Isolation**: AI processing never has write permissions to GitHub APIs
- **Audit Trail**: Clear separation between AI reasoning and GitHub API actions
- **Blast Radius Containment**: Compromised AI cannot directly modify repository or create issues
- **Defense in Depth**: Multiple layers of validation before any write operations occur
- **Code Review**: Generated safe-output jobs are visible in `.lock.yml` files for review

**Example Architecture**:
```yaml
# Workflow splits into multiple jobs automatically:

main-job:
  permissions:
    contents: read    # AI can read repository
    actions: read     # AI can access artifacts
  # AI processing happens here with minimal permissions

safe-output-create-issue:  # Generated automatically
  needs: main-job          # Only runs after AI completes
  permissions:
    issues: write          # Specific permission for this action only
  # Creates issues based on AI output artifact

safe-output-add-comment:   # Generated automatically  
  needs: main-job          # Only runs after AI completes
  permissions:
    issues: write          # Specific permission for this action only
  # Adds comments based on AI output artifact
```

#### Permission Configuration

**Recommended**: Use safe-outputs for secure permission separation:

```yaml
# Safe-outputs pattern (recommended)
permissions:
  contents: read      # Minimal permissions for main job
  actions: read

safe-outputs:
  create-issue:       # Automatic issue creation in separate job
  add-comment:        # Automatic comments in separate job
```

**Alternative**: Direct permissions (use only when safe-outputs cannot meet requirements):

```yaml
# Direct permissions pattern (not recommended)
permissions:
  issues: write       # Avoid when possible - use safe-outputs instead
  contents: read
```

#### Network Access Control

**IMPORTANT**: Control AI engine network access using the top-level `network:` field to prevent data exfiltration and limit attack surface.

**Recommended Pattern**:
```yaml
engine:
  id: claude

# Use ecosystem identifiers for development needs
network:
  allowed:
    - defaults        # Basic infrastructure (certificates, package mirrors)
    - python         # Python/PyPI ecosystem access
    - node           # Node.js/NPM ecosystem access
    - github         # GitHub API domains
```

**Restrictive Pattern** (for sensitive environments):
```yaml
engine:
  id: claude

# Deny all network access
network: {}
```

**Network Security Benefits**:
- **Data Exfiltration Prevention**: Blocks unauthorized external connections
- **Supply Chain Protection**: Limits access to trusted package registries only
- **Compliance**: Meets organizational network security requirements
- **Audit Trail**: Network access attempts are logged for monitoring

For complete network configuration options, see [Network Permissions](/gh-aw/reference/network/).

### Human in the Loop

GitHub Actions workflows are designed to be steps within a larger process. Safe-outputs enhance human oversight while maintaining security:

**Built-in Human Review Points**:
- **Safe-Output Visibility**: All generated issues, comments, and PRs are immediately visible to humans
- **Pull Requests Require Humans**: GitHub Actions cannot approve or merge pull requests - human review is always required
- **Workflow History**: All AI actions are logged in GitHub Actions for audit and review
- **Lock File Review**: Generated `.lock.yml` files show exactly what permissions and jobs will execute

**Additional Human Controls**:
- **Approval Gates**: Use manual approval steps for high-risk operations like deployments or secret management
- **Plan-Apply Separation**: Implement a "plan" phase that generates a preview of actions before execution via safe-outputs
- **Review and Audit**: Regularly review workflow history, permissions, and tool usage to ensure compliance with security policies

**Safe-Outputs as Human Interface**:
- **Immediate Visibility**: Issues and comments created by safe-outputs are immediately visible to repository users
- **No Silent Actions**: Unlike direct API calls, safe-outputs always create visible artifacts for human review
- **Granular Control**: Each safe-output type can be configured with limits (max issues, comment targets, etc.)

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

### Repository Access Control

Agentic workflows include built-in access control to prevent unauthorized execution:

By default, workflows restrict execution to users with administrative privileges:

- **Default roles**: `admin` and `maintainer` repository permissions are required
- **Automatic enforcement**: Permission checks are automatically added to workflows with potentially unsafe triggers (`push`, `issues`, `pull_request`, etc.)
- **Safe trigger exceptions**: Workflows that only use "safe" triggers (`workflow_dispatch`, `schedule`, `workflow_run`) skip permission checks by default

Use the `roles:` frontmatter field to customize who can trigger workflows:

```yaml
# Default (recommended for most workflows)
roles: [admin, maintainer]

# Allow contributors with write access (use carefully)
roles: [admin, maintainer, write]

# Disable restrictions entirely (high risk in public repos)
roles: all
```

#### Security Behavior

- Permission checks happen at workflow runtime, not when the workflow is installed
- Failed permission checks automatically cancel the workflow with a logged warning
- Users see the workflow start but then immediately stop if they lack permissions
- All permission check results are visible in the Actions tab for debugging

**Important:** Use `roles: all` with extreme caution, especially in public repositories where any authenticated user can potentially trigger workflows through issues, comments, or pull requests.

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



## Engine Security Guide

Different agentic engines have distinct defaults and operational surfaces. Apply security controls consistently across all engines.

#### Security Checklist for All Engines

**Permission Strategy**:
- ✅ **Use safe-outputs** instead of direct write permissions whenever possible
- ✅ **Minimal main job permissions**: `contents: read`, `actions: read` only
- ✅ **Review generated jobs**: Check `.lock.yml` for all generated safe-output jobs

**Network Controls**:
- ✅ **Configure network permissions**: Use `network:` with ecosystem identifiers
- ✅ **Start restrictive**: Begin with `defaults` and add only needed ecosystems
- ✅ **Document requirements**: Comment why specific network access is needed

**Tool Configuration**:
- ✅ **Explicit tool allowlists**: Never use wildcards or broad permissions
- ✅ **Minimal tool access**: Grant only the specific tools needed for the task
- ✅ **Review compiled tools**: Check `.lock.yml` for actual tool permissions

#### Engine-Specific Considerations

**`engine: claude`**:
- Use network permissions with ecosystem identifiers (`python`, `node`, etc.)
- Restrict tool access to specific Claude capabilities needed
- Monitor network usage through Claude Code hooks

**`engine: codex`** and **`engine: copilot`**:
- Tool allowlists and network restrictions are your primary controls
- Focus on limiting CLI command access and external tool invocations

**All Engines**:
- Safe-outputs provide consistent security regardless of engine choice
- Job separation works identically across all engine types
- Network and tool restrictions should be applied universally

## See also

- [Safe Output Processing](/gh-aw/reference/safe-outputs/) - Complete guide to safe-outputs configuration
- [Network Permissions](/gh-aw/reference/network/) - Detailed network access control options
- [Tools Configuration](/gh-aw/reference/tools/) - Tool allowlist configuration
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol security
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Understanding workflow architecture

## References

- Model Context Protocol: Security Best Practices (2025-06-18) — <https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices>
