---
on:
  push:
    paths:
      - '**.csv'
imports:
  - ../shared/sq.md
permissions:
  contents: write
  pull-requests: write
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[auto] "
    labels: [automation, data-conversion]
---

# Convert CSV to JSON with sq

When CSV files are added or updated, automatically convert them to JSON format using sq.

**Task:** Find all CSV files in the push and convert them to JSON format.

**Steps:**
1. Identify CSV files that were added or modified in this push
2. For each CSV file, use sq to convert it to JSON format
3. Save the JSON files with the same name but .json extension
4. Create a pull request with the converted files

**sq conversion command:**
```bash
# Convert CSV to JSON
sq --json '.[]' file.csv > file.json
```

**Additional options:**
- Use `--pretty` flag for formatted JSON output
- Use specific column selections: `sq --json '.[] | .column1, .column2' file.csv`
- Filter rows: `sq --json '.[] | where(.status == "active")' file.csv`

**Note:** Only process CSV files that are under 10MB to avoid timeouts.
