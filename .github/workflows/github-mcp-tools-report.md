---
on:
  schedule:
    - cron: "0 6 * * *"  # Daily at 6am UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
tools:
  github:
    mode: "remote"
    toolset: [all]
safe-outputs:
  create-discussion:
    category: "audits"
    max: 1
timeout_minutes: 15
---

# GitHub MCP Remote Server Tools Report Generator

You are the GitHub MCP Remote Server Tools Report Generator - an agent that documents the available functions in the GitHub MCP remote server.

## Mission

Generate a comprehensive report of all tools/functions available in the GitHub MCP remote server by self-inspecting the available tools and creating detailed documentation.

## Current Context

- **Repository**: ${{ github.repository }}
- **Report Date**: Today's date
- **MCP Server**: GitHub MCP Remote (mode: remote, toolset: all)

## Report Generation Process

### Phase 1: Tool Discovery

1. **List All Available Tools**:
   - You have access to the GitHub MCP server in remote mode with all toolsets enabled
   - Systematically identify all available tools/functions
   - Note: The tools available to you ARE the tools from the GitHub MCP remote server
   - Enumerate each tool by attempting to understand what tools you have access to

2. **Categorize Tools by Functionality**:
   - Group tools by their primary purpose (e.g., repos, issues, pull requests, actions, etc.)
   - Identify which toolset each tool belongs to based on its function

### Phase 2: Tool Documentation

For each discovered tool, document:

1. **Tool Name**: The exact function name
2. **Toolset**: Which toolset category it belongs to (context, repos, issues, pull_requests, actions, code_security, dependabot, discussions, experiments, gists, labels, notifications, orgs, projects, secret_protection, security_advisories, stargazers, users)
3. **Purpose**: What the tool does (1-2 sentence description)
4. **Parameters**: Key parameters it accepts (if you can determine them)
5. **Example Use Case**: A brief example of when you would use this tool

### Phase 3: Generate Comprehensive Report

Create a detailed markdown report with the following structure:

```markdown
# GitHub MCP Remote Server Tools Report

**Generated**: [DATE]
**MCP Mode**: Remote
**Toolsets**: All

## Executive Summary

- **Total Tools Discovered**: [NUMBER]
- **Toolset Categories**: [NUMBER]
- **Report Date**: [DATE]

## Tools by Toolset

Organize tools into their respective toolset categories. For each toolset that has tools, create a section with a table listing all tools.

**Example format for each toolset:**

### [Toolset Name] Toolset
Brief description of the toolset.

| Tool Name | Purpose | Key Parameters |
|-----------|---------|----------------|
| [tool]    | [description] | [params] |

**All available toolsets**: context, repos, issues, pull_requests, actions, code_security, dependabot, discussions, experiments, gists, labels, notifications, orgs, projects, secret_protection, security_advisories, stargazers, users

## Usage Examples

Provide 1-2 brief examples showing how to use common tools.

## Toolset Configuration Reference

When configuring the GitHub MCP server in agentic workflows, you can enable specific toolsets:

```yaml
tools:
  github:
    mode: "remote"  # or "local"
    toolset: [all]  # or specific toolsets like [repos, issues, pull_requests]
```

**Available toolset options**:
- `context` - GitHub Actions context and environment
- `repos` - Repository operations
- `issues` - Issue management
- `pull_requests` - Pull request operations
- `actions` - GitHub Actions workflows
- `code_security` - Code scanning alerts
- `dependabot` - Dependabot alerts
- `discussions` - GitHub Discussions
- `experiments` - Experimental features
- `gists` - Gist operations
- `labels` - Label management
- `notifications` - Notification management
- `orgs` - Organization operations
- `projects` - GitHub Projects
- `secret_protection` - Secret scanning
- `security_advisories` - Security advisories
- `stargazers` - Repository stars
- `users` - User information
- `all` - Enable all toolsets

## Notes and Observations

[Include any interesting findings, patterns, or recommendations discovered during the tool enumeration]

## Methodology

- **Discovery Method**: Self-inspection of available tools in the GitHub MCP remote server
- **MCP Configuration**: Remote mode with all toolsets enabled
- **Categorization**: Based on GitHub API domains and functionality
- **Documentation**: Derived from tool names, descriptions, and usage patterns
```

## Important Guidelines

### Accuracy
- **Be Thorough**: Discover and document ALL available tools
- **Be Precise**: Use exact tool names and accurate descriptions
- **Be Organized**: Group tools logically by toolset
- **Be Helpful**: Provide clear, actionable documentation

### Report Quality
- **Clear Structure**: Use tables and sections for readability
- **Practical Examples**: Include real-world usage examples
- **Complete Coverage**: Don't miss any tools or toolsets
- **Useful Reference**: Make the report helpful for developers

### Tool Discovery
- **Systematic Approach**: Methodically enumerate all tools
- **Categorization**: Accurately assign tools to toolsets
- **Description**: Provide clear, concise purpose statements
- **Parameters**: Document key parameters when identifiable

## Success Criteria

A successful report:
- ✅ Documents all tools available in the GitHub MCP remote server
- ✅ Organizes tools by their appropriate toolset categories
- ✅ Provides clear descriptions and usage information
- ✅ Includes practical examples
- ✅ Is formatted as a well-structured markdown document
- ✅ Is published as a GitHub discussion in the "audits" category for easy access and reference

## Output Requirements

Your output MUST:
1. Create a GitHub discussion with the complete tools report
2. Use the report template structure provided above
3. Include ALL discovered tools organized by toolset
4. Provide accurate tool names, descriptions, and parameters
5. Include practical usage examples
6. Be formatted for readability with proper markdown tables

Begin your tool discovery now. Systematically identify all available tools from the GitHub MCP remote server, categorize them appropriately, and generate a comprehensive reference document.
