# Serena MCP Server Container Image
# Multi-language semantic code analysis via Language Server Protocol (LSP)
# Supports Go, TypeScript, and Python analysis

# Stage 1: Build gopls
# Use golang:1.25-alpine to build gopls binary (compile-time only dependency)
FROM golang:1.25-alpine AS gopls-builder
# Pin to gopls v0.21.0 which is compatible with Go 1.25
RUN go install golang.org/x/tools/gopls@v0.21.0

# Stage 2: Final image
# python:3.13-slim uses Debian bookworm with significantly fewer CVEs than 3.11-slim
FROM python:3.13-slim

# OCI labels for GitHub Container Registry
LABEL org.opencontainers.image.title="Serena MCP Server"
LABEL org.opencontainers.image.description="A containerized MCP server with semantic code analysis for Python, Go, and TypeScript"
LABEL org.opencontainers.image.source="https://github.com/github/gh-aw/tree/main/containers/serena-mcp-server"
LABEL org.opencontainers.image.documentation="https://github.com/github/gh-aw/blob/main/containers/serena-mcp-server/README.md"
LABEL org.opencontainers.image.url="https://github.com/github/gh-aw/pkgs/container/serena-mcp-server"
LABEL org.opencontainers.image.vendor="GitHub"

# Set working directory
WORKDIR /app

# Install minimal system dependencies
# - git: required for some Serena operations
# - nodejs/npm: for TypeScript language server
# - ca-certificates: for HTTPS
# - golang-go: Go runtime for Go code analysis
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    nodejs \
    npm \
    ca-certificates \
    golang-go \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Copy pre-built gopls from builder stage
COPY --from=gopls-builder /go/bin/gopls /usr/local/bin/gopls

# Install Serena MCP server from GitHub
RUN pip install --no-cache-dir git+https://github.com/oraios/serena.git || \
    (echo "GitHub installation failed, trying PyPI..." && \
     pip install --no-cache-dir serena-agent)

# Install Python language server for Python code analysis
RUN pip install --no-cache-dir python-lsp-server[all]

# Install TypeScript language server (minimal)
RUN npm install -g typescript typescript-language-server && \
    rm -rf /root/.npm /tmp/*

# Create directories for Serena configuration and caching
RUN mkdir -p /workspace /tmp/serena-cache /root/.serena

# Set environment variables for Serena
ENV SERENA_WORKSPACE=/workspace
ENV SERENA_CACHE_DIR=/tmp/serena-cache

# Create a startup script that ensures multi-language support
# This script will initialize the workspace with Go, TypeScript, and Python if not already configured
RUN printf '#!/bin/bash\n\
set -e\n\
\n\
# Ensure .serena directory exists in workspace\n\
mkdir -p /workspace/.serena\n\
\n\
# Check if project.yml needs to be created or updated\n\
# Create/update if: file does not exist, OR it is missing any of the three required languages\n\
if [ ! -f /workspace/.serena/project.yml ] || \\\n\
   ! grep -q "^- go$" /workspace/.serena/project.yml || \\\n\
   ! grep -q "^- typescript$" /workspace/.serena/project.yml || \\\n\
   ! grep -q "^- python$" /workspace/.serena/project.yml; then\n\
  echo "Initializing Serena project with Go, TypeScript, and Python support..."\n\
  printf "languages:\\n- go\\n- typescript\\n- python\\n" > /workspace/.serena/project.yml\n\
fi\n\
\n\
# Start the Serena MCP server\n\
exec serena-mcp-server "$@"\n\
' > /usr/local/bin/serena-init.sh && chmod +x /usr/local/bin/serena-init.sh

# Create a dummy project to pre-warm Serena's LSP cache
# Focus on Python, Go, and TypeScript for codex context
RUN mkdir -p /tmp/dummy-project && \
    echo 'def hello(): pass' > /tmp/dummy-project/main.py && \
    printf 'package main\nfunc main() {}\n' > /tmp/dummy-project/main.go && \
    echo 'export function hello() {}' > /tmp/dummy-project/index.ts

# Pre-index to warm up LSP caches
RUN serena project create /tmp/dummy-project --name dummy \
    --language python --language go --language typescript \
    --index --log-level INFO --timeout 120 || \
    echo "Pre-indexing completed (some warnings may be expected)"

# Clean up
RUN rm -rf /tmp/dummy-project

# Expose the workspace directory as a volume mount point
VOLUME ["/workspace"]

# Set default entrypoint to run initialization script
ENTRYPOINT ["/usr/local/bin/serena-init.sh"]

CMD []
