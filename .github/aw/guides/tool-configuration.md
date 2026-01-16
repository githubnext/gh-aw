# Tool Configuration Reference

This guide provides quick reference for configuring tools in agentic workflows.

## Built-in Tools

### GitHub Tool

**Always use `toolsets:` for GitHub tools** - Use `toolsets: [default]` instead of manually listing individual tools.

```yaml
tools:
  github:
    toolsets: [default]
```

**Available toolsets**:
- `default`: Common read operations (repos, issues, pull_requests, search, etc.)
- `repos`: Repository operations
- `issues`: Issue operations
- `pull_requests`: Pull request operations
- `search`: Search operations
- `actions`: GitHub Actions operations

**IMPORTANT**:
- ⚠️ **Never recommend GitHub mutation tools** like `create_issue`, `add_issue_comment`, `update_issue`, etc.
- ✅ **Always use `safe-outputs` instead** for any GitHub write operations
- ⚠️ **Do NOT recommend `mode: remote`** for GitHub tools - it requires additional configuration. Use `mode: local` (default) instead.

**Example with safe outputs**:
```yaml
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-comment:
    max: 1
  create-issue:
    enabled: true
```

### Serena (Language Server)

Serena provides language server capabilities for code analysis. Detect the repository's primary programming language and specify it in the array.

```yaml
tools:
  serena: ["go"]  # Update with your programming language
```

**Supported languages**: `go`, `typescript`, `python`, `ruby`, `rust`, `java`, `cpp`, `csharp`, and many more (see `.serena/project.yml` for full list).

**How to detect language**:
- Check file extensions in the repository
- Look for language-specific files: `go.mod`, `package.json`, `requirements.txt`, `Gemfile`, etc.
- Choose the primary language used in the codebase

### Playwright (Browser Automation)

```yaml
tools:
  playwright:
    version: "v1.41.0"  # Optional: specify version
    allowed_domains:
      - "github.com"
      - "example.com"
network:
  allowed:
    - "github.com"
    - "example.com"
```

**Use cases**:
- Browser automation
- Web scraping
- Visual testing
- Accessibility analysis

**Installation required**: Add installation step before agent job:
```yaml
steps:
  - name: Install Playwright
    run: npx playwright install chromium
```

### Web Fetch

```yaml
tools:
  web-fetch: {}
network:
  allowed:
    - "api.example.com"
    - "example.com"
```

**Use cases**:
- Fetch web pages as markdown
- API requests
- Download files

### Web Search

```yaml
tools:
  web-search: {}
network:
  allowed:
    - "*"  # Web search requires broad access
```

**Use cases**:
- Research tasks
- Finding information
- Discovering resources

### Bash

**Default behavior**: 
- ✅ **`bash` is enabled by default** when sandboxing is active
- ✅ Defaults to `*` (all commands) in sandboxed environments
- Only specify `bash:` if you need to restrict beyond secure defaults

**Custom restrictions** (only if needed):
```yaml
tools:
  bash:
    - "git"
    - "npm"
    - "python"
```

**Common command-line tools available via bash**:
- `jq` - JSON processing
- `git` - Version control
- `curl` - HTTP requests (with network permissions)
- Language runtimes: `node`, `python`, `go`, etc.

### Edit

**Default behavior**:
- ✅ **`edit` is enabled by default** when sandboxing is active
- No explicit configuration needed in most cases

## Tool Categories by Use Case

### API Integration
```yaml
tools:
  github:
    toolsets: [default]
  web-fetch: {}
  bash: ["jq"]  # For JSON processing
network:
  allowed:
    - "api.example.com"
```

### Browser Automation
```yaml
tools:
  playwright:
    allowed_domains: ["example.com"]
network:
  allowed:
    - "example.com"
```

### Media Manipulation
```yaml
# No built-in tool - install via steps
steps:
  - name: Install FFmpeg
    run: sudo apt-get update && sudo apt-get install -y ffmpeg
```

Then use via bash:
```yaml
tools:
  bash: ["ffmpeg"]
```

### Code Analysis
```yaml
tools:
  serena: ["typescript"]  # Language server
  bash: ["ast-grep", "codeql"]  # CLI tools
```

**Installation for CLI tools**:
```yaml
steps:
  - name: Install analysis tools
    run: |
      npm install -g @ast-grep/cli
      # or: pip install codeql
```

## Default Tools (No Configuration Needed)

When sandboxing is active (via `sandbox.agent` or network restrictions), these tools are enabled by default:

- ✅ **`edit`** - File editing operations
- ✅ **`bash`** - Command execution (all commands by default)

You only need to specify these tools if you want to add restrictions beyond the defaults.

## Safe Outputs (Write Operations)

For GitHub write operations, always use safe outputs instead of mutation tools:

```yaml
safe-outputs:
  # Issue operations
  create-issue:
    enabled: true
  add-comment:
    max: 1  # Limit number of comments
  update-issue:
    enabled: true
  
  # Pull request operations
  create-pull-request:
    enabled: true
  create-pull-request-review-comment:
    max: 5
  
  # Daily reporting workflows
  create-issue:
    close-older-issues: true  # Prevent clutter
  create-discussion:
    close-older-discussions: true
```

### Missing Tool Tracking

For new workflows, enable automatic missing tool tracking:

```yaml
safe-outputs:
  missing-tool:
    create-issue: true  # Auto-create issues for missing tools (expire after 1 week)
```

### Custom Safe Output Jobs

For write operations to external services (email, Slack, webhooks), use custom safe output jobs:

```yaml
safe-outputs:
  jobs:
    notify-slack:
      steps:
        - name: Send Slack notification
          uses: slackapi/slack-github-action@v1
          with:
            payload: |
              {
                "text": "${{ needs.agent.outputs.summary }}"
              }
          env:
            SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

**DO NOT use `post-steps:` for custom write operations** - `post-steps:` are for cleanup/logging only.

## Tool Installation Patterns

### NPM Package
```yaml
steps:
  - name: Install tool
    run: npm install -g package-name
```

### Python Package
```yaml
steps:
  - name: Install tool
    run: pip install package-name
```

### System Package
```yaml
steps:
  - name: Install tool
    run: sudo apt-get update && sudo apt-get install -y package-name
```

## Network Configuration

When using tools that access external resources:

```yaml
network:
  allowed:
    - "api.github.com"  # Specific domain
    - "*.example.com"   # Wildcard subdomain
    - "*"               # All domains (use sparingly)
  
  # Or restrict by ecosystem
  ecosystems:
    - "node"    # NPM registry
    - "python"  # PyPI
    - "go"      # Go modules
```

## Common Patterns

### Read-Only GitHub Integration
```yaml
tools:
  github:
    toolsets: [default]
```

### GitHub with Write Operations
```yaml
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-comment:
    max: 1
```

### Web Research
```yaml
tools:
  web-search: {}
  web-fetch: {}
network:
  allowed:
    - "*"
```

### Code Analysis with Language Server
```yaml
tools:
  serena: ["typescript", "go"]
  github:
    toolsets: [default]
```

### Multi-Tool Integration
```yaml
tools:
  github:
    toolsets: [default]
  web-fetch: {}
  serena: ["python"]
network:
  allowed:
    - "api.example.com"
safe-outputs:
  create-issue:
    enabled: true
```

## Summary

- **GitHub**: Use `toolsets: [default]` and safe outputs for writes
- **Serena**: Detect and specify repository language(s)
- **Playwright**: Requires installation and network permissions
- **Bash & Edit**: Enabled by default in sandboxed mode
- **Safe Outputs**: Always use instead of mutation tools
- **Network**: Explicitly allow domains/ecosystems for external access
- **Installation**: Use workflow steps to install required tools
