---
runtimes:
  sq:
    version: "latest"
    action-repo: "docker://ghcr.io/neilotoole/sq"

steps:
  - name: Verify sq installation
    run: |
      docker run --rm ghcr.io/neilotoole/sq:latest sq version
      echo "sq is available via Docker container"
---
<!--
## sq Data Wrangler

This shared configuration provides setup for `sq`, a command-line tool that offers jq-style access to structured data sources including SQL databases, CSV, Excel, and other document formats.

### About sq

`sq` is the lovechild of sql+jq. It executes jq-like queries or database-native SQL, can join across sources (e.g., join a CSV file to a Postgres table), and outputs to multiple formats including JSON, Excel, CSV, HTML, Markdown, and XML.

**Links:**
- Documentation: https://sq.io/
- GitHub Repository: https://github.com/neilotoole/sq
- Terminal Trove: https://terminaltrove.com/sq/
- Docker Image: https://github.com/neilotoole/sq/pkgs/container/sq

### Installation

The shared workflow configures sq via Docker container. The `ghcr.io/neilotoole/sq` image is preloaded with `sq` and related tools like `jq`.

### Usage in Workflows

Import this shared configuration to make sq available in your workflow:

```yaml
imports:
  - shared/sq.md
```

Then use sq commands in your workflow steps:

```bash
# Run sq commands via Docker
docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq inspect /data/database.db

# Query a CSV file
docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq '.actor | .first_name, .last_name' /data/actors.csv

# Join data from multiple sources
docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq '@csv_data | join @postgres_db.users'
```

### Common Use Cases

1. **Query structured data files**: Use jq-like syntax to query CSV, Excel, JSON files
2. **Cross-source joins**: Combine data from different sources (databases, files)
3. **Data format conversion**: Convert between formats (CSV to JSON, Excel to Markdown, etc.)
4. **Database inspection**: View metadata about database structure
5. **Database operations**: Copy, truncate, or drop tables
6. **Data comparison**: Use `sq diff` to compare tables or databases

### Example Workflow

```yaml
---
on:
  workflow_dispatch:
imports:
  - shared/sq.md
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[data-analysis] "
---

# Data Analysis with sq

Analyze the CSV files in the repository using sq and create a summary report.

Use sq to:
1. Inspect the data structure
2. Query for interesting patterns
3. Generate summary statistics
4. Create a formatted report

Available sq commands via Docker:
- `docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq inspect /data/file.csv`
- `docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq '.table | select(.column > 100)' /data/file.csv`
- `docker run --rm -v $(pwd):/data ghcr.io/neilotoole/sq sq --json '.table' /data/file.csv`
```

### Tips

- Mount the working directory as a volume: `-v $(pwd):/data`
- Use `--rm` flag to automatically clean up containers
- Specify output format with flags like `--json`, `--csv`, `--markdown`
- Access GitHub workspace with `-v ${{ github.workspace }}:/workspace`
- For databases, pass connection strings via environment variables
-->

You have access to the `sq` data wrangling tool for working with structured data sources.

**sq capabilities:**
- Query CSV, Excel, JSON, and database files using jq-like syntax
- Join data across different source types
- Convert between data formats
- Inspect database structures and metadata
- Perform database operations (copy, truncate, drop tables)
- Compare data with `sq diff`

**Using sq in this workflow:**
All sq commands should be run via Docker:
```bash
docker run --rm -v ${{ github.workspace }}:/workspace -w /workspace ghcr.io/neilotoole/sq sq [command]
```

**Example commands:**
```bash
# Inspect a data file
docker run --rm -v ${{ github.workspace }}:/workspace -w /workspace ghcr.io/neilotoole/sq sq inspect file.csv

# Query data with jq-like syntax
docker run --rm -v ${{ github.workspace }}:/workspace -w /workspace ghcr.io/neilotoole/sq sq '.table | .column' file.csv

# Output as JSON
docker run --rm -v ${{ github.workspace }}:/workspace -w /workspace ghcr.io/neilotoole/sq sq --json '.table' file.csv
```

For more information, see: https://sq.io/docs/
