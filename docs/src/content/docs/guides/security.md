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

Thorough review is essential when working with agentic workflows. Review workflow contents before installation, treating prompt templates and rule files as code. Assess compiled `.lock.yml` files to understand actual permissions and operations. GitHub Actions' built-in protections (read-only defaults for fork PRs, restricted secret access) apply to agentic workflows. See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use) and [permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions). When specifying permissions explicitly, all unspecified permissions default to `none`. By default, workflows restrict execution to users with `admin`, `maintainer`, or `write` permissions—use `roles: all` carefully in public repositories.

## Threat Model

Understanding the security risks in agentic workflows helps inform protective measures:

### Primary Threats

**Command execution**: Workflows run in GitHub Actions' partially-sandboxed environment. By default, arbitrary shell commands are disallowed, but specific commands can be manually allowlisted. If misconfigured on sensitive code, attackers might fetch and run malicious code to exfiltrate data or perform unauthorized execution.

**Malicious inputs**: Attackers can craft inputs that poison AI agents. Workflows pull data from Issues, PRs, comments, and code. Untrusted inputs (e.g., in open source settings) can carry hidden AI payloads. Workflows minimize risk by restricting expressions in markdown content, requiring GitHub MCP access, though returned data can still manipulate AI behavior if not sanitized.

**Tool exposure**: By default, workflows only access GitHub MCP in read-only mode. Unconstrained 3rd-party MCP tools enable data exfiltration or privilege escalation.

**Supply chain**: Unpinned Actions, npm packages, and container images are vulnerable to tampering—standard GitHub Actions threats apply.

### Core Security Principles

Agentic Workflows are GitHub Actions workflows and inherit their security model: isolated repository copies, read-only defaults for forked PRs, restricted secret access, and explicit permissions (default `none`). See [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use).

Additional compilation-time security measures include expression restrictions (limited frontmatter expressions), highly restricted commands (explicit allowlisting required), tool allowlisting, engine network restrictions (domain allowlists), workflow longevity limits, and chat iteration limits.

Apply consistently: least privilege by default (elevate permissions only when required), default-deny approach (explicit tool allowlisting), separation of concerns (plan/apply phases with approval gates), and supply chain integrity (pin dependencies to immutable SHAs).

## Implementation Guidelines

### Workflow Permissions and Triggers

Configure GitHub Actions with defense in depth:

#### Permission Configuration

Set minimal read-only permissions for the agentic processing. Use `safe-outputs` for write operations:

```yaml wrap
# Applies to the agentic processing (read-only)
permissions:
  contents: read
  actions: read

# Use safe-outputs for write operations
safe-outputs:
  create-issue:
  add-comment:
```

#### Fork Protection for Pull Request Triggers

Pull request workflows block forks by default for security. Workflows triggered by `pull_request` events only execute for pull requests from the same repository unless explicitly configured to allow forks.

**Default behavior (blocks all forks):**
```yaml wrap
on:
  pull_request:
    types: [opened, synchronize]
# Blocks all forked PRs, only allows same-repo PRs
```

**Allow specific fork patterns:**
```yaml wrap
on:
  pull_request:
    types: [opened, synchronize]
    forks: ["trusted-org/*"]  # Allow forks from specific org
```

**Allow all forks (use with caution):**
```yaml wrap
on:
  pull_request:
    types: [opened, synchronize]
    forks: ["*"]  # Allow all forks
```

The compiler generates conditions using repository ID comparison (`github.event.pull_request.head.repo.id == github.repository_id`) for reliable fork detection that is not affected by repository renames.

#### workflow_run Trigger Security

Workflows triggered by `workflow_run` events include automatic protections against cross-repository attacks:

**Automatic repository validation:**

The compiler automatically injects a repository ID check into the activation job for all workflows using `workflow_run` triggers:

```yaml wrap
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
```

This generates a safety condition that prevents execution if the triggering workflow_run is from a different repository:

```yaml wrap
if: >
  (user_condition) &&
  ((github.event_name != 'workflow_run') ||
   (github.event.workflow_run.repository.id == github.repository_id))
```

The safety check combines with user-specified conditions using AND logic and protects all downstream jobs through job dependencies.

**Branch restriction validation:**

Workflows with `workflow_run` triggers should include branch restrictions to prevent execution for workflow runs on all branches:

```yaml wrap
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches:
      - main
      - develop
```

Without branch restrictions, workflows emit warnings during compilation (or errors in strict mode). Branch restrictions improve security by limiting which branch workflows can trigger the workflow_run event.

**Production workflows**: Consider using strict mode to enforce additional security constraints:

```yaml wrap
# Enable strict mode for production workflows
strict: true
permissions:
  contents: read  # Write permissions are blocked in strict mode
timeout-minutes: 10
network:
  allowed:
    - "api.example.com"
```

Strict mode prevents write permissions (`contents:write`, `issues:write`, `pull-requests:write`) and requires explicit network configuration. Use `safe-outputs` configuration instead for controlled GitHub API interactions. See [Strict Mode Validation](#strict-mode-validation) for details.

### Human in the Loop

Critical operations require human review. Use `manual-approval` to require approval before execution (configure environment protection rules in repository settings). See [Manual Approval Gates](/gh-aw/reference/triggers/#manual-approval-gates-manual-approval). GitHub Actions cannot approve or merge PRs, ensuring human involvement. Implement plan-apply separation (preview via output issue/PR). Regularly audit workflow history, permissions, and tool usage.

### Limit operations

#### Strict Mode Validation

Enable strict mode for production workflows to enforce enhanced security constraints:

```yaml wrap
strict: true  # In frontmatter
permissions:
  contents: read
network:
  allowed:
    - "api.example.com"
```

Or via CLI: `gh aw compile --strict`

**Enforces**: write permissions blocked (use `safe-outputs`), explicit network configuration required, no network wildcards, MCP network configuration required.

**Benefits**: Minimizes attack surface, ensures compliance, improves auditability. CLI flag takes precedence over frontmatter. See [Frontmatter Reference](/gh-aw/reference/frontmatter/#strict-mode-strict).

#### Limit workflow longevity by `stop-after:`

Use `stop-after:` in the `on:` section to limit the time of operation of an agentic workflow. For example, using

```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+7d"
```

will mean the agentic workflow no longer operates 7 days after time of compilation.

For complete documentation on `stop-after:` configuration and supported formats, see [Trigger Events](/gh-aw/reference/triggers/#stop-after-configuration-stop-after).

#### Limit workflow runs by engine `max-turns:`

Use `max-turns:` in the engine configuration to limit the number of chat iterations per run. This prevents runaway loops and excessive resource consumption. For example:

```yaml wrap
engine:
  id: claude
  max-turns: 5
```

This limits the workflow to a maximum of 5 interactions with the AI engine per run.

#### Monitor costs by `gh aw logs`

Use `gh aw logs` to monitor the costs of running agentic workflows. This command provides insights into the number of turns, tokens used, and other metrics that can help you understand the cost implications of your workflows. Reported information may differ based on the AI engine used (e.g., Claude vs. Codex).

### Repository Access Control

By default, workflows restrict execution to users with `admin`, `maintainer`, or `write` permissions. Permission checks auto-apply to potentially unsafe triggers (`push`, `issues`, `pull_request`). Safe triggers (`schedule`, `workflow_run`) skip checks. `workflow_dispatch` is safe only when `write` is allowed.

Customize via `roles:` frontmatter:

```yaml wrap
roles: [admin, maintainer, write]  # Default
roles: [admin, maintainer]         # Restrictive (recommended for sensitive ops)
roles: [write]                     # Write access only
roles: all                         # All users (high risk in public repos)
```

Permission checks occur at runtime, not installation. Failed checks auto-cancel with logged warnings. Use `roles: all` with extreme caution in public repositories.

### Authorization and Token Management

Token precedence (highest to lowest): individual safe-output `github-token` → safe-outputs global `github-token` → top-level `github-token` → default fallback (`${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`).

:::tip[Automatic Secret Validation]
`github-token` fields are validated during compilation to ensure use of GitHub Actions secret expressions. Plaintext tokens or environment variables cause compilation failure.
:::

#### Token Configuration Examples

```yaml wrap
# Top-level token
github-token: ${{ secrets.CUSTOM_PAT }}

# Per safe-output tokens
safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUT_PAT }}
  create-issue:
    github-token: ${{ secrets.ISSUE_SPECIFIC_PAT }}
```

**Security**: Use least privilege, rotate PATs regularly, use fine-grained PATs, monitor via audit logs, store as secrets (never in code).

### MCP Tool Hardening

Run MCP servers in sandboxes: container isolation (no shared state), non-root UIDs, drop capabilities, apply seccomp/AppArmor, disable privilege escalation. Pin images (digest/SHAs), scan vulnerabilities, track SBOMs.

```yaml wrap
tools:
  web:
    mcp:
      container: "ghcr.io/example/web-mcp@sha256:abc123..."
    allowed: [fetch]
```

#### Tool Allow/Disallow

Configure explicit allow-lists. See `docs/tools.md` for full options.

```yaml wrap
# Minimal GitHub tools (recommended)
tools:
  github:
    allowed: [get_issue, add_issue_comment]

# Restricted bash (avoid wildcards)
engine: claude
tools:
  edit:
  bash: ["echo", "git status"]

# Avoid: ["*"] or [":*"] (too broad)
```

#### Egress Filtering

Declarative network allowlists for containerized MCP servers:

```yaml wrap
mcp-servers:
  fetch:
    container: mcp/fetch
    network:
      allowed: ["example.com"]
    allowed: ["fetch"]
```

Compiler generates per-tool Squid proxy; MCP egress forced through proxy via iptables. Only listed domains reachable. Applies to `mcp.container` stdio servers only. Use bare domains, minimal allowlists; review `.lock.yml`.

### Agent Security and Prompt Injection Defense

Protect against model manipulation through layered defenses:

#### Sanitized Context Text Usage

**CRITICAL**: Always use `${{ needs.activation.outputs.text }}` instead of raw `github.event` fields. Raw fields enable prompt injection, @mentions, bot triggers, XML/HTML injection, and resource exhaustion.

Sanitized output provides: neutralized @mentions/bot triggers, safe XML format, only HTTPS URIs from trusted domains, 0.5MB/65k line limits, removed control characters.

```aw wrap
# SECURE
Analyze: "${{ needs.activation.outputs.text }}"

# INSECURE (vulnerable)
Title: "${{ github.event.issue.title }}"
```

Implement plan-validate-execute flow with policy checks against risk thresholds.

### Safe Outputs Security Model

Safe outputs provide a security-first approach to GitHub API interactions by separating AI processing from write operations. The agentic portion of workflows runs with minimal read-only permissions, while separate jobs handle validated GitHub API operations like creating issues, comments, or pull requests.

This architecture ensures the AI never has direct write access to your repository, preventing unauthorized changes while still enabling automated actions. All agent output is automatically sanitized and validated before processing.

See the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete configuration details and available output types.

### Threat Detection

Automatic threat detection analyzes agent output and code changes for prompt injection, secret leaks, and malicious patches. Auto-enabled with safe outputs; uses AI-powered analysis with workflow context to reduce false positives.

```yaml wrap
safe-outputs:
  create-pull-request:
  threat-detection:
    enabled: true                    # Default
    prompt: "Focus on SQL injection" # Optional
    engine:                          # Optional custom engine
      id: claude
    steps:                           # Optional additional scanning
      - name: Run TruffleHog
        uses: trufflesecurity/trufflehog@main
```

Add specialized scanners (Ollama/LlamaGuard, Semgrep, TruffleHog) for defense-in-depth. See [Threat Detection Guide](/gh-aw/guides/threat-detection/).

### Automated Security Scanning

[zizmor](https://github.com/zizmorcore/zizmor) scans compiled workflows during compilation:

```bash wrap
gh aw compile --zizmor              # Scan with warnings
gh aw compile --strict --zizmor     # Block on findings
```

Analyzes `.lock.yml` files for excessive permissions, insecure practices, supply chain vulnerabilities, and misconfigurations. Reports include severity, location, context, and description in IDE-parseable format. Requires Docker. Best practices: run during development, use `--strict --zizmor` in CI/CD, address High/Critical findings.

### Network Isolation

Network isolation in GitHub Agentic Workflows operates at two layers to prevent unauthorized network access:

1. **MCP Tool Network Controls**: Containerized tools with network-level domain allowlisting
2. **AI Engine Network Permissions**: Configurable network access controls for AI engines

See the [Network Reference](/gh-aw/reference/network/) for detailed configuration options and the [Engine Network Permissions](#engine-network-permissions) section below for engine-specific controls.

## Engine Network Permissions

Fine-grained control over AI engine network access, separate from MCP tool permissions. Provides defense in depth, compliance, audit trails, and least privilege.

**Claude Engine**: Hook-based enforcement (PreToolUse hooks), runtime validation, clear error messages, ~10ms overhead.

**Copilot Engine with AWF**: Uses [AWF](https://github.com/githubnext/gh-aw-firewall) firewall wrapper, process-level domain allowlisting, execution wrapping, activity logging. See [Copilot Engine - Network Permissions](/gh-aw/reference/engines/#network-permissions).

**Best Practices**: Start with `defaults`, add needed ecosystems; use ecosystem identifiers over individual domains; use wildcards carefully (matches all subdomains); test thoroughly, monitor logs, document reasoning.

### Permission Modes

```yaml wrap
# No network (defaults to basic infrastructure)
engine:
  id: claude

# Basic infrastructure only
network: defaults

# Ecosystem-based
network:
  allowed:
    - defaults
    - python
    - node
    - containers

# Granular domains
network:
  allowed:
    - "api.github.com"
    - "*.company-internal.com"

# Complete denial
network: {}
```

## Engine Security Guide

**Claude**: Restrict `claude.allowed` to needed capabilities, keep `allowed_tools` minimal, use network permissions with ecosystem identifiers.

**Security posture**: Copilot/Claude expose richer default tools and optional Bash; Codex relies on CLI behaviors. Primary controls: tool allow-lists, network restrictions, pinned dependencies.

## See also

- [Threat Detection Guide](/gh-aw/guides/threat-detection/) - Comprehensive threat detection configuration and examples
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Network Configuration](/gh-aw/reference/network/)
- [Tools](/gh-aw/reference/tools/)
- [MCPs](/gh-aw/guides/mcps/)
- [Workflow Structure](/gh-aw/reference/workflow-structure/)

## References

- Model Context Protocol: Security Best Practices (2025-06-18) — <https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices>
