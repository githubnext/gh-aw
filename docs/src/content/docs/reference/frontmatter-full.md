---
title: Frontmatter Full Reference
description: Complete JSON Schema-based reference for all GitHub Agentic Workflows frontmatter configuration options with YAML examples.
sidebar:
  order: 201
---

This document provides a comprehensive reference for all available frontmatter configuration options in GitHub Agentic Workflows. The examples below are generated from the JSON Schema and include inline comments describing each field.

:::note
This documentation is automatically generated from the JSON Schema. For a more user-friendly guide, see [Frontmatter](/gh-aw/reference/frontmatter/).

**Note**: Unless specified as `(required)`, all fields below are optional. Fields marked with `oneOf` support multiple format options.
:::

## Complete Frontmatter Reference

```yaml
---
# Workflow name that appears in the GitHub Actions interface
# Defaults to filename without extension if not specified
name: "My Workflow"

# Workflow description rendered as comment in .lock.yml file
description: "Description of the workflow"

# Source reference: owner/repo/path@ref (e.g., githubnext/agentics/workflows/ci-doctor.md@v1.0.0)
source: "example-value"

# Array of workflow specifications to import (format: owner/repo/path@ref)
imports: []

# Workflow triggers - when the agentic workflow should run
# Supports standard GitHub Actions events plus special command triggers
# Simple string format:
on: "push"

# Complex format with event-specific filters:
on:
  # Special command trigger for /command workflows
  # String shorthand:
  command: "customname"  # Creates /customname trigger
  # Or null to use filename as command name:
  command: null
  # Or object with configuration:
  command:
    name: "My Workflow"
    events: "*"  # or array: ["issues", "pull_request"]

  # Push event
  push:
    branches: []
    branches-ignore: []
    paths: []
    paths-ignore: []
    tags: []
    tags-ignore: []

  # Pull request event
  pull_request:
    types: []
    branches: []
    branches-ignore: []
    paths: []
    paths-ignore: []
    draft: true
    forks: "*"  # or array: ["org/*", "org/repo"]
    names: "bug"  # or array for labeled/unlabeled events

  # Issues event
  issues:
    types: []
    names: "bug"  # or array for labeled/unlabeled events

  # Other event types
  issue_comment:
    types: []
  discussion:
    types: []
  discussion_comment:
    types: []
  schedule:
    - cron: "0 0 * * *"

  # Manual workflow dispatch
  workflow_dispatch: null  # or object with inputs: {}

  workflow_run:
    workflows: []
    types: []
    branches: []
    branches-ignore: []

  release:
    types: []
  pull_request_review_comment:
    types: []
  branch_protection_rule:
    types: []
  check_run:
    types: []
  check_suite:
    types: []

  # Simple event triggers (no configuration needed)
  create: null
  delete: null
  deployment: null
  deployment_status: null
  fork: null
  gollum: null

  label:
    types: []
  merge_group:
    types: []
  milestone:
    types: []

  page_build: null
  public: null

  pull_request_target:
    types: []
    branches: []
    branches-ignore: []
    paths: []
    paths-ignore: []
    draft: true
    forks: "example-value"  # or array

  pull_request_review:
    types: []
  registry_package:
    types: []
  repository_dispatch:
    types: []
  status: null
  watch:
    types: []
  workflow_call: null

  # Workflow execution limits
  stop-after: "2025-06-01"  # or relative: "+3d", "+1d12h30m"
  reaction: "eyes"  # +1, -1, laugh, confused, heart, hooray, rocket, eyes

# GitHub token permissions - use least privilege principle
# Simple string format:
permissions: "read-all"  # or "write-all", "read", "write"

# Detailed object format:
permissions:
  actions: "read"        # read, write, none
  attestations: "read"
  checks: "read"
  contents: "read"
  deployments: "read"
  discussions: "read"
  id-token: "read"
  issues: "read"
  models: "read"        # Access AI models for agentic workflows
  packages: "read"
  pages: "read"
  pull-requests: "read"
  repository-projects: "read"
  security-events: "read"
  statuses: "read"

# Custom run name (supports GitHub expressions)
run-name: "${{ github.event.issue.title }}"

# Default settings for all jobs
defaults:
  run:
    shell: "bash"
    working-directory: "./src"

# Jobs configuration
jobs: {}

# Runner specification
# String format:
runs-on: "ubuntu-latest"
# Array format:
runs-on: ["self-hosted", "linux"]
# Object format:
runs-on:
  group: "my-group"
  labels: ["gpu", "large"]

# Workflow timeout (minutes, defaults to 15 for agentic workflows)
timeout_minutes: 30

# Concurrency control
# Simple string:
concurrency: "my-group"
# Object format:
concurrency:
  group: "my-group"
  cancel-in-progress: true

# Environment variables
env:
  NODE_ENV: "production"

# Environment configuration
# String format:
environment: "staging"
# Object format:
environment:
  name: "production"
  url: "https://example.com"

# Container configuration
# String format:
container: "node:18"
# Object format:
container:
  image: "node:18"
  credentials:
    username: "${{ secrets.USER }}"
    password: "${{ secrets.PASS }}"
  env: {}
  ports: []
  volumes: []
  options: "--cpus 2"

# Service containers
services: {}

# Network access control for AI engines
# Simple string:
network: "defaults"  # Basic infrastructure domains
# Object format:
network:
  allowed: ["python", "node", "*.example.com"]  # Ecosystem IDs or domains

# Conditional execution
if: "github.event.issue.title contains '[bug]'"

# Custom workflow steps (before AI execution)
steps: []

# Custom workflow steps (after AI execution)
post-steps: []

# AI engine configuration (defaults to 'claude')
engine: null

# Claude-specific configuration
claude:
  model: "claude-sonnet-4"
  version: "2023-06-01"
  allowed: []  # or object

# MCP server definitions
mcp-servers: {}

# Tools and MCP servers available to AI
tools:
  # GitHub API tools
  github: null  # Enable with defaults
  # or:
  github:
    allowed: ["create_issue", "add_comment"]
    mode: "local"  # or "remote"
    version: "latest"
    args: []
    read-only: true
    github-token: "${{ secrets.CUSTOM_PAT }}"
    toolset: ["issues", "pull_requests"]

  # Bash command execution
  bash: null  # Enable all commands
  # or list of allowed commands:
  bash: ["echo", "ls", "git status"]

  # Web content fetching
  web-fetch: null

  # Web search
  web-search: null

  # File editing
  edit: null

  # Playwright browser automation
  playwright: null
  # or:
  playwright:
    version: "v1.41.0"
    allowed_domains: ["github.com", "*.example.com"]
    args: []

  # Workflow introspection
  agentic-workflows: true

  # Cache memory
  cache-memory: true
  # or:
  cache-memory:
    key: "my-cache"
    description: "Persistent memory storage"
    docker-image: "mcp/memory"
    retention-days: 7
  # or array of cache configurations:
  cache-memory: []

  # Security settings
  safety-prompt: true
  timeout: 60  # seconds
  startup-timeout: 120  # seconds

# Command name for workflow (alternative to on.command)
command: "example-command"

# Cache configuration (actions/cache syntax)
# Single cache:
cache:
  key: "node-modules-${{ hashFiles('**/package-lock.json') }}"
  path: "node_modules"  # or array: ["node_modules", ".next"]
  restore-keys: "node-modules-"  # or array
  upload-chunk-size: 8388608
  fail-on-cache-miss: false
  lookup-only: false
# Multiple caches:
cache: []

# Safe output processing - create GitHub resources without write permissions
safe-outputs:
  # Domain filtering for security
  allowed-domains: ["github.com"]

  # Create issues
  create-issue: null  # Enable with defaults
  # or:
  create-issue:
    title-prefix: "[ai] "
    labels: ["automation", "ai-generated"]
    assignees: "user1"  # or array: ["user1", "copilot"]
    max: 5
    min: 0
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Create agent tasks
  create-agent-task: null
  # or:
  create-agent-task:
    base: "main"
    max: 1
    min: 0
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Create discussions
  create-discussion: null
  # or:
  create-discussion:
    title-prefix: "[AI] "
    category: "General"  # or category ID (number)
    max: 1
    min: 0
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Add comments
  add-comment: null
  # or:
  add-comment:
    max: 1
    min: 0
    target: "triggering"  # or "*" or issue number
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"
    discussion: false

  # Create pull requests
  create-pull-request: null
  # or:
  create-pull-request:
    title-prefix: "feat: "
    labels: ["automated"]
    reviewers: "copilot"  # or array: ["user1", "copilot"]
    draft: true
    if-no-changes: "warn"  # or "error", "ignore"
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # PR review comments
  create-pull-request-review-comment: null
  # or:
  create-pull-request-review-comment:
    max: 1
    min: 0
    side: "RIGHT"  # or "LEFT"
    target: "triggering"  # or "*" or PR number
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Code scanning alerts (SARIF)
  create-code-scanning-alert: null
  # or:
  create-code-scanning-alert:
    max: 10
    min: 0
    driver: "Custom Security Scanner"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Add labels
  add-labels: null  # Allow any labels
  # or:
  add-labels:
    allowed: ["bug", "enhancement"]
    max: 3
    min: 0
    target: "triggering"  # or "*" or issue number
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Update issues
  update-issue: null
  # or:
  update-issue:
    status: null  # Allow status updates
    title: null   # Allow title updates
    body: null    # Allow body updates
    max: 1
    min: 0
    target: "triggering"  # or "*" or issue number
    target-repo: "owner/repo"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Push to PR branch
  push-to-pull-request-branch: null
  # or:
  push-to-pull-request-branch:
    branch: "main"
    target: "triggering"  # or "*" or PR number
    title-prefix: "feat: "
    labels: ["automated"]
    if-no-changes: "warn"
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Missing tool reporting
  missing-tool: null  # Enable
  # or false to disable:
  missing-tool: false
  # or:
  missing-tool:
    max: 10
    min: 0
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Upload assets
  upload-assets: null
  # or:
  upload-assets:
    branch: "assets/${{ github.workflow }}"
    max-size: 10240  # KB
    allowed-exts: [".png", ".jpg", ".pdf"]
    max: 10
    min: 0
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Safe outputs settings
  staged: false  # true for preview mode
  env: {}
  github-token: "${{ secrets.GITHUB_TOKEN }}"
  max-patch-size: 1024  # KB

  # Threat detection
  threat-detection: true
  # or:
  threat-detection:
    enabled: true
    prompt: "Additional security instructions"
    engine: null
    steps: []

  # Custom safe-output jobs
  jobs: {}

  # Runner for safe-output jobs
  runs-on: "ubuntu-latest"

# Repository access roles (defaults to ['admin', 'maintainer'])
roles: "all"  # Allow any authenticated user (⚠️ use with caution)
# or array:
roles: ["admin", "maintainer", "write", "triage"]

# Strict mode validation
strict: false

# Runtime environment version overrides
runtimes: {}

# GitHub token for all authenticated steps
# Defaults to: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
github-token: "${{ secrets.GITHUB_TOKEN }}"
---
```

## Additional Information

- Unless marked `(required)`, all fields are optional
- Fields with `oneOf` annotations support multiple format options
- See the [Frontmatter guide](/gh-aw/reference/frontmatter/) for detailed explanations and examples
- See individual reference pages: [Triggers](/gh-aw/reference/triggers/), [Tools](/gh-aw/reference/tools/), [Safe Outputs](/gh-aw/reference/safe-outputs/)
