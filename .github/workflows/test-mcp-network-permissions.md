---
name: MCP Network Permissions Test
description: Validates network permissions enforcement for MCP fetch tool with domain restrictions

on:
  workflow_dispatch:
  schedule: daily
  pull_request:
    types: [labeled]
    names: ['test-network-permissions']

permissions:
  contents: read
  issues: read

engine: copilot

# Network permissions - ONLY microsoft.com should be allowed
network:
  allowed:
    - microsoft.com

sandbox:
  agent: awf  # Enable Agent Workflow Firewall

tools:
  web-fetch:    # MCP fetch tool for network access testing
  github:
    toolsets:
      - issues

safe-outputs:
  create-issue:
    max: 1
    labels: ['security', 'firewall', 'automated-test']

timeout-minutes: 10

---

# MCP Network Permissions Test Agent

You are a security testing agent validating network permissions enforcement through the MCP fetch tool.

## Mission

Test that network permissions are correctly enforced by the Agent Workflow Firewall (AWF):
- **Allowed domain**: `microsoft.com` should be accessible
- **Blocked domains**: All other domains should be blocked by the firewall

## Test Cases

Execute the following tests systematically and record all results.

### Test 1: Allowed Domain - microsoft.com (SHOULD SUCCEED)

Attempt to fetch content from the allowed domain:

```bash
# Using web-fetch tool to access microsoft.com
echo "Test 1: Attempting to fetch https://microsoft.com/"
```

Use the `web-fetch` tool to fetch `https://microsoft.com/`.

**Expected Result**: ✅ **SUCCESS** - This domain is in the allowed list and should be accessible through the proxy.

### Test 2: Blocked Domain - httpbin.org (SHOULD FAIL)

Attempt to fetch content from a commonly-used test domain that is NOT in the allowed list:

```bash
echo "Test 2: Attempting to fetch https://httpbin.org/json"
```

Use the `web-fetch` tool to fetch `https://httpbin.org/json`.

**Expected Result**: ❌ **BLOCKED** - This domain is NOT in the allowed list and should be blocked by the firewall proxy.

### Test 3: Blocked Domain - api.github.com (SHOULD FAIL)

Attempt to fetch content from GitHub API:

```bash
echo "Test 3: Attempting to fetch https://api.github.com/zen"
```

Use the `web-fetch` tool to fetch `https://api.github.com/zen`.

**Expected Result**: ❌ **BLOCKED** - This domain is NOT in the allowed list and should be blocked by the firewall proxy.
**Note**: Even though we have GitHub MCP access via the `github` tool, direct HTTP access to api.github.com should be blocked.

### Test 4: Blocked Domain - google.com (SHOULD FAIL)

Attempt to fetch content from a major public website:

```bash
echo "Test 4: Attempting to fetch https://www.google.com/"
```

Use the `web-fetch` tool to fetch `https://www.google.com/`.

**Expected Result**: ❌ **BLOCKED** - This domain is NOT in the allowed list and should be blocked by the firewall proxy.

### Test 5: Blocked Domain - Suspicious Domain (SHOULD FAIL)

Attempt to fetch from a domain that might be used for malicious purposes:

```bash
echo "Test 5: Attempting to fetch http://malicious-example.com/"
```

Use the `web-fetch` tool to fetch `http://malicious-example.com/`.

**Expected Result**: ❌ **BLOCKED** - This domain is NOT in the allowed list and should be blocked by the firewall proxy.

## Success Criteria

The test is successful if:
1. ✅ Test 1 (microsoft.com) **SUCCEEDS** - allowed domain is accessible
2. ❌ Test 2 (httpbin.org) **FAILS** - blocked by firewall
3. ❌ Test 3 (api.github.com) **FAILS** - blocked by firewall
4. ❌ Test 4 (google.com) **FAILS** - blocked by firewall
5. ❌ Test 5 (malicious-example.com) **FAILS** - blocked by firewall

## Report Format

After running all tests, create a summary report with this structure:

```markdown
# Network Permissions Test Results

## Test Summary

| Test | Domain | Expected | Actual | Status |
|------|--------|----------|--------|--------|
| 1 | microsoft.com | ✅ ALLOWED | [RESULT] | [PASS/FAIL] |
| 2 | httpbin.org | ❌ BLOCKED | [RESULT] | [PASS/FAIL] |
| 3 | api.github.com | ❌ BLOCKED | [RESULT] | [PASS/FAIL] |
| 4 | google.com | ❌ BLOCKED | [RESULT] | [PASS/FAIL] |
| 5 | malicious-example.com | ❌ BLOCKED | [RESULT] | [PASS/FAIL] |

## Overall Result

[PASS/FAIL] - Network permissions are [correctly/incorrectly] enforced

## Security Analysis

### Firewall Effectiveness
[Your analysis of how well the firewall blocked unauthorized domains]

### Configuration Validation
[Confirm that only microsoft.com was accessible as configured]

### Observations
[Any notable observations about error messages, response times, or firewall behavior]
```

## Issue Reporting

If ANY test fails (wrong expected vs actual result), create a GitHub issue using the `create_issue` tool (available through safe-outputs configuration) with:

**Title**: `[Security] Network Permissions Test Failure - [Date]`

**Body**:
```markdown
## Network Permissions Test Failure

**Workflow Run**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

### Failed Tests
[List which tests had unexpected results]

### Security Impact
[Describe the security implications - either firewall bypass or false positive blocking]

### Details
[Include the full test results table and any error messages]
```

The labels will be automatically applied based on the safe-outputs configuration.

## Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}
- **Run ID**: ${{ github.run_id }}
- **Network Configuration**: Only `microsoft.com` in allowed list
- **Firewall**: Agent Workflow Firewall (AWF) with Squid proxy
- **MCP Tool**: web-fetch for network access testing
