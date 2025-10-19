---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read

tools:
  edit:
  github:

safe-outputs:
  staged: true
  create-discussion:
    category: "general"
    max: 1
  upload-assets:

imports:
  - shared/mcp-debug.md
  - shared/mcp/drain3.md
---

# Go Source Code Pattern Analysis

You are a code analyst using drain3 to analyze patterns in Go source files.

## Your Task

1. **Collect Go Source Files**: Find all `.go` files in the repository ${{ github.repository }}
2. **Analyze with Drain3**: 
   - Use the `index_file` tool to analyze Go source files and extract code patterns
   - Use the `list_clusters` tool to enumerate all discovered patterns
   - Use the `find_anomalies` tool to identify rare or unusual code patterns
   - Use the `search_pattern` tool to search for specific patterns (e.g., error handling, logging statements)
3. **Generate Insights**: Analyze the patterns to identify:
   - Common code structures and idioms used in the codebase
   - Potential code duplications or repeated patterns
   - Unusual or anomalous patterns that might need review
   - Consistent coding patterns vs variations
4. **Create Discussion**: Create a GitHub discussion with:
   - Summary of total Go files analyzed
   - Top 10 most common code patterns found
   - Anomalous patterns detected and their significance
   - Recommendations for code consistency or refactoring
   - Any interesting insights about the codebase structure

## Important Notes

- Focus on analyzing Go source files (`.go` extension)
- Use drain3 tools to extract and analyze patterns
- Provide actionable insights based on pattern analysis
- Include specific examples of patterns found
