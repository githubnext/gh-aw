---
description: Security smoke test workflow that validates firewall, strict mode, and safe-inputs functionality
on: 
  schedule:
    - cron: "0 3,15 * * *"  # Twice daily, offset from core tests
  workflow_dispatch:
    inputs:
      variant:
        description: 'Security variant to test (all, firewall, no-firewall, safe-inputs)'
        required: false
        default: 'all'
        type: choice
        options:
          - all
          - firewall
          - no-firewall
          - safe-inputs
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Security
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
sandbox:
  agent: awf
imports:
  - shared/gh.md
tools:
  edit:
  bash:
    - "*"
  github:
safe-outputs:
  add-comment:
    hide-older-comments: true
  create-issue:
    expires: 1d
  add-labels:
    allowed: [smoke-security, smoke-firewall, smoke-no-firewall, smoke-safe-inputs]
  messages:
    footer: "> 🛡️ *Security tested by [{workflow_name}]({run_url})*"
    run-started: "🛡️ SECURITY CHECK: [{workflow_name}]({run_url}) validating security features for {event_type}..."
    run-success: "✅ SECURITY: [{workflow_name}]({run_url}) PASSED. Security features validated. 🔒"
    run-failure: "❌ SECURITY: [{workflow_name}]({run_url}) {status}. Security validation issues detected..."
timeout-minutes: 10
strict: true
---

# Smoke Test: Security Variants Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This workflow validates security-related configuration variants including firewall settings, strict mode, and safe-inputs functionality.

## Variant Selection

Current configuration: **firewall enabled, strict mode on, GitHub MCP enabled** (default)

This workflow uses a secure configuration by default (firewall enabled, strict mode). When triggered via `workflow_dispatch` with the `variant` input parameter, the AI agent will be instructed to validate the specified security variant:
- `all` - Validate all security variants (firewall, no-firewall, safe-inputs)
- `firewall` - Validate firewall-enabled configuration
- `no-firewall` - Validate configuration without firewall
- `safe-inputs` - Validate safe-inputs as alternative to GitHub MCP

**Note:** The workflow configuration uses default secure settings. The `variant` input controls which security scenarios to validate and report on.

## Security Test Requirements

### Firewall Variant Tests

When testing **firewall** variant:

1. **Firewall Validation**: Attempt to access a domain NOT in the allowed list (e.g., example.com) using curl - this should fail or be blocked
2. **GitHub MCP Access**: Verify GitHub MCP server works through the firewall
3. **Strict Mode**: Confirm strict mode is enabled and enforcing constraints
4. **Network Sandboxing**: Test that only allowed network ecosystems (defaults, node, github) are accessible

### No-Firewall Variant Tests

When testing **no-firewall** variant:

1. **Firewall Status**: Verify that firewall is disabled (sandbox.agent: false)
2. **GitHub MCP Access**: Verify GitHub MCP server still works without firewall
3. **Strict Mode Status**: Confirm strict mode is disabled (strict: false)
4. **Network Access**: Test broader network access is available

### Safe-Inputs Variant Tests

When testing **safe-inputs** variant:

1. **GitHub MCP Disabled**: Confirm GitHub MCP is intentionally disabled (github: false)
2. **Safe-Inputs gh Tool**: Use the `safeinputs-gh` tool to run "gh pr list --state merged --limit 2" to verify the tool can access GitHub data
3. **CLI Alternative**: Verify that safe-inputs provides an alternative way to access GitHub without MCP
4. **File Operations**: Confirm basic file operations still work

### Common Tests (All Variants)

1. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-security-${{ github.run_id }}.txt` with content "Security variant test passed at $(date)"
2. **Bash Tool Testing**: Execute bash commands to verify file creation was successful

## Multi-Variant Testing

If the `variant` input is set to `all` (or triggered without specifying a variant), validate all security configurations and report findings:

1. **Firewall Variant** - Validate firewall blocking and strict mode enforcement
2. **No-Firewall Variant** - Note: This workflow runs with firewall enabled; report conceptual differences
3. **Safe-Inputs Variant** - Test safe-inputs as GitHub MCP alternative

For each variant validation, report results separately.

## Output

Add a **very brief** comment (max 10-15 lines) to the current pull request with:

### Format:
```
## Smoke Security Test Results - Run ${{ github.run_id }}

**Variant(s) Tested:** [variant list]

| Variant | Firewall | Strict Mode | Result |
|---------|----------|-------------|--------|
| firewall | ✅ Enabled | ✅ On | ✅/❌ |
| no-firewall | ❌ Disabled | ❌ Off | ✅/❌ |
| safe-inputs | ✅ Enabled | ✅ On | ✅/❌ |

**Overall Status:** PASS/FAIL

**Key Findings:**
- Firewall blocking: [status]
- Safe-inputs working: [status]
```

If all tests pass, add the label `smoke-security` to the pull request. For specific variants, also add variant-specific labels (`smoke-firewall`, `smoke-no-firewall`, or `smoke-safe-inputs`).
