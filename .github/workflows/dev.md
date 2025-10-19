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
  - shared/mcp/python-code-interpreter.md
---

# Repository File Size Histogram

You are a data analyst creating visualizations of repository file sizes.

**IMPORTANT - Sandbox Environment:**
You are running in a restricted sandbox where direct bash Python execution is locked down for security. You MUST use the `run_python_query` MCP tool to execute any Python code. Local `python` command will not work. All Python code execution must go through the python-code-interpreter MCP server.

## Your Task

1. **Collect File Information**: Use bash commands to find all files in the repository ${{ github.repository }} and collect their sizes
   - Use `find` command to list all files (excluding .git directory)
   - Collect file sizes in bytes
   - Create a data file with file paths and their sizes

2. **Generate Histogram Plot**: Use the `run_python_query` tool from the python-code-interpreter to:
   - Read the collected file size data
   - Create a histogram showing the distribution of file sizes
   - Use matplotlib to generate a professional-looking plot
   - Save the plot as `histogram.png` in the run directory
   - Include proper labels, title, and formatting

3. **Upload Results**: Upload the generated histogram.png as an artifact so it can be viewed

## Important Notes

- Focus on meaningful file sizes (exclude empty files if needed)
- Use appropriate bin sizes for the histogram
- Make the visualization clear and professional
- Include axis labels and a descriptive title
