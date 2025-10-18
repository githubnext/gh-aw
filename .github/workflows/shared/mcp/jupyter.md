---
steps:
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
---

<!--

# Jupyter Notebook MCP Server
# Manipulate Jupyter notebooks and execute code cells

Provides integration with Jupyter servers to run code cells, manage notebooks,
and visualize data using the Jupyter MCP Server.

Documentation: https://pypi.org/project/jupyter-mcp-server/

Available tools:
  - execute_cell: Execute code in a notebook cell
  - get_cell_output: Retrieve output from executed cells
  - create_notebook: Create new Jupyter notebooks
  - list_notebooks: List available notebooks
  - get_notebook_content: Read notebook contents

Configuration:
  The server connects to a Jupyter server instance using the provided URL and token.
  Set DOCUMENT_ID to specify the default notebook to work with.
  Enable ALLOW_IMG_OUTPUT to support image outputs from cells.

Setup:
  1. Start a Jupyter server locally or remotely
  2. Generate a Jupyter token for authentication
  3. Add the following secrets to your GitHub repository:
     - JUPYTER_TOKEN: Your Jupyter server authentication token

  4. Include in Your Workflow:
     imports:
       - shared/mcp/jupyter.md

Connection:
  The server connects to Jupyter via the JUPYTER_URL (default: http://host.docker.internal:8888)
  which allows Docker containers to access services running on the host machine.

Security:
  - Store the JUPYTER_TOKEN as a GitHub secret
  - Ensure your Jupyter server is properly secured
  - Consider network restrictions if running in production

Example Usage:
  Create a Jupyter notebook that analyzes repository data and generates visualizations.
  Execute Python code cells to process data and create charts.

Usage:
  imports:
    - shared/mcp/jupyter.md

-->
