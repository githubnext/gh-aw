# Pre-Download and Installation Strategies

This guide covers strategies for pre-downloading dependencies, installing tools, and preparing the workflow environment before the agent executes.

## Overview

Agentic workflows often require external tools, libraries, or data that need to be available before the AI agent runs. Use the `steps:` section to prepare the environment.

## Core Concepts

### Steps vs Agent Execution

**Steps** (runs before agent):
```yaml
steps:
  - name: Install dependencies
    run: npm install
  - name: Setup tools
    run: pip install -r requirements.txt
```

**Agent job** (runs after steps complete):
- AI agent executes with access to installed tools
- Can use tools installed in steps
- Operates in the prepared environment

## Common Installation Patterns

### 1. Browser Automation (Playwright)

**Required for**: Web scraping, browser automation, visual testing

```yaml
tools:
  playwright:
    allowed_domains: ["example.com"]

steps:
  - name: Install Playwright browsers
    run: |
      npm install -g playwright
      npx playwright install chromium
      # Or for specific browsers:
      # npx playwright install chromium firefox webkit
```

**Why pre-install**:
- Browser binaries are large (~100-300 MB)
- Installation takes time (1-2 minutes)
- Agent can use immediately without delays

### 2. Media Processing Tools

**Required for**: Video/audio/image manipulation

```yaml
steps:
  - name: Install FFmpeg
    run: |
      sudo apt-get update
      sudo apt-get install -y ffmpeg
      ffmpeg -version  # Verify installation
```

**Why pre-install**:
- System packages require root access
- Large binaries
- Dependencies need to be resolved

**Access in agent**:
```yaml
tools:
  bash: ["ffmpeg"]
```

### 3. Code Analysis Tools

**Required for**: AST analysis, linting, code quality checks

```yaml
steps:
  - name: Install analysis tools
    run: |
      # AST-grep for pattern matching
      npm install -g @ast-grep/cli
      
      # Or CodeQL for security analysis
      pip install codeql
      
      # Or language-specific tools
      go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Access in agent**:
```yaml
tools:
  bash: ["ast-grep", "codeql", "golangci-lint"]
```

### 4. Language Runtimes

**Required for**: Running scripts, custom tools

```yaml
steps:
  - name: Setup Node.js
    uses: actions/setup-node@v4
    with:
      node-version: '20'
  
  - name: Setup Python
    uses: actions/setup-python@v5
    with:
      python-version: '3.11'
  
  - name: Setup Go
    uses: actions/setup-go@v5
    with:
      go-version: '1.21'
```

**Alternative: Runtime version overrides**:
```yaml
runtimes:
  node:
    version: "20"
  python:
    version: "3.11"
  go:
    version: "1.21"
```

### 5. Project Dependencies

**Required for**: Running project code, tests, builds

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
  
  - name: Install npm dependencies
    run: npm install
  
  - name: Install Python dependencies
    run: pip install -r requirements.txt
  
  - name: Install Go dependencies
    run: go mod download
```

### 6. MCP Server Dependencies

**Required for**: Custom MCP servers

```yaml
steps:
  - name: Install MCP server
    run: |
      npm install -g mcp-server-package
      # Or build from source
      cd mcp-servers/custom
      npm install
      npm run build

mcp-servers:
  custom:
    command: "node"
    args: ["./mcp-servers/custom/dist/index.js"]
    allowed:
      - custom_tool
```

## Pre-Download Strategies

### 1. Data Files

**Use case**: Download datasets, reference files, configuration

```yaml
steps:
  - name: Download reference data
    run: |
      curl -L https://example.com/data.json -o /tmp/reference-data.json
      # Verify download
      jq . /tmp/reference-data.json > /dev/null
```

**Access in agent**:
```markdown
The reference data is available at `/tmp/reference-data.json`.
Use `jq` to query it as needed.
```

### 2. Git Repositories

**Use case**: Clone additional repositories for analysis

```yaml
steps:
  - name: Clone reference repository
    run: |
      git clone https://github.com/org/reference-repo /tmp/reference-repo
      cd /tmp/reference-repo
      git log --oneline -10
```

### 3. Docker Images

**Use case**: Pre-pull Docker images for containerized tools

```yaml
steps:
  - name: Pull Docker images
    run: |
      docker pull image:tag
      docker images
```

### 4. Model Files

**Use case**: Download AI models, embeddings, trained data

```yaml
steps:
  - name: Download model
    run: |
      mkdir -p /tmp/models
      curl -L https://huggingface.co/model/files/model.bin -o /tmp/models/model.bin
      ls -lh /tmp/models/
```

## Verification Patterns

### Verify Installation

```yaml
steps:
  - name: Install and verify tool
    run: |
      npm install -g tool-name
      tool-name --version
      which tool-name
```

### Test Functionality

```yaml
steps:
  - name: Test Playwright
    run: |
      npx playwright install chromium
      node -e "const { chromium } = require('playwright'); (async () => { const browser = await chromium.launch(); await browser.close(); })()"
```

### Check Dependencies

```yaml
steps:
  - name: Verify dependencies
    run: |
      npm install
      npm run test  # Run project tests
```

## Performance Optimization

### 1. Caching Dependencies

```yaml
steps:
  - name: Cache npm dependencies
    uses: actions/cache@v4
    with:
      path: ~/.npm
      key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
  
  - name: Install dependencies
    run: npm install
```

### 2. Parallel Installation

```yaml
steps:
  - name: Install tools in parallel
    run: |
      npm install -g tool1 &
      pip install package1 &
      apt-get install -y package1 &
      wait  # Wait for all background jobs
```

### 3. Minimal Installation

Install only what's needed:

```yaml
steps:
  - name: Install minimal Playwright
    run: |
      # Only Chromium, not all browsers
      npx playwright install chromium
  
  - name: Install minimal Python packages
    run: |
      # Only required packages, no dev dependencies
      pip install --no-dev -r requirements.txt
```

## Error Handling

### Graceful Failure

```yaml
steps:
  - name: Install optional tool
    run: |
      npm install -g optional-tool || echo "Optional tool installation failed, continuing..."
    continue-on-error: true
```

### Retry Logic

```yaml
steps:
  - name: Install with retry
    run: |
      for i in {1..3}; do
        npm install -g tool && break
        echo "Attempt $i failed, retrying..."
        sleep 5
      done
```

### Validation

```yaml
steps:
  - name: Install and validate
    run: |
      npm install -g tool
      if ! command -v tool &> /dev/null; then
        echo "Tool installation failed"
        exit 1
      fi
      echo "Tool installed successfully"
```

## Common Patterns by Use Case

### Web Automation Workflow

```yaml
tools:
  playwright:
    allowed_domains: ["example.com"]

steps:
  - name: Setup browser automation
    run: |
      npm install -g playwright
      npx playwright install chromium
      npx playwright install-deps  # System dependencies
```

### Code Analysis Workflow

```yaml
tools:
  serena: ["typescript"]

steps:
  - name: Checkout code
    uses: actions/checkout@v4
  
  - name: Install dependencies
    run: npm install
  
  - name: Install analysis tools
    run: |
      npm install -g @ast-grep/cli
      npm install -g eslint
```

### Data Processing Workflow

```yaml
steps:
  - name: Setup Python environment
    uses: actions/setup-python@v5
    with:
      python-version: '3.11'
  
  - name: Install data tools
    run: |
      pip install pandas numpy matplotlib
      pip install jupyter  # For notebook processing
  
  - name: Download dataset
    run: |
      curl -L https://example.com/dataset.csv -o /tmp/data.csv
```

### Multi-Tool Workflow

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
  
  - name: Setup Node.js
    uses: actions/setup-node@v4
    with:
      node-version: '20'
  
  - name: Setup Python
    uses: actions/setup-python@v5
    with:
      python-version: '3.11'
  
  - name: Install all dependencies
    run: |
      npm install
      pip install -r requirements.txt
      npm install -g @ast-grep/cli
      npx playwright install chromium
```

## Summary

- Use `steps:` to prepare the environment before the agent runs
- Pre-install large binaries and dependencies (Playwright, FFmpeg, etc.)
- Verify installations to catch errors early
- Cache dependencies for faster runs
- Install only what's needed for the workflow
- Use GitHub Actions setup actions for language runtimes
- Pre-download data files and reference materials
- Test functionality after installation
- Handle errors gracefully with retries and validation
