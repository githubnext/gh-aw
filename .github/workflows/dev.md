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

services:
  jupyter:
    image: jupyter/base-notebook:latest
    ports:
      - 8888:8888
    env:
      JUPYTER_TOKEN: ${{ github.run_id }}

steps:
  - name: Generate and verify Jupyter Token
    id: jupyter-token
    run: |
      # Use github.run_id as the token (it's unique and secure enough for ephemeral sessions)
      TOKEN="${{ github.run_id }}"
      echo "token=$TOKEN" >> $GITHUB_OUTPUT
      echo "Generated Jupyter token from run ID"
      
  - name: Wait for Jupyter to be ready
    run: |
      echo "Waiting for Jupyter server to start..."
      for i in {1..30}; do
        if curl -f http://jupyter:8888/api 2>/dev/null; then
          echo "✓ Jupyter server is ready!"
          break
        fi
        echo "Attempt $i: Waiting for Jupyter..."
        sleep 2
      done
      curl -f http://jupyter:8888/api || (echo "Failed to connect to Jupyter" && exit 1)

tools:
  edit:
  github:
  
mcp-servers:
  jupyter:
    container: "datalayer/jupyter-mcp-server"
    version: "latest"
    env:
      JUPYTER_URL: "http://jupyter:8888"
      JUPYTER_TOKEN: "${{ github.run_id }}"
      DOCUMENT_ID: "notebook.ipynb"
      ALLOW_IMG_OUTPUT: "true"
    allowed: ["*"]

safe-outputs:
  create-discussion:
    category: "general"
    max: 1
  upload-assets:
---

# File Size Distribution Analysis

You are a data analyst using Jupyter notebooks to analyze the repository.

## Your Task

1. **Analyze Repository Files**: Scan the repository ${{ github.repository }} to gather file size information for all files
2. **Generate Distribution Chart**: Use Jupyter notebook to create a visual distribution chart showing:
   - File size ranges (e.g., <1KB, 1-10KB, 10-100KB, 100KB-1MB, >1MB)
   - Number of files in each size range
   - A bar chart or histogram visualization
3. **Save Chart**: Save the generated chart as a file in the current directory
4. **Upload Chart as Asset**: Use the `upload_asset` output type to upload the chart file
5. **Create Discussion**: Create a GitHub discussion with:
   - Summary statistics (total files, average size, median size, largest files)
   - Description of the file size distribution
   - Link to the uploaded chart asset on the assets branch
   - Any interesting insights about the repository structure

## Important Notes

- Use Python code in the Jupyter notebook for data processing
- Use matplotlib or seaborn for visualization
- Ensure the chart is clear, labeled, and professional-looking
- The chart will be automatically uploaded to the assets branch via safe-outputs
