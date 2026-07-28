---
mcp-servers:
  azure:
    container: "mcr.microsoft.com/azure-sdk/azure-mcp"
    version: "latest"
    entrypointArgs:
      - "server"
      - "start"
      - "--read-only"
    env:
      AZURE_TENANT_ID: "${{ secrets.AZURE_TENANT_ID }}"
      AZURE_CLIENT_ID: "${{ secrets.AZURE_CLIENT_ID }}"
      AZURE_CLIENT_SECRET: "${{ secrets.AZURE_CLIENT_SECRET }}"
    # Security decision (2026-05-19): restrict Azure MCP to read-only discovery tools.
    # This replaces wildcard access to reduce blast radius if a future tool is added upstream.
    allowed:
      - "subscription_list"
      - "subscription_get"
      - "group_list"
      - "group_get"
      - "resource_list"
      - "resource_get"

network:
  allowed:
    - login.microsoftonline.com
    - management.azure.com
    - "*.azure.com"
---

<!--
## Azure MCP Server

This shared configuration provides the Microsoft Azure MCP Server with read-only access to Azure services.

The Azure MCP Server supercharges agentic workflows with Azure context across **40+ different Azure services**, including:

- **Azure Key Vault** - Secrets and certificate management
- **Azure Storage** - Blob storage operations
- **Azure SQL Database** - Database management
- **Azure Service Bus** - Message queuing
- **Azure RBAC** - Access control management
- **Azure Monitor** - Monitoring and diagnostics
- **Azure Virtual Network** - Network configuration
- **Azure Virtual Desktop** - Virtual desktop infrastructure
- **Azure AI Foundry** - AI service integration
- **Azure Resource Groups** - Resource organization
- And many more...

### Authentication: Service Principal (Client Secret)

This shared workflow uses a service principal with a client secret. This is the simplest
approach when you have a dedicated service principal for your workflow.

**Required Secrets**:
- `AZURE_TENANT_ID`: Your Azure tenant ID
- `AZURE_CLIENT_ID`: Your Azure client (application) ID
- `AZURE_CLIENT_SECRET`: Your Azure client secret

### Authentication: OIDC (Workload Identity Federation)

If your workflow uses GitHub Actions OIDC (no long-lived secrets), use the command-based
approach with `shared/azure-auth.md` instead:

```yaml
---
permissions:
  contents: read
  id-token: write
imports:
  - uses: shared/azure-auth.md
    with:
      azure-client-id: "00000000-0000-0000-0000-000000000000"  # your app client ID
      azure-tenant-id: "00000000-0000-0000-0000-000000000000"  # your tenant ID
mcp-servers:
  azure:
    command: npx
    args: ["-y", "@azure/mcp@latest", "server", "start"]
    allowed: ["*"]
---
```

`shared/azure-auth.md` re-authenticates the Azure CLI inside the agent sandbox using a
short-lived OIDC token. The `@azure/mcp` package resolves credentials through
`DefaultAzureCredential`, which succeeds via the `AzureCliCredential` provider after
`shared/azure-auth.md` runs.

### Configuration

This shared workflow runs the Azure MCP server in **read-only mode** using Docker, preventing any write operations to Azure resources.

**Container**: `mcr.microsoft.com/azure-sdk/azure-mcp:latest`

### Setup

1. Create an Azure Service Principal with read-only permissions:
   ```bash
   az ad sp create-for-rbac --name "gh-aw-readonly" --role Reader --scopes /subscriptions/{subscription-id}
   ```

2. Add the following secrets to your GitHub repository:
   - `AZURE_TENANT_ID`: Tenant ID from the service principal output
   - `AZURE_CLIENT_ID`: App ID from the service principal output
   - `AZURE_CLIENT_SECRET`: Password from the service principal output

3. Include this configuration in your workflow:
   ```yaml
   imports:
     - shared/mcp/azure.md
   ```

### Example Usage

```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: claude
imports:
  - shared/mcp/azure.md
---

# Azure Resource Analyzer

Analyze Azure resources mentioned in issue #${{ github.event.issue.number }} and provide insights.

Review the issue content and identify any Azure resource references (subscription IDs, resource groups, storage accounts, etc.).

Use the Azure MCP tools to query information about these resources and provide a summary.
```

### Available Tools

All Azure MCP server tools are available with read-only access. Tools are organized by Azure service namespaces in the default mode.

For complete documentation of available services and tools, see:
- [Azure MCP Server Documentation](https://learn.microsoft.com/azure/developer/azure-mcp-server/)
- [Azure MCP Commands Reference](https://github.com/microsoft/mcp/blob/main/servers/Azure.Mcp.Server/docs/azmcp-commands.md)

### Security

- **Read-only mode**: The `--read-only` flag ensures only read operations are permitted
- **Credential security**: Azure credentials are handled securely via the Azure Identity SDK
- **Service Principal permissions**: Use a service principal with minimal Reader role

### More Information

- GitHub Repository: https://github.com/microsoft/mcp/tree/main/servers/Azure.Mcp.Server
- Docker Image: https://mcr.microsoft.com/artifact/mar/azure-sdk/azure-mcp
- License: MIT

-->
