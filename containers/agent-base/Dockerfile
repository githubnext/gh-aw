FROM node:20-slim

# Install necessary system dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories
RUN mkdir -p /github/workspace \
    /tmp/gh-aw/mcp-config \
    /tmp/gh-aw/aw-prompts \
    /tmp/gh-aw/logs \
    /tmp/gh-aw/safe-outputs \
    /tmp/gh-aw/.claude

# Set up environment for npm global installs
ENV NPM_CONFIG_PREFIX=/usr/local
ENV PATH="/usr/local/bin:$PATH"

# Set working directory to GitHub Actions workspace
WORKDIR /github/workspace

# Default command (will be overridden by docker-compose)
CMD ["sleep", "infinity"]
