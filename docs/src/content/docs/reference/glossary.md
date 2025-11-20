---
title: Glossary
description: Definitions of technical terms and concepts used throughout GitHub Agentic Workflows documentation.
sidebar:
  order: 1000
---

This glossary provides definitions for key technical terms and concepts used in GitHub Agentic Workflows.

## Core Concepts

### Agentic Workflow
An AI-powered workflow that can reason, make decisions, and take autonomous actions, unlike traditional workflows with fixed if/then rules. Agentic workflows interpret natural language instructions, understand context, and adapt their behavior based on the situation they encounter.

**Why it matters:** Instead of writing complex conditional logic, you describe what you want accomplished in plain language, and the AI figures out how to do it.

### Agent
The AI system (typically GitHub Copilot CLI) that executes the natural language instructions in an agentic workflow. The agent interprets tasks, uses available tools, and generates appropriate outputs based on the context it receives.

**Why it matters:** The agent is the "brain" that reads your instructions and decides what actions to take to accomplish your goals.

### Frontmatter
The configuration section at the top of a workflow file, enclosed between `---` markers. Borrowed from blogging and documentation systems, frontmatter contains YAML settings that control when the workflow runs, what permissions it has, and what tools it can use. Example:

```yaml
---
on: issues
permissions: read-all
tools:
  github:
---
```

**Why it matters:** Frontmatter separates the technical configuration (when and how to run) from the natural language instructions (what to do), making workflows easier to read and maintain.

### Compilation
The process of translating human-friendly Markdown workflows (`.md` files) into GitHub Actions YAML format (`.lock.yml` files). During compilation, the workflow is validated, imports are resolved, tools are configured, and security hardening is applied.

**Why it matters:** Compilation ensures your workflow is correct and secure before it runs, catching errors early and applying best practices automatically.

### Workflow Lock File (.lock.yml)
The compiled GitHub Actions workflow file generated from a workflow markdown file (`.md`). The `.lock.yml` file contains the complete GitHub Actions YAML with security hardening applied. Both the `.md` source file and the `.lock.yml` compiled file should be committed to version control.

**Why it matters:** The lock file is what GitHub Actions actually runs, while the `.md` file remains easy to read and edit.

## Tools and Integration

### MCP (Model Context Protocol)
A standardized protocol that allows AI agents to securely connect to external tools, databases, and services. MCP enables workflows to integrate with GitHub APIs, web services, file systems, and custom integrations while maintaining security controls.

### MCP Server
A service that implements the Model Context Protocol to provide specific capabilities to AI agents. Examples include the GitHub MCP server (for GitHub API operations), Playwright MCP server (for browser automation), or custom MCP servers for specialized tools.

### Tools
Capabilities that an AI agent can use during workflow execution. Tools are configured in the frontmatter and include GitHub operations (`github:`), file editing (`edit:`), web access (`web-fetch:`, `web-search:`), shell commands (`bash:`), browser automation (`playwright:`), and custom MCP servers.

## Security and Outputs

### Safe Outputs
Pre-approved actions the AI can take safely without requiring elevated permissions. The AI generates structured output describing what it wants to create (like issues, comments, or pull requests), which is then processed by separate, permission-controlled jobs. Configured using the `safe-outputs:` section in frontmatter.

**Why it matters:** Safe outputs let AI agents create GitHub content without giving them direct write access, reducing security risks while maintaining functionality.

### Staged Mode
A preview mode where workflows simulate their actions without actually making changes. Useful for testing and validation before running workflows in production. The AI generates output showing what would happen, but no GitHub API write operations are performed.

**Why it matters:** Test workflows safely before they affect your real repository data.

### Permissions
Access controls that define what operations a workflow can perform. Workflows follow the principle of least privilege, starting with read-only access by default. Write operations are typically handled through safe outputs rather than direct permissions.

**Why it matters:** Limiting permissions prevents workflows from accidentally or maliciously making unwanted changes to your repository.

## Workflow Components

### Engine
The AI system that powers the agentic workflow. GitHub Agentic Workflows supports multiple engines:
- **GitHub Copilot** (default): Uses GitHub's coding assistant

### Triggers
Events that cause a workflow to run. Triggers are defined in the `on:` section of frontmatter and can include:
- Issue events (`issues:`)
- Pull request events (`pull_request:`)
- Scheduled runs (`schedule:`)
- Manual runs (`workflow_dispatch:`)
- Comment commands (`command:`)

**Why it matters:** Triggers determine when your workflow activates, letting you automate responses to specific repository events.

### Network Permissions
Controls over what external domains and services a workflow can access. Configured using the `network:` section in frontmatter. Options include:
- `defaults`: Access to common development infrastructure
- Custom allow-lists: Specific domains and services
- `{}` (empty): No network access

**Why it matters:** Network permissions control what external websites and APIs the workflow can access, preventing unauthorized data access.

### Imports
Reusable workflow components that can be shared across multiple workflows. Imports are specified in the `imports:` field and can include tool configurations, common instructions, or security guidelines stored in separate files.

**Why it matters:** Imports let you define common patterns once and reuse them across multiple workflows, reducing duplication and ensuring consistency.

## GitHub and Infrastructure Terms

### GitHub Actions
GitHub's built-in automation platform that runs workflows in response to repository events. Agentic workflows compile to GitHub Actions YAML format, leveraging the existing infrastructure for execution, permissions, and secrets management.

**Why it matters:** GitHub Actions provides the underlying execution environment where agentic workflows run.

### YAML
A human-friendly data format used for configuration files. YAML uses indentation and simple syntax to represent structured data. In agentic workflows, YAML appears in the frontmatter section and in the compiled `.lock.yml` files.

**Why it matters:** Understanding basic YAML syntax helps you configure workflow settings in the frontmatter section.

### Personal Access Token (PAT)
A token that authenticates you to GitHub's APIs with specific permissions. Required for GitHub Copilot CLI to access Copilot services on your behalf. Created at github.com/settings/personal-access-tokens.

**Why it matters:** The PAT allows workflows to use AI capabilities while maintaining security through token-based authentication instead of passwords.

## Development and Compilation

### CLI (Command Line Interface)
The `gh-aw` extension for the GitHub CLI (`gh`) that provides commands for managing agentic workflows:
- `gh aw compile`: Compile workflows
- `gh aw run`: Trigger workflow runs
- `gh aw status`: Check workflow status
- `gh aw logs`: Download and analyze logs
- `gh aw add`: Add workflows from repositories

**Why it matters:** The CLI provides the primary interface for creating, testing, and managing agentic workflows from your terminal.

### Validation
The process of checking workflow files for errors, security issues, and best practices. Validation occurs during compilation and can be enhanced with strict mode and security scanners (actionlint, zizmor, poutine).

**Why it matters:** Validation catches problems before workflows run, preventing errors and security issues in production.

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
