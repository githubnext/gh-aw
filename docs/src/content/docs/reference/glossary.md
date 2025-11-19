---
title: Glossary
description: Definitions of technical terms and concepts used throughout GitHub Agentic Workflows documentation.
sidebar:
  order: 1000
---

This glossary provides definitions for key technical terms and concepts used in GitHub Agentic Workflows.

## Core Concepts

### Agentic Workflow
A workflow that uses AI agents to interpret natural language instructions and execute tasks autonomously. Unlike traditional GitHub Actions that follow pre-programmed steps, agentic workflows understand context, make decisions, and adapt to situations.

### Agent
An AI system (such as GitHub Copilot, Claude, or Codex) that executes the natural language instructions in an agentic workflow. The agent interprets the task, uses available tools, and generates appropriate outputs.

### Frontmatter
The YAML configuration section at the beginning of a workflow file, enclosed between `---` markers. The frontmatter defines triggers, permissions, tools, engines, and other workflow settings. Example:

```yaml
---
on: issues
permissions: read-all
tools:
  github:
---
```

### Workflow Lock File (.lock.yml)
The compiled GitHub Actions workflow file generated from a workflow markdown file (`.md`). The `.lock.yml` file contains the complete GitHub Actions YAML with security hardening applied. Both the `.md` source file and the `.lock.yml` compiled file should be committed to version control.

## Tools and Integration

### MCP (Model Context Protocol)
A standardized protocol that allows AI agents to securely connect to external tools, databases, and services. MCP enables workflows to integrate with GitHub APIs, web services, file systems, and custom integrations while maintaining security controls.

### MCP Server
A service that implements the Model Context Protocol to provide specific capabilities to AI agents. Examples include the GitHub MCP server (for GitHub API operations), Playwright MCP server (for browser automation), or custom MCP servers for specialized tools.

### Tools
Capabilities that an AI agent can use during workflow execution. Tools are configured in the frontmatter and include GitHub operations (`github:`), file editing (`edit:`), web access (`web-fetch:`, `web-search:`), shell commands (`bash:`), browser automation (`playwright:`), and custom MCP servers.

## Security and Outputs

### Safe Outputs
A security feature that enables workflows to create GitHub issues, comments, pull requests, and other content without giving the AI agent direct write permissions. The AI generates structured output that is then processed by separate, permission-controlled jobs. Configured using the `safe-outputs:` section in frontmatter.

### Staged Mode
A preview mode where workflows simulate their actions without actually making changes. Useful for testing and validation before running workflows in production. The AI generates output showing what would happen, but no GitHub API write operations are performed.

### Permissions
Access controls that define what operations a workflow can perform. Workflows follow the principle of least privilege, starting with read-only access by default. Write operations are typically handled through safe outputs rather than direct permissions.

## Workflow Components

### Engine
The AI system that powers the agentic workflow. GitHub Agentic Workflows supports multiple engines:
- **GitHub Copilot** (default): Uses GitHub's coding assistant
- **Claude**: Uses Anthropic's Claude AI
- **Codex**: Uses OpenAI's Codex
- **Custom**: Allows defining custom execution steps

### Triggers
Events that cause a workflow to run. Triggers are defined in the `on:` section of frontmatter and can include:
- Issue events (`issues:`)
- Pull request events (`pull_request:`)
- Scheduled runs (`schedule:`)
- Manual runs (`workflow_dispatch:`)
- Comment commands (`command:`)

### Network Permissions
Controls over what external domains and services a workflow can access. Configured using the `network:` section in frontmatter. Options include:
- `defaults`: Access to common development infrastructure
- Custom allow-lists: Specific domains and services
- `{}` (empty): No network access

### Imports
Reusable workflow components that can be shared across multiple workflows. Imports are specified in the `imports:` field and can include tool configurations, common instructions, or security guidelines stored in separate files.

## Development and Compilation

### Compilation
The process of converting a workflow markdown file (`.md`) into a GitHub Actions workflow file (`.lock.yml`). Performed using the `gh aw compile` command. Compilation validates the workflow, resolves imports, configures tools, and applies security hardening.

### CLI (Command Line Interface)
The `gh-aw` extension for the GitHub CLI (`gh`) that provides commands for managing agentic workflows:
- `gh aw compile`: Compile workflows
- `gh aw run`: Trigger workflow runs
- `gh aw status`: Check workflow status
- `gh aw logs`: Download and analyze logs
- `gh aw add`: Add workflows from repositories

### Validation
The process of checking workflow files for errors, security issues, and best practices. Validation occurs during compilation and can be enhanced with strict mode and security scanners (actionlint, zizmor, poutine).

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
