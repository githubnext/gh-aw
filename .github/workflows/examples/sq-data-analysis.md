---
on:
  workflow_dispatch:
    inputs:
      data_file:
        description: 'Path to CSV or database file to analyze'
        required: false
        default: 'data.csv'
imports:
  - ../shared/sq.md
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  create-issue:
    title-prefix: "[data-analysis] "
    labels: [automation, data]
---

# Analyze Structured Data with sq

Analyze the structured data file in the repository using the sq data wrangler tool.

**Task:** Use sq to analyze the data file at `${{ github.event.inputs.data_file }}` (or search for CSV/database files if not specified).

**Steps:**
1. Find structured data files in the repository (CSV, Excel, SQLite databases)
2. Use sq to inspect the data structure and schema
3. Generate summary statistics using sq queries
4. Identify interesting patterns or insights
5. Create a detailed analysis report as an issue

**Available sq commands:**
```bash
# Inspect data structure
sq inspect file.csv

# Query data with jq-like syntax
sq '.table | .column' file.csv

# Get JSON output
sq --json '.table' file.csv

# Count rows
sq '.table | count' file.csv

# Filter and aggregate
sq '.table | where(.column > 100) | group_by(.category)' file.csv
```

**Analysis report should include:**
- Data file location and type
- Schema/structure information
- Row counts and basic statistics
- Notable patterns or anomalies
- Recommendations for data quality improvements
