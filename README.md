# GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them safely in GitHub Actions. From [GitHub Next](https://githubnext.com/) and [Microsoft Research](https://www.microsoft.com/en-us/research/group/research-software-engineering-rise/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

<!--
> [!NOTE]
> **For AI Agents**: To learn about GitHub Agentic Workflows syntax, file formats, tools, and best practices, please read the comprehensive instructions at: [.github/aw/github-agentic-workflows.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/aw/github-agentic-workflows.md)
>
> **Custom Agent**: Use the custom agent at `.github/agents/create-agentic-workflow.md` to interactively create agentic workflows. The custom agent is available at: [.github/agents/create-agentic-workflow.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/agents/create-agentic-workflow.md)
-->

## Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/) to install the extension, add a sample workflow, and see it in action.

## Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://githubnext.github.io/gh-aw/introduction/how-it-works/).

## How It Works

GitHub Agentic Workflows transforms natural language markdown files into GitHub Actions that are executed by AI agents. Here's an example:

```markdown
---
on: daily
permissions: read
safe-outputs:
  create-discussion:
---

# Daily Issues Report

Analyze repository issues and create a daily discussion 
with metrics, trends, and key insights.
```

The `gh aw` cli converts this into a GitHub Actions Workflow (.yml) that runs an AI agent (Copilot, Claude, Codex, ...) in a containerized environment on a schedule or manually.

The AI agent reads your repository context, analyzes issues, generates visualizations, and creates reports - all defined in natural language rather than complex code.

## Safe Agentic Workflows

Security is foundational to GitHub Agentic Workflows. The system implements multiple layers of protection to ensure AI agents operate safely within controlled boundaries:

### Core Security Architecture

**Least Privilege by Default**: Workflows run with minimal read-only permissions. The agentic portion never has direct write access to your repository, preventing unauthorized changes while enabling automation.

**Safe Outputs Separation**: Write operations are strictly separated from AI processing through the `safe-outputs` system. Agents request actions via structured output, and separate permission-controlled jobs execute validated operations—creating issues, PRs, comments, or discussions only after sanitization and validation.

**Sandboxed Execution**: AI agents run in GitHub Actions' isolated environment with containerized tools. Each workflow executes in a fresh, ephemeral environment with no persistent state between runs.

### Input & Output Protection

**Sanitized Context Processing**: User input from issues, PRs, and comments is automatically sanitized before reaching AI agents—neutralizing @mentions, filtering URIs to trusted domains, enforcing content limits (0.5MB/65k lines), and removing control characters that could enable prompt injection attacks.

**Threat Detection**: Automated AI-powered analysis scans agent output for prompt injection attempts, secret leaks, and malicious patches before any actions are taken, reducing false positives through intelligent pattern recognition.

**Template Injection Prevention**: Expressions in GitHub Actions are carefully controlled—untrusted data never flows directly into `${{ }}` expressions, preventing code injection and secret access.

### Network & Tool Isolation

**Network Egress Control**: Containerized MCP tools use domain allowlisting with per-tool Squid proxies. AI engines have configurable network access with ecosystem-based or granular domain controls, enforcing network policies at both tool and engine layers.

**Tool Allowlisting**: Explicit allow-lists control which tools and commands agents can use. Wildcard access is refused by default, requiring specific tool permissions for bash commands, GitHub operations, and MCP servers.

**MCP Server Hardening**: Model Context Protocol servers run in sandboxed containers with non-root UIDs, dropped capabilities, seccomp/AppArmor profiles, and no privilege escalation. Images are pinned to cryptographic digests and scanned for vulnerabilities.

### Supply Chain Security

**Immutable Dependencies**: All GitHub Actions are pinned to commit SHAs rather than mutable tags or branches, preventing supply chain attacks via tag manipulation or repository compromise.

**Container Image Verification**: Docker images used by MCP servers and tools are pinned to SHA256 digests, ensuring reproducible builds and preventing image tampering.

### Access Control & Governance

**Role-Based Execution**: Workflows restrict execution to users with `admin`, `maintainer`, or `write` permissions by default. Access controls auto-apply to unsafe triggers and can be customized per workflow.

**Fork Protection**: Pull request workflows block forks by default, using repository ID comparison for reliable fork detection unaffected by renames. Trusted fork patterns can be explicitly allowed.

**Strict Mode**: Production workflows can enable strict mode to enforce security policies—blocking write permissions, requiring explicit network configuration, refusing wildcard access, enforcing Action SHA pinning, and validating all configurations at compile time.

### Human Oversight & Transparency

**Manual Approval Gates**: Critical operations can require human review before execution through environment protection rules, ensuring AI agents never operate fully autonomously for sensitive actions.

**Audit Trails**: All workflow executions are logged by GitHub Actions with complete visibility into what ran, when, who triggered it, and what actions were taken. Workflow history provides transparency and accountability.

**Plan-Apply Separation**: Workflows can preview changes via output issues or PRs before applying them, enabling human review of AI-generated modifications before they affect the repository.

### Compilation-Time Security

**Expression Restrictions**: Frontmatter expressions are limited and validated to prevent dangerous constructs from reaching the compiled workflow.

**Validation & Scanning**: Workflows are validated at compile time with automatic security scanning via zizmor, actionlint, and poutine—catching excessive permissions, insecure practices, supply chain vulnerabilities, and misconfigurations before deployment.

**Lock File Review**: Compiled `.lock.yml` files are human-readable GitHub Actions YAML with all security hardening applied, enabling teams to review exactly what will execute before merging.

### Workflow Lifecycle Controls

**Expiration & Limits**: Workflows can set expiration dates, iteration limits, and timeout constraints to prevent runaway automation or resource exhaustion.

**Cost Monitoring**: Built-in tools track workflow costs—turns, tokens, and resource usage—helping teams maintain control over AI automation expenses.

GitHub Agentic Workflows inherits GitHub Actions' proven security model—isolated repository copies, restricted secret access, explicit permissions (default `none`)—and extends it with agentic-specific protections. Defense-in-depth is applied consistently: least privilege, default-deny, separation of concerns, and supply chain integrity.

**Learn more**: See the [Security Guide](https://githubnext.github.io/gh-aw/guides/security/) for comprehensive security documentation, threat modeling, implementation guidelines, and best practices.

## Documentation

For complete documentation, examples, and guides, see the [Documentation](https://githubnext.github.io/gh-aw/).

## Contributing

We welcome contributions to GitHub Agentic Workflows! Here's how you can help:

- **Report bugs and request features** by filing issues in this repository
- **Improve documentation** by contributing to our docs
- **Contribute code** by following our [Development Guide](DEVGUIDE.md)
- **Share ideas** in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Share Feedback

We welcome your feedback on GitHub Agentic Workflows! Please file bugs and feature requests as issues in this repository,
and share your thoughts in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord).

## Labs

See the [Labs](https://githubnext.github.io/gh-aw/labs/) page for experimental agentic workflows used by the team to learn, build, and use agentic workflows.
