---
mcp-servers:
  fabric-rti:
    command: "uvx"
    args:
      - "microsoft-fabric-rti-mcp"
    allowed: ["*"]
---

## Microsoft Fabric Real-Time Intelligence (RTI) MCP Server

This shared configuration provides the Microsoft Fabric Real-Time Intelligence (RTI) MCP Server for AI-assisted data querying and analysis.

The Fabric RTI MCP Server enables AI agents to interact with Microsoft Fabric RTI services by providing tools through the MCP interface, allowing for seamless data querying and analysis capabilities.

### 🔍 Supported Services

**Eventhouse (Kusto)**: Execute KQL queries against Microsoft Fabric RTI [Eventhouse](https://aka.ms/eventhouse) and [Azure Data Explorer (ADX)](https://aka.ms/adx).

**Eventstreams**: Manage Microsoft Fabric [Eventstreams](https://learn.microsoft.com/fabric/real-time-intelligence/eventstream/eventstream-introduction) for real-time data processing:
- List Eventstreams in workspaces
- Get Eventstream details and definitions

### Available Tools

#### Eventhouse (Kusto) - 12 Tools:
- **`kusto_known_services`** - List all available Kusto services configured in the MCP
- **`kusto_query`** - Execute KQL queries on the specified database
- **`kusto_command`** - Execute Kusto management commands (destructive operations)
- **`kusto_list_databases`** - List all databases in the Kusto cluster
- **`kusto_list_tables`** - List all tables in a specified database
- **`kusto_get_entities_schema`** - Get schema information for all entities (tables, materialized views, functions) in a database
- **`kusto_get_table_schema`** - Get detailed schema information for a specific table
- **`kusto_get_function_schema`** - Get schema information for a specific function, including parameters and output schema
- **`kusto_sample_table_data`** - Retrieve random sample records from a specified table
- **`kusto_sample_function_data`** - Retrieve random sample records from the result of a function call
- **`kusto_ingest_inline_into_table`** - Ingest inline CSV data into a specified table
- **`kusto_get_shots`** - Retrieve semantically similar query examples from a shots table using AI embeddings

#### Eventstreams - 6 Tools:
- **`list_eventstreams`** - List all Eventstreams in your Fabric workspace
- **`get_eventstream`** - Get detailed information about a specific Eventstream
- **`get_eventstream_definition`** - Retrieve complete JSON definition of an Eventstream

### 🔑 Authentication

The MCP Server seamlessly integrates with your host operating system's authentication mechanisms. It uses Azure Identity via [`DefaultAzureCredential`](https://learn.microsoft.com/azure/developer/python/sdk/authentication/credential-chains?tabs=dac), which tries these authentication methods in order:

1. **Environment Variables** (`EnvironmentCredential`) - Perfect for CI/CD pipelines
2. **Visual Studio** (`VisualStudioCredential`) - Uses your Visual Studio credentials
3. **Azure CLI** (`AzureCliCredential`) - Uses your existing Azure CLI login
4. **Azure PowerShell** (`AzurePowerShellCredential`) - Uses your Az PowerShell login
5. **Azure Developer CLI** (`AzureDeveloperCliCredential`) - Uses your azd login
6. **Interactive Browser** (`InteractiveBrowserCredential`) - Falls back to browser-based login if needed

If you're already logged in through any of these methods, the Fabric RTI MCP Server will automatically use those credentials.

**Authentication Requirements:**
- Your Azure identity must have access to the Microsoft Fabric workspace and resources
- The identity should have appropriate permissions for Eventhouse and Eventstreams

### Setup

1. **Ensure you have authentication set up** via one of the methods above (Azure CLI is recommended):
   ```bash
   az login
   ```

2. **Include this configuration in your workflow**:
   ```yaml
   imports:
     - shared/mcp/fabric-rti.md
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
  - shared/mcp/fabric-rti.md
---

# Fabric RTI Data Analyzer

Analyze data mentioned in issue #${{ github.event.issue.number }} using Microsoft Fabric RTI.

Review the issue content and identify any data analysis requests related to Eventhouse or Eventstreams.

Use the Fabric RTI MCP tools to:
- List available databases and tables
- Execute KQL queries for data analysis
- Retrieve Eventstream information
- Provide insights based on the data
```

### Example Prompts

**Eventhouse Analytics:**
- "Get databases in my Eventhouse"
- "Sample 10 rows from table 'StormEvents' in Eventhouse"
- "What can you tell me about StormEvents data?"
- "Analyze the StormEvents to come up with trend analysis across past 10 years of data"
- "Analyze the commands in 'CommandExecution' table and categorize them as low/medium/high risks"

**Eventstream Management:**
- "List all Eventstreams in my workspace"
- "Show me the details of my IoT data Eventstream"

### Security

- **Credential Security**: Your credentials are always handled securely through the official [Azure Identity SDK](https://github.com/Azure/azure-sdk-for-net/blob/main/sdk/identity/Azure.Identity/README.md) - credentials are never stored or managed directly
- **Least Privilege**: Grant only the minimum required permissions to your Azure identity
- **Destructive Operations**: The `kusto_command` tool can execute management commands - use with caution

### More Information

- **GitHub Repository**: https://github.com/microsoft/fabric-rti-mcp
- **PyPI Package**: https://pypi.org/project/microsoft-fabric-rti-mcp/
- **Microsoft Fabric RTI Documentation**: https://aka.ms/fabricrti
- **License**: MIT
- **Status**: Public Preview

