# This Dockerfile is used to create a pre-built devcontainer image
# Start from the basic devcontainer with features pre-installed
FROM mcr.microsoft.com/devcontainers/go:1-bookworm

# Install additional system packages that might be needed
RUN apt-get update && \
    apt-get install -y git curl build-essential && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set up the workspace directory
WORKDIR /workspace

# Copy dependency files for better Docker layer caching
COPY go.mod go.sum package.json package-lock.json ./

# Pre-install Go dependencies
RUN go mod download && \
    go install golang.org/x/tools/gopls@latest && \
    go install github.com/rhymd/actionlint/cmd/actionlint@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Pre-install npm dependencies if package.json exists
RUN if [ -f "package.json" ]; then npm ci; fi

# Download GitHub Actions workflow schema
RUN mkdir -p pkg/workflow/schemas && \
    curl -s -o pkg/workflow/schemas/github-workflow.json \
        "https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json" || true

# Set up Go environment paths
ENV GOPATH="/go"
ENV PATH="${PATH}:/usr/local/go/bin:/go/bin"

# Create .devcontainer directory and copy setup script
RUN mkdir -p .devcontainer
COPY .devcontainer/setup.sh .devcontainer/
RUN chmod +x .devcontainer/setup.sh

# Set default working directory
WORKDIR /workspace