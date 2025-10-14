FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir drain3 mcp

# Copy the MCP server script
COPY drain3_mcp_server.py /app/

# Make the script executable
RUN chmod +x /app/drain3_mcp_server.py

# Set the entrypoint to run the MCP server
ENTRYPOINT ["python", "/app/drain3_mcp_server.py"]
