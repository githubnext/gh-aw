---
# Sentry MCP Server
# Stdio MCP server for application monitoring and debugging
#
# Provides tools for managing issues, projects, and traces in Sentry
# Documentation: https://mcp.sentry.dev
#
# Available tools:
#   - whoami: Get current user information
#   - find_organizations: List Sentry organizations
#   - find_teams: Find teams in an organization
#   - find_projects: Find projects in an organization
#   - find_releases: Find release information
#   - get_issue_details: Get detailed information about a specific issue
#   - get_trace_details: Get distributed trace details
#   - get_event_attachment: Retrieve event attachments
#   - update_issue: Update issue status, assignee, or other details
#   - search_events: AI-powered search for events (requires OpenAI API key)
#   - search_issues: AI-powered search for issues (requires OpenAI API key)
#   - create_team: Create a new team
#   - create_project: Create a new project
#   - update_project: Update project settings
#   - create_dsn: Create a Data Source Name (DSN)
#   - find_dsns: Find DSNs for a project
#   - analyze_issue_with_seer: AI-powered issue analysis
#   - search_docs: Search Sentry documentation
#   - get_doc: Get specific Sentry documentation
#
# Authentication:
#   Requires SENTRY_ACCESS_TOKEN secret with appropriate scopes
#
# Note: AI-powered search tools (search_events, search_issues) require
#       an OpenAI API key to be configured in your Sentry account
#
# Usage:
#   imports:
#     - shared/mcp/sentry.md

mcp-servers:
  sentry:
    command: "npx"
    args: ["@sentry/mcp-server@latest", "--access-token=${{ secrets.SENTRY_ACCESS_TOKEN }}"]
    allowed:
      - whoami
      - find_organizations
      - find_teams
      - find_projects
      - find_releases
      - get_issue_details
      - get_trace_details
      - get_event_attachment
      - update_issue
      - search_events
      - search_issues
      - create_team
      - create_project
      - update_project
      - create_dsn
      - find_dsns
      - analyze_issue_with_seer
      - search_docs
      - get_doc
---

<!--

## Sentry Integration

This shared configuration provides Sentry MCP server integration for application monitoring and debugging workflows.

### Available Tools

**User & Organization Management:**
- `whoami`: Get current authenticated user information
- `find_organizations`: List all accessible Sentry organizations
- `find_teams`: Find teams within an organization
- `find_projects`: Find projects within an organization

**Release Management:**
- `find_releases`: Search for releases across projects

**Issue & Event Management:**
- `get_issue_details`: Get detailed information about a specific issue including stack traces, breadcrumbs, and context
- `get_trace_details`: Get distributed trace details for performance monitoring
- `get_event_attachment`: Retrieve attachments associated with events
- `update_issue`: Update issue properties (status, assignee, tags, etc.)

**AI-Powered Search:**
- `search_events`: Natural language search for events (requires OpenAI API key in Sentry)
- `search_issues`: Natural language search for issues (requires OpenAI API key in Sentry)
- `analyze_issue_with_seer`: AI-powered root cause analysis of issues

**Project Management:**
- `create_team`: Create a new team in an organization
- `create_project`: Create a new project
- `update_project`: Update project settings and configuration

**DSN Management:**
- `create_dsn`: Create a Data Source Name for a project
- `find_dsns`: List all DSNs for a project

**Documentation:**
- `search_docs`: Search Sentry's documentation
- `get_doc`: Retrieve specific documentation pages

### Authentication

The Sentry MCP server uses stdio transport with access token authentication:

1. Create a User Auth Token in Sentry with required scopes
2. Add `SENTRY_ACCESS_TOKEN` secret to your repository
3. The token is automatically passed to the MCP server

Required scopes:
- Minimum (read-only): org:read, project:read, team:read, event:read
- Write operations: org:write, project:write, team:write, event:write

### Setup

1. **Create Sentry Access Token:**
   - Go to Sentry Settings → API → Auth Tokens
   - Create a new token with appropriate scopes
   - Copy the token value

2. **Add Repository Secret:**
   - Go to GitHub repository Settings → Secrets and variables → Actions
   - Create a new secret named `SENTRY_ACCESS_TOKEN`
   - Paste the Sentry token as the value

3. **For AI-powered search tools:**
   - Configure an OpenAI API key in your Sentry account settings
   - This enables `search_events` and `search_issues` natural language queries

### Example Usage in Workflows

**Issue Triage Assistant:**
```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/sentry.md
---

# Sentry Issue Triage

When an issue is created, search Sentry for related errors and provide context.

1. Use search_issues to find similar issues in Sentry
2. Use get_issue_details to get detailed information
3. Summarize findings and suggest potential fixes
```

**Release Monitor:**
```aw
---
on:
  schedule:
    - cron: "0 */6 * * *"
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/sentry.md
safe-outputs:
  create-issue:
---

# Release Health Monitor

Monitor recent releases for issues and create GitHub issues for critical problems.

1. Use find_releases to get recent releases
2. Use search_issues to find errors related to those releases
3. If critical issues found, create a GitHub issue with details
```

-->
