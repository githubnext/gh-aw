---
title: Glossary
description: Definitions of technical terms and concepts used throughout GitHub Agentic Workflows documentation.
sidebar:
  order: 1000
---

This glossary provides definitions for key technical terms and concepts used in GitHub Agentic Workflows.

## Core Concepts

### Agentic Workflow
An AI-powered workflow that can reason, make decisions, and take autonomous actions using natural language instructions. Unlike traditional workflows with fixed if/then rules, agentic workflows interpret context and adapt their behavior based on the situation they encounter.

### Agent
The AI system (typically GitHub Copilot CLI) that executes natural language instructions in an agentic workflow. The agent interprets tasks, uses available tools, and generates outputs based on context.

### Frontmatter
The configuration section at the top of a workflow file, enclosed between `---` markers. Contains YAML settings that control when the workflow runs, what permissions it has, and what tools it can use. Separates technical configuration from natural language instructions.

```yaml
---
on: issues
permissions: read-all
tools:
  github:
---
```

### Compilation
The process of translating Markdown workflows (`.md` files) into GitHub Actions YAML format (`.lock.yml` files). During compilation, workflows are validated, imports are resolved, tools are configured, and security hardening is applied.

### Workflow Lock File (.lock.yml)
The compiled GitHub Actions workflow file generated from a workflow markdown file (`.md`). Contains complete GitHub Actions YAML with security hardening applied. Both the `.md` source file and `.lock.yml` compiled file should be committed to version control. GitHub Actions runs the lock file, while the `.md` file remains easy to read and edit.

## Tools and Integration

### MCP (Model Context Protocol)
A standardized protocol that allows AI agents to securely connect to external tools, databases, and services. MCP enables workflows to integrate with GitHub APIs, web services, file systems, and custom integrations while maintaining security controls.

### MCP Server
A service that implements the Model Context Protocol to provide specific capabilities to AI agents. Examples include the GitHub MCP server (for GitHub API operations), Playwright MCP server (for browser automation), or custom MCP servers for specialized tools.

### Tools
Capabilities that an AI agent can use during workflow execution. Tools are configured in the frontmatter and include GitHub operations (`github:`), file editing (`edit:`), web access (`web-fetch:`, `web-search:`), shell commands (`bash:`), browser automation (`playwright:`), and custom MCP servers.

## Security and Outputs

### Safe Outputs
Pre-approved actions the AI can take without requiring elevated permissions. The AI generates structured output describing what it wants to create (issues, comments, pull requests), which is processed by separate, permission-controlled jobs. Configured using the `safe-outputs:` section in frontmatter. This approach lets AI agents create GitHub content without direct write access, reducing security risks.

### Staged Mode
A preview mode where workflows simulate their actions without making changes. The AI generates output showing what would happen, but no GitHub API write operations are performed. Use for testing and validation before running workflows in production.

### Permissions
Access controls that define what operations a workflow can perform. Workflows follow the principle of least privilege, starting with read-only access by default. Write operations are typically handled through safe outputs rather than direct permissions.

## Workflow Components

### Engine
The AI system that powers the agentic workflow. GitHub Agentic Workflows supports multiple engines:
- **GitHub Copilot** (default): Uses GitHub's coding assistant

### Triggers
Events that cause a workflow to run. Defined in the `on:` section of frontmatter. Includes issue events (`issues:`), pull request events (`pull_request:`), scheduled runs (`schedule:`), manual runs (`workflow_dispatch:`), and comment commands (`command:`).

### Cron Schedule
A time-based trigger format using standard cron syntax with five fields: minute, hour, day of month, month, and day of week.

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM UTC
```

### workflow_dispatch
A manual trigger that runs a workflow on demand from the GitHub Actions UI or via the GitHub API. Requires explicit user initiation.

```yaml
on: workflow_dispatch
```

### Network Permissions
Controls over what external domains and services a workflow can access. Configured using the `network:` section in frontmatter. Options: `defaults` (common development infrastructure), custom allow-lists (specific domains), or `{}` (no network access).

### Imports
Reusable workflow components that can be shared across multiple workflows. Specified in the `imports:` field. Can include tool configurations, common instructions, or security guidelines stored in separate files.

## GitHub and Infrastructure Terms

### GitHub Actions
GitHub's built-in automation platform that runs workflows in response to repository events. Agentic workflows compile to GitHub Actions YAML format, leveraging the existing infrastructure for execution, permissions, and secrets management.

### GitHub Actions Secret
A secure, encrypted variable stored in repository or organization settings. Holds sensitive values like API keys or tokens. Access in workflows using `${{ secrets.SECRET_NAME }}` syntax.

### YAML
A human-friendly data format used for configuration files. Uses indentation and simple syntax to represent structured data. In agentic workflows, YAML appears in the frontmatter section and in compiled `.lock.yml` files.

### Personal Access Token (PAT)
A token that authenticates you to GitHub's APIs with specific permissions. Required for GitHub Copilot CLI to access Copilot services. Created at github.com/settings/personal-access-tokens.

### Fine-grained Personal Access Token
A type of GitHub Personal Access Token with granular permission control. Specify exactly which repositories the token can access and what permissions it has (`contents: read`, `issues: write`, etc.). Created at github.com/settings/personal-access-tokens.

## Development and Compilation

### CLI (Command Line Interface)
The `gh-aw` extension for the GitHub CLI (`gh`) that provides commands for managing agentic workflows: `gh aw compile` (compile workflows), `gh aw run` (trigger runs), `gh aw status` (check status), `gh aw logs` (download and analyze logs), and `gh aw add` (add workflows from repositories).

### Validation
The process of checking workflow files for errors, security issues, and best practices. Occurs during compilation and can be enhanced with strict mode and security scanners (actionlint, zizmor, poutine).

## Advanced Features

### Cache Memory
Persistent storage for workflows that preserves data between runs. Configured using `cache-memory:` in the tools section, it enables workflows to remember information and build on previous interactions.

### Command Triggers
Special triggers that respond to slash commands in issue and PR comments (e.g., `/review`, `/deploy`). Configured using the `command:` section with a command name.

### Concurrency Control
Settings that limit how many instances of a workflow can run simultaneously. Configured using the `concurrency:` field to prevent resource conflicts or rate limiting.

### Custom Agents
Specialized instructions or configurations that customize AI agent behavior for specific tasks or repositories. Stored in `.github/agents/` or `.github/copilot/instructions/` directories.

### Strict Mode
An enhanced validation mode that enforces additional security checks and best practices. Enabled using `strict: true` in frontmatter or the `--strict` flag when compiling.

## Related Resources

For detailed documentation on specific topics, see:
- [Frontmatter Reference](/gh-aw/reference/frontmatter/)
- [Tools Reference](/gh-aw/reference/tools/)
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Using MCPs Guide](/gh-aw/guides/mcps/)
- [Security Guide](/gh-aw/guides/security/)
- [AI Engines Reference](/gh-aw/reference/engines/)
