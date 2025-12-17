---
name: Secure Workflow Template
description: Template demonstrating security best practices for GitHub Agentic Workflows

# SECURITY: Event Triggers
# Choose appropriate triggers for your workflow
# Risky triggers (slash_command, reaction, issue_comment) require careful permission management
on:
  slash_command:
    name: example-command
    events: [issues, issue_comment]
  # Alternative safe triggers:
  # workflow_dispatch:  # Manual trigger (safe)
  # schedule:           # Cron-based trigger (safe)
  #   - cron: "0 9 * * 1"

# SECURITY: Explicit Permissions (Principle of Least Privilege)
# Only grant the minimum permissions necessary for your workflow
# Unspecified permissions default to 'none' - this is intentional
# Use read-only permissions by default; use safe-outputs for write operations
permissions:
  contents: read        # Read repository contents
  issues: read          # Read issues
  pull-requests: read   # Read pull requests
  # Never use 'write' permissions with risky triggers unless absolutely necessary
  # Write operations should use safe-outputs instead

# SECURITY: Strict Mode (Recommended for Production)
# Enforces additional security constraints:
# - Blocks write permissions (use safe-outputs instead)
# - Requires explicit network configuration
# - Refuses wildcard (*) in network domains
# - Enforces Action pinning to commit SHAs
strict: true

# SECURITY: AI Engine Configuration
engine:
  id: copilot
  model: gpt-5-mini  # or gpt-4o for more complex tasks

# SECURITY: Network Isolation
# Explicitly define which domains the AI engine can access
# Start with 'defaults' and add only necessary ecosystems/domains
network:
  allowed:
    - defaults  # Basic infrastructure (GitHub, PyPI, npm, Docker Hub, etc.)
    # Add specific ecosystems as needed:
    # - python    # Python package ecosystem
    # - node      # Node.js package ecosystem
    # - containers # Container registries

# SECURITY: Tool Configuration
# Use explicit allow-lists for tools - never use wildcards ["*"]
tools:
  github:
    mode: local           # or 'remote' for hosted MCP server
    read-only: true       # Prevent accidental write operations
    toolsets:
      - default           # Basic GitHub operations
      # Other toolsets: repos, issues, pull_requests, discussions
  
  # SECURITY: Bash commands require explicit allowlisting
  # bash:
  #   - "echo"
  #   - "git status"
  #   - "cat"
  # Never use: bash: ["*"]  # This is extremely dangerous
  
  # SECURITY: Edit tool for file modifications (use with caution)
  # edit:  # Enable only if file editing is required

# SECURITY: Safe Outputs (Write Operations)
# Separate AI processing (read-only) from write operations
# The AI never has direct write access - all writes are validated
safe-outputs:
  add-comment:
    max: 1  # Limit number of comments
  
  # SECURITY: Threat Detection (Recommended)
  # Automatically scans agent output for prompt injection, secrets, malicious patches
  threat-detection:
    enabled: true
    # Optional: Add custom scanning focus
    # prompt: "Focus on SQL injection patterns"

  # SECURITY: Custom messages for user transparency
  messages:
    footer: "> 🤖 *Automated response from [{workflow_name}]({run_url})*"
    run-started: "🚀 [{workflow_name}]({run_url}) is processing this {event_type}..."
    run-success: "✅ [{workflow_name}]({run_url}) completed successfully!"
    run-failure: "❌ [{workflow_name}]({run_url}) {status}. Please check the logs."

# SECURITY: Timeout Limits
# Prevent runaway workflows and excessive costs
timeout-minutes: 10

# SECURITY: Role-Based Access Control (Optional)
# Default: [admin, maintainer, write]
# Restrict to specific roles for sensitive operations:
# roles: [admin, maintainer]
# Use 'roles: all' with extreme caution in public repositories

# SECURITY: Workflow Expiration (Optional for scheduled workflows)
# Set expiration date for time-limited workflows
# stop-after: "+30d"  # Expires 30 days after compilation

---

# Secure Workflow Template

## Overview

This workflow demonstrates security best practices for GitHub Agentic Workflows. It serves as a starting point for creating new workflows with security built-in from the start.

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: @${{ github.actor }}
- **Content**: "${{ needs.activation.outputs.text }}"

## Security Features Implemented

### 1. Explicit Permissions (Least Privilege)
- Read-only permissions for repository access
- No write permissions in the AI processing job
- Write operations handled through validated safe-outputs

### 2. Strict Mode Enforcement
- Blocks write permissions on the main job
- Requires explicit network configuration
- Enforces Action pinning to commit SHAs
- Refuses wildcard network domains

### 3. Sanitized Context Usage
**CRITICAL**: Always use `${{ needs.activation.outputs.text }}` instead of raw `github.event` fields to prevent:
- Prompt injection attacks
- @mention abuse
- Bot triggers
- XML/HTML injection

The sanitized output provides:
- Neutralized @mentions
- Safe XML format
- HTTPS URIs from trusted domains only
- Content limits (0.5MB/65k lines)
- Removed control characters

### 4. Network Isolation
- Domain allowlisting for AI engine
- Starts with 'defaults' (essential infrastructure)
- Adds specific ecosystems only as needed
- No wildcard domains in strict mode

### 5. Tool Hardening
- Explicit tool allow-lists (no wildcards)
- Read-only GitHub access
- Bash commands require explicit allowlisting
- File editing disabled by default

### 6. Safe Outputs Security Model
- Separates AI processing from write operations
- AI never has direct write access
- All write operations are validated
- Automatic output sanitization

### 7. Threat Detection
- Automatic scanning of agent output
- Detects prompt injection attempts
- Identifies secret leaks
- Scans for malicious code patterns

### 8. Resource Limits
- Timeout limits prevent runaway workflows
- Comment limits prevent spam
- Role-based access control
- Optional workflow expiration

## Task Instructions

[Replace this section with your workflow's specific instructions]

### Step 1: Analyze Context
Review the triggering content and understand the request.

### Step 2: Process Request
Perform the necessary analysis or operations using the available tools.

### Step 3: Generate Response
Create a helpful response using the safe-output mechanisms.

## Important Security Notes

### DO ✅
- Use sanitized context (`needs.activation.outputs.text`)
- Declare explicit permissions
- Use safe-outputs for write operations
- Enable strict mode for production workflows
- Implement threat detection
- Set appropriate timeout limits
- Use tool allow-lists
- Pin Actions to commit SHAs (in strict mode)

### DON'T ❌
- Use raw `github.event` fields in prompts
- Grant write permissions to the AI processing job
- Use wildcard tool configurations `["*"]`
- Allow unrestricted network access
- Skip threat detection for workflows handling external input
- Use `roles: all` in public repositories without careful consideration

## Additional Resources

For more information on security best practices:
- [Security Best Practices Guide](/gh-aw/guides/security/)
- [Threat Detection Guide](/gh-aw/guides/threat-detection/)
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Network Configuration](/gh-aw/reference/network/)
- [Strict Mode Documentation](/gh-aw/reference/frontmatter/#strict-mode-strict)

## Customization Guide

To adapt this template for your use case:

1. **Update Triggers**: Choose appropriate event triggers for your workflow
2. **Adjust Permissions**: Add only the minimum permissions needed
3. **Configure Tools**: Enable only the tools your workflow requires
4. **Set Network Access**: Add specific ecosystems or domains as needed
5. **Customize Safe Outputs**: Configure the specific GitHub API operations needed
6. **Add Task Instructions**: Replace the placeholder instructions with your workflow logic
7. **Test Thoroughly**: Compile and test with `gh aw compile --strict --zizmor`

## Testing

Before deploying this workflow:

```bash
# Compile with security checks
gh aw compile --strict --zizmor secure-workflow.md

# Test in a safe environment
gh workflow run secure-workflow.lock.yml
```

## Support

If you need help securing your workflow or have security questions:
- Review the [Security Best Practices Guide](/gh-aw/guides/security/)
- Open an issue in the repository
- Join the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)
