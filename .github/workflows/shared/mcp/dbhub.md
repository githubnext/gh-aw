---
mcp-servers:
  dbhub:
    container: "bytebase/dbhub"
    version: "latest"
    entrypointArgs:
      - "--demo"
    allowed:
      - execute_sql
      - search_objects
---

<!--

# DBHub SQLite MCP Server
# Access to local SQLite databases via MCP

Provides a zero-dependency, token efficient MCP server for database operations.
DBHub implements the Model Context Protocol (MCP) server interface for SQLite
(and other databases like PostgreSQL, MySQL, MariaDB, SQL Server).

This configuration uses demo mode with an in-memory SQLite database for testing
and exploration. For production use, mount a SQLite database file or configure
a DSN to connect to an external database.

Repository: https://github.com/bytebase/dbhub
Documentation: https://dbhub.ai/

## Available Tools

DBHub provides two core MCP tools designed for token efficiency:

- **execute_sql**: Execute SQL queries with transaction support and safety controls
  - Supports SELECT, INSERT, UPDATE, DELETE statements
  - Built-in guardrails: read-only mode, row limiting, query timeout
  - Transaction support for multi-statement operations
  - Returns query results in structured format

- **search_objects**: Search and explore database schemas with progressive disclosure
  - Search tables, columns, indexes, views, and stored procedures
  - Filter by object type, name pattern, or schema
  - Progressive disclosure to minimize token usage
  - Returns metadata about database objects

- **Custom Tools**: Define reusable, parameterized SQL operations in configuration
  - Create domain-specific tools for common queries
  - Parameterized queries with type safety
  - Requires custom `dbhub.toml` configuration file

## Configuration

### Demo Mode (Default)

This shared configuration runs DBHub in demo mode with an in-memory SQLite database.
This is ideal for testing, learning, and exploration without requiring external setup.

The container uses the default stdio transport for MCP communication.

### Production Mode with SQLite File

To use a real SQLite database file, modify the MCP server configuration:

```yaml
mcp-servers:
  dbhub:
    container: "bytebase/dbhub"
    version: "latest"
    entrypointArgs:
      - "--dsn"
      - "sqlite:///data/your-database.db"
    volumes:
      - ./data:/data
    allowed:
      - execute_sql
      - search_objects
```

### Other Database Types

DBHub supports multiple database types via DSN (Data Source Name):

**PostgreSQL:**
```yaml
entrypointArgs:
  - "--dsn"
  - "******host:5432/dbname?sslmode=disable"
```

**MySQL:**
```yaml
entrypointArgs:
  - "--dsn"
  - "******host:3306/dbname"
```

**SQL Server:**
```yaml
entrypointArgs:
  - "--dsn"
  - "******host:1433?database=dbname"
```

For multi-database setups, see: https://dbhub.ai/config/toml

## Safety Features

DBHub includes built-in guardrails to prevent runaway operations:

- **Read-only mode**: Restrict operations to SELECT queries only
- **Row limiting**: Limit maximum rows returned per query
- **Query timeout**: Automatic timeout for long-running queries
- **Transaction safety**: Rollback on errors, commit on success

These can be configured via command-line options or TOML configuration.

## Setup

### Basic Usage (Demo Mode)

No setup required! The demo mode creates an in-memory SQLite database
automatically with sample data for exploration.

```yaml
imports:
  - shared/mcp/dbhub.md
```

### Production Usage

1. **Prepare your database**:
   - For SQLite: Place your `.db` file in a mounted volume
   - For other databases: Ensure network connectivity and credentials

2. **Configure secrets** (for external databases):
   - Add database credentials to GitHub repository secrets
   - Example: `DB_PASSWORD`, `DB_USER`, `DB_HOST`

3. **Update the MCP server configuration** in your shared workflow or inline:
   ```yaml
   mcp-servers:
     dbhub:
       container: "bytebase/dbhub"
       version: "latest"
       entrypointArgs:
         - "--dsn"
         - "postgres://user:PASSWORD@host:5432/dbname"
       allowed:
         - execute_sql
         - search_objects
   ```
   
   Replace `PASSWORD` with your secret reference (e.g., `secrets.DB_PASSWORD`) in your actual workflow.

4. **Include in your workflow**:
   ```yaml
   imports:
     - shared/mcp/dbhub.md
   ```

## Example Usage

### Query SQLite Database

```aw
---
on: workflow_dispatch
permissions: read
engine: copilot
imports:
  - shared/mcp/dbhub.md
---

# SQLite Database Query

Explore the database schema using search_objects, then execute queries
to analyze the data. Provide insights about the database structure and
key metrics from the data.

1. Use search_objects to discover available tables
2. Execute SQL queries to retrieve relevant data
3. Summarize findings in a clear, structured format
```

### Data Analysis Workflow

```aw
---
on: daily
permissions: read
engine: copilot
imports:
  - shared/mcp/dbhub.md
  - shared/reporting.md
safe-outputs:
  create-discussion:
---

# Daily Database Metrics

Connect to the SQLite database and generate a daily metrics report:

1. Query key performance indicators from the database
2. Compare with historical data to identify trends
3. Create a discussion with visualizations and insights

Focus on actionable metrics and notable changes from previous days.
```

## Security Considerations

- **Container isolation**: DBHub runs in an isolated Docker container
- **Credentials**: Store database passwords in GitHub secrets, never in code
- **Read-only mode**: Enable `--read-only` flag for query-only access
- **Row limits**: Use `--max-rows` to prevent excessive data retrieval
- **Query timeout**: Configure `--timeout` to prevent long-running queries

## Workbench

DBHub includes a built-in web interface for testing queries and custom tools
without requiring an MCP client. This is useful for local development and debugging.

To run the workbench locally with HTTP transport:

```bash
docker run --rm --init \
  --name dbhub \
  --publish 8080:8080 \
  bytebase/dbhub \
  --transport http \
  --port 8080 \
  --demo
```

Then visit: http://localhost:8080

Note: The shared workflow uses stdio transport (default) for MCP protocol communication,
not HTTP. The workbench is a separate feature for local testing.

## More Information

- **Repository**: https://github.com/bytebase/dbhub
- **Documentation**: https://dbhub.ai/
- **Docker Image**: https://hub.docker.com/r/bytebase/dbhub
- **NPM Package**: https://www.npmjs.com/package/@bytebase/dbhub
- **License**: MIT (see repository for details)

## Troubleshooting

### Connection Issues

If the MCP server cannot connect to DBHub:

1. Check container status in GitHub Actions logs
2. Verify the container image is accessible
3. Check for any error messages in the MCP server logs

### Query Errors

If queries fail or return unexpected results:

1. Use `search_objects` to verify table and column names
2. Check SQL syntax for the specific database type
3. Verify row limits and timeout settings aren't too restrictive

### Demo Mode Limitations

Demo mode uses an in-memory database that resets on each workflow run.
For persistent data, use a real database with appropriate DSN configuration.

-->
