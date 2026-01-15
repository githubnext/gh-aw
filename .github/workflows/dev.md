---
on: 
  workflow_dispatch:
    inputs:
      issue_number:
        description: Issue number to read
        required: true
        type: string
name: Dev
description: Find security issues in Go source code using CodeQL
timeout-minutes: 15
strict: false
sandbox: false
engine: copilot

permissions:
  contents: read
  issues: read
  security-events: read

network:
  allowed:
    - "*"

tools:
  github:
    toolsets: [issues]

imports:
  - shared/mcp/codeql.md

safe-outputs:
  staged: true
  add-comment:
    max: 1
  create-issue:
    title-prefix: "[security] "
    labels: [security, codeql]
---

# CodeQL Security Analysis for Go Code

Analyze the Go source code in this repository to find security vulnerabilities using CodeQL.

**Requirements:**
1. Use the CodeQL MCP server to analyze the Go codebase
2. Register the CodeQL database at `/tmp/codeql-db` with the MCP server using `register_database`
3. Run security-focused CodeQL queries to identify potential vulnerabilities in the Go code
4. Focus on common security issues like:
   - SQL injection vulnerabilities
   - Command injection risks
   - Path traversal vulnerabilities
   - Insecure cryptographic practices
   - Uncontrolled resource consumption
   - Unsafe reflection usage
5. Decode the query results using `decode_bqrs` to get human-readable output
6. Analyze the findings and create a summary report
7. If security issues are found, create a new issue with:
   - Clear description of each vulnerability
   - Location (file and line numbers)
   - Severity assessment
   - Recommended fixes
8. Post a comment on issue #${{ github.event.inputs.issue_number }} with a summary of the analysis
9. Use staged mode to preview all outputs before creating them

**CodeQL Database Location**: `/tmp/codeql-db`

**Expected Workflow**:
1. Register the database: `register_database("/tmp/codeql-db")`
2. Run security queries or evaluate specific security patterns
3. Decode results to JSON format for analysis
4. Generate actionable security report
5. Create issue if vulnerabilities found
6. Comment on the triggering issue with summary
