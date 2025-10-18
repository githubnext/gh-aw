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
  create-discussion:
    category: "general"
    max: 1
imports:
  - shared/mcp/jupyter.md
post-steps:
  - name: Upload Chart Artifact
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: file-size-distribution-chart
      path: /tmp/file-size-chart.png
      retention-days: 7
      if-no-files-found: ignore
---

# File Size Distribution Analysis

You are a data analyst using Jupyter notebooks to analyze the repository.

## Your Task

1. **Analyze Repository Files**: Scan the repository ${{ github.repository }} to gather file size information for all files
2. **Generate Distribution Chart**: Use Jupyter notebook to create a visual distribution chart showing:
   - File size ranges (e.g., <1KB, 1-10KB, 10-100KB, 100KB-1MB, >1MB)
   - Number of files in each size range
   - A bar chart or histogram visualization
3. **Save Chart**: Save the generated chart as `/tmp/file-size-chart.png`
4. **Create Discussion**: Create a GitHub discussion with:
   - Summary statistics (total files, average size, median size, largest files)
   - Description of the file size distribution
   - Link to the uploaded chart artifact
   - Any interesting insights about the repository structure

## Important Notes

- Use Python code in the Jupyter notebook for data processing
- Use matplotlib or seaborn for visualization
- Ensure the chart is clear, labeled, and professional-looking
- The chart will be automatically uploaded as an artifact after execution
