# Agentic Workflow Creation Guides

This directory contains focused guides for creating GitHub Agentic Workflows (gh-aw). Each guide covers a specific aspect of workflow creation to keep the main creation prompts concise and maintainable.

## Available Guides

### [Writing Style Guidelines](writing-style.md)
Style guidelines for creating effective agentic workflow prompts and user interactions.

**Topics covered**:
- Interactive mode communication style (Copilot CLI style, emoji usage)
- Workflow prompt writing best practices
- Frontmatter configuration minimalism
- Documentation comments

**Use when**: You need guidance on tone, style, or formatting for prompts and user interactions.

### [Tool Configuration Reference](tool-configuration.md)
Comprehensive reference for configuring tools in agentic workflows.

**Topics covered**:
- Built-in tools (GitHub, Serena, Playwright, Web Fetch, Bash, Edit)
- Tool categories by use case
- Safe outputs for write operations
- Network configuration
- Tool installation patterns

**Use when**: You need to configure tools, understand tool options, or set up safe outputs.

### [MCP Server Usage Patterns](mcp-usage.md)
Best practices for configuring and using Model Context Protocol (MCP) servers.

**Topics covered**:
- When to use MCP servers
- MCP server configuration
- Inspecting MCP servers with `gh aw mcp`
- Common patterns (API integration, database query, file system)
- Troubleshooting and debugging

**Use when**: You need to integrate custom MCP servers or understand MCP server patterns.

### [Data Computation Best Practices](data-computation.md)
Patterns for processing, transforming, and computing data within workflows.

**Topics covered**:
- Using bash tools (jq, git, grep, etc.)
- Leveraging language servers for code analysis
- GitHub API for repository data
- Common patterns (aggregation, filtering, transformation)
- Data caching and performance considerations

**Use when**: You need to process data, transform JSON, analyze repository information, or compute statistics.

### [Pre-Download and Installation Strategies](pre-download.md)
Strategies for pre-downloading dependencies and installing tools before the agent executes.

**Topics covered**:
- Steps vs agent execution
- Common installation patterns (Playwright, FFmpeg, code analysis tools)
- Pre-download strategies (data files, git repos, Docker images)
- Performance optimization
- Error handling

**Use when**: You need to install tools, pre-download data, or set up the environment before the agent runs.

## Usage

These guides are referenced from the main creation prompts:
- `.github/aw/create-agentic-workflow.md` - Main workflow creation prompt
- `.github/aw/update-agentic-workflow.md` - Workflow update prompt
- `.github/aw/debug-agentic-workflow.md` - Workflow debugging prompt

### How to Use These Guides

1. **During workflow creation**: The main prompts reference these guides for specific topics
2. **For deep dives**: Read a guide directly when you need detailed information on a specific topic
3. **As a reference**: Use guides as quick reference for common patterns and configurations

### Updating Guides

When updating these guides:
1. Keep content focused on the guide's specific topic
2. Maintain cross-references between related guides
3. Update main prompts if guide structure changes
4. Ensure examples are tested and accurate
5. Follow the writing style guidelines from `writing-style.md`

## Quick Reference

| Guide | Primary Topics | Best For |
|-------|---------------|----------|
| **writing-style.md** | Style, tone, minimalism | Prompt writing, user interaction |
| **tool-configuration.md** | Tools, safe outputs, network | Tool setup, configuration |
| **mcp-usage.md** | MCP servers, patterns | Custom integrations, MCP |
| **data-computation.md** | Data processing, bash tools | Data analysis, transformation |
| **pre-download.md** | Installation, dependencies | Environment setup, tooling |

## Additional Resources

- **Complete Documentation**: `.github/aw/github-agentic-workflows.md`
- **Debugging Workflows**: `.github/aw/debug-agentic-workflow.md`
- **Workflow Health**: `.github/aw/runbooks/workflow-health.md`
- **Main Repository**: https://github.com/githubnext/gh-aw
