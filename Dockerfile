# Dockerfile for GitHub Agentic Workflows compiler
# Provides a minimal container with gh-aw, gh CLI, git, and jq

# Use Alpine 3.19 for minimal size (3.20+ removed gh CLI due to Python 3.12 compatibility)
FROM alpine:3.19

# Install required dependencies
RUN apk update && apk add --no-cache \
    git \
    jq \
    bash \
    curl \
    ca-certificates \
    gh

# Docker Buildx automatically provides these ARGs for multi-platform builds
# Expected values: TARGETOS=linux, TARGETARCH=amd64|arm64
# For local builds without buildx, these must be provided explicitly:
#   docker build --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 ...
# Default to linux/amd64 if not provided
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Create a directory for the binary
WORKDIR /usr/local/bin

# Copy the appropriate binary based on target platform
# TARGETOS=linux, TARGETARCH=amd64 -> dist/linux-amd64
# TARGETOS=linux, TARGETARCH=arm64 -> dist/linux-arm64
COPY dist/${TARGETOS}-${TARGETARCH} /usr/local/bin/gh-aw

# Ensure the binary is executable and verify it exists
RUN chmod +x /usr/local/bin/gh-aw && \
    /usr/local/bin/gh-aw --version || \
    (echo "Error: gh-aw binary not found or not executable" && exit 1)

# Configure git to trust all directories to avoid "dubious ownership" errors
# This is necessary when the container runs with mounted volumes owned by different users
RUN git config --global --add safe.directory '*'

# Set working directory for users
WORKDIR /workspace

# Set the entrypoint to gh-aw
ENTRYPOINT ["gh-aw"]

# Default command runs MCP server with actor validation enabled
# The GITHUB_ACTOR environment variable must be set for logs and audit tools to be available
# Binary path detection is automatic via os.Executable()
CMD ["mcp-server", "--validate-actor"]

# Metadata labels
LABEL org.opencontainers.image.source="https://github.com/github/gh-aw"
LABEL org.opencontainers.image.description="GitHub Agentic Workflows - Write agentic workflows in natural language markdown"
LABEL org.opencontainers.image.licenses="MIT"
