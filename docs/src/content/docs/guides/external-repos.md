---
title: Working with External Repositories
description: Using GitHub Agentic Workflows with Azure DevOps Repos, GitLab, Bitbucket, and other non-GitHub repository platforms
sidebar:
  badge: { text: 'Advanced', variant: 'caution' }
---

GitHub Agentic Workflows is designed to run as GitHub Actions and primarily supports GitHub repositories. However, you can still use agentic workflows with code hosted on external platforms like Azure DevOps Repos, GitLab, or Bitbucket through repository mirroring and cross-platform integration patterns.

## Current Support Status

GitHub Agentic Workflows does **not** support external repository platforms as first-class hosts. The tool:

- Requires GitHub Actions as the execution environment
- Uses GitHub APIs for repository operations (issues, PRs, comments)
- Relies on GitHub authentication and permissions model
- Integrates with GitHub's security and workflow features

External platforms like Azure DevOps Repos, GitLab, and Bitbucket have different APIs, authentication mechanisms, and workflow systems that are not directly compatible.

## Recommended Approaches

### Mirror to GitHub (Recommended)

The most effective approach is to mirror your external repository to GitHub and run agentic workflows from the GitHub mirror. This provides full functionality while keeping your source of truth elsewhere.

**Benefits:**
- Full agentic workflow functionality
- No authentication complexity
- Secure execution in GitHub Actions
- Can run workflows repeatedly in "campaign" mode for iterative improvements

**Setup:**

1. **Create a GitHub mirror repository:**
   ```bash
   gh repo create my-org/my-project-mirror --private
   ```

2. **Configure automatic synchronization:**

   For Azure DevOps Repos, use a GitHub Actions workflow in the mirror repository:

   ```yaml
   name: Sync from Azure DevOps
   on:
     schedule:
       - cron: '*/15 * * * *'  # Every 15 minutes
     workflow_dispatch:

   jobs:
     sync:
       runs-on: ubuntu-latest
       steps:
         - name: Mirror repository
           env:
             AZURE_PAT: ${{ secrets.AZURE_DEVOPS_PAT }}
           run: |
             git clone https://${AZURE_PAT}@dev.azure.com/org/project/_git/repo temp-repo
             cd temp-repo
             git remote add github https://x-access-token:${{ secrets.GITHUB_TOKEN }}@github.com/${{ github.repository }}.git
             git push github --mirror
   ```

3. **Run agentic workflows on the mirror:**
   ```bash
   cd my-project-mirror
   gh aw init
   gh aw add workflow-name
   gh aw compile
   ```

4. **Sync results back (optional):**
   If workflows create artifacts or changes you need in Azure DevOps, use safe-outputs to create tracking issues or use Azure DevOps APIs from workflow steps.

### SideRepoOps Pattern

Use a separate GitHub repository to host agentic workflows that read from and optionally interact with your external repository through APIs or cloning.

**Architecture:**

```
┌─────────────────────┐          ┌──────────────────────┐
│  GitHub Repo        │          │  Azure DevOps Repo   │
│  (workflows only)   │ ────────>│  (source code)       │
│                     │   Clone  │                      │
│  - .github/         │    or    │  - src/              │
│    workflows/       │   API    │  - tests/            │
└─────────────────────┘          └──────────────────────┘
```

**Example workflow:**

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      analysis_task:
        description: "What analysis should the agent perform?"
        required: true

engine: copilot

permissions:
  contents: read

safe-outputs:
  create-issue:
    title-prefix: "[analysis] "
    labels: [automation]

tools:
  bash:
  github:
    mode: remote
    toolsets: [default]
---

# Azure DevOps Code Analyzer

Clone and analyze code from Azure DevOps, then create GitHub issues with findings.

## Task

{{ inputs.analysis_task }}

## Steps

1. Clone the Azure DevOps repository using credentials
2. Analyze the codebase according to the task description
3. Create a GitHub issue with findings and recommendations

## Authentication

Azure DevOps credentials are available via secrets. Use:
- Organization: `dev.azure.com/my-org`
- Project: `my-project`
- Repository: `my-repo`
- PAT stored in: `secrets.AZURE_DEVOPS_PAT`

## Commands

```bash
# Clone repository
git clone https://${{ secrets.AZURE_DEVOPS_PAT }}@dev.azure.com/my-org/my-project/_git/my-repo
cd my-repo

# Your analysis commands here
# Results can be used to create GitHub issues via safe-outputs
```
```

**Setup requirements:**

1. Create a Personal Access Token in Azure DevOps with Code (Read) permissions
2. Store it as a GitHub secret: `gh secret set AZURE_DEVOPS_PAT -a actions`
3. Configure the workflow to clone and analyze the external repository
4. Use safe-outputs to create issues/PRs in the GitHub repository

See [SideRepoOps Guide](/gh-aw/guides/siderepoops/) for complete patterns and best practices.

### Custom MCP Server Integration

For advanced use cases, create a custom MCP server that wraps the external platform's API (Azure DevOps, GitLab, Bitbucket) and provides tools to the agent.

**Example: Azure DevOps MCP server**

```javascript
// azure-devops-mcp-server.js
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import * as azdev from 'azure-devops-node-api';

const server = new Server({
  name: 'azure-devops-mcp-server',
  version: '1.0.0',
}, {
  capabilities: {
    tools: {},
  },
});

// Tool: List work items
server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'list_work_items',
        description: 'List work items from Azure DevOps',
        inputSchema: {
          type: 'object',
          properties: {
            project: { type: 'string', description: 'Project name' },
            wiql: { type: 'string', description: 'Work Item Query Language query' },
          },
          required: ['project'],
        },
      },
    ],
  };
});

// Tool implementation
server.setRequestHandler('tools/call', async (request) => {
  if (request.params.name === 'list_work_items') {
    const authHandler = azdev.getPersonalAccessTokenHandler(process.env.AZURE_DEVOPS_PAT);
    const connection = new azdev.WebApi('https://dev.azure.com/myorg', authHandler);
    const workItemApi = await connection.getWorkItemTrackingApi();
    
    // Query work items...
    const results = await workItemApi.queryByWiql({ query: request.params.arguments.wiql });
    
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(results, null, 2),
        },
      ],
    };
  }
});

const transport = new StdioServerTransport();
server.connect(transport);
```

**Workflow configuration:**

```yaml
mcp-servers:
  azure-devops:
    command: "node"
    args: ["./azure-devops-mcp-server.js"]
    env:
      AZURE_DEVOPS_PAT: "${{ secrets.AZURE_DEVOPS_PAT }}"
      AZURE_DEVOPS_ORG: "https://dev.azure.com/myorg"
```

See [MCP Integration Guide](/gh-aw/guides/mcps/) for complete MCP server development patterns.

## Platform-Specific Considerations

### Azure DevOps Repos

**Authentication:**
- Create a [Personal Access Token (PAT)](https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate) with appropriate scopes
- Store as GitHub secret: `gh secret set AZURE_DEVOPS_PAT -a actions`
- Use in git URLs: `https://${AZURE_DEVOPS_PAT}@dev.azure.com/org/project/_git/repo`

**Clone URL format:**
```bash
git clone https://dev.azure.com/organization/project/_git/repository
```

**API Access:**
- [Azure DevOps REST API](https://learn.microsoft.com/en-us/rest/api/azure/devops/)
- [azure-devops-node-api](https://www.npmjs.com/package/azure-devops-node-api) SDK for Node.js

**Work Item Integration:**
If you want to create Azure DevOps work items from workflows, use the REST API or SDK from custom workflow steps.

### GitLab

**Clone URL format:**
```bash
git clone https://gitlab.com/group/project.git
# or with authentication
git clone https://${GITLAB_TOKEN}@gitlab.com/group/project.git
```

**API Access:**
- [GitLab REST API](https://docs.gitlab.com/ee/api/)
- Authentication via Personal Access Token or Project Access Token

### Bitbucket

**Clone URL format:**
```bash
git clone https://bitbucket.org/workspace/repository.git
# or with authentication
git clone https://${BITBUCKET_USERNAME}:${BITBUCKET_APP_PASSWORD}@bitbucket.org/workspace/repository.git
```

**API Access:**
- [Bitbucket Cloud REST API](https://developer.atlassian.com/cloud/bitbucket/rest/)
- Authentication via App Password or OAuth

## Security Best Practices

When working with external repositories:

1. **Use fine-grained access tokens:**
   - Limit token permissions to minimum required scopes
   - Set expiration dates and rotate regularly
   - Store in GitHub Secrets, never in code

2. **Keep credentials isolated:**
   - Store external platform credentials as GitHub secrets
   - Use read-only tokens when possible
   - Separate tokens for read vs. write operations

3. **Audit and monitor:**
   - Review workflow run logs regularly
   - Monitor token usage and access patterns
   - Revoke compromised tokens immediately

4. **Use safe-outputs:**
   - Never give agents direct write access to external platforms
   - Use safe-outputs for GitHub resources (issues, PRs, comments)
   - Implement approval workflows for sensitive operations

5. **Repository mirroring security:**
   - Keep mirror repositories private unless source is public
   - Use separate tokens for sync vs. workflow operations
   - Enable branch protection on mirror repositories

See [Security Best Practices Guide](/gh-aw/guides/security/) for comprehensive security patterns.

## Common Use Cases

### Performance and Test Analysis

Mirror an Azure DevOps repository to GitHub and run agentic workflows repeatedly to analyze performance, suggest optimizations, and create tracking issues.

```bash
# Setup mirror
gh repo create my-org/project-mirror --private
# Configure sync (see Mirror to GitHub above)
# Add performance analysis workflow
gh aw add performance-analyzer
# Run repeatedly for improvements
gh aw run performance-analyzer --repo my-org/project-mirror
```

### Cross-Platform Issue Tracking

Use workflows to sync issues between platforms or create GitHub issues that reference work items in external systems.

### Code Quality Monitoring

Schedule workflows to clone external repositories, run analysis, and report findings as GitHub issues or discussions.

### Release Coordination

Monitor external repositories for releases and coordinate cross-platform release processes.

## Limitations

When working with external repositories through these patterns:

- **No native pull request creation:** Cannot directly create PRs in Azure DevOps, GitLab, or Bitbucket using safe-outputs
- **API differences:** Each platform has unique APIs and capabilities
- **Authentication complexity:** Managing multiple authentication systems
- **Sync delays:** Mirror-based approaches have synchronization latency
- **Limited context:** Agent may not have full repository history or context

## Getting Help

If you're implementing external repository integration:

1. Start with the [Mirror to GitHub approach](#mirror-to-github-recommended) for simplest setup
2. Review [SideRepoOps patterns](/gh-aw/guides/siderepoops/) for API-based integration
3. Explore [Custom MCP Servers](/gh-aw/guides/mcps/) for advanced scenarios
4. Search [existing issues](https://github.com/githubnext/gh-aw/issues) for platform-specific questions
5. Share your approach in [GitHub Discussions](https://github.com/githubnext/gh-aw/discussions)

## Related Resources

- [SideRepoOps Guide](/gh-aw/guides/siderepoops/) - Running workflows from separate repositories
- [MultiRepoOps Guide](/gh-aw/guides/multirepoops/) - Cross-repository coordination patterns
- [MCP Integration Guide](/gh-aw/guides/mcps/) - Custom MCP server development
- [Security Best Practices](/gh-aw/guides/security/) - Secure workflow patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Output sanitization and permissions
