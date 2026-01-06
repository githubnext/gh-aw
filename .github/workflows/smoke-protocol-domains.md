---
description: Smoke test for protocol-specific domain filtering
on: 
  schedule: every 24h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke-protocol-domains"]
  reaction: "eyes"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Protocol Domains
engine: copilot
network:
  allowed:
    - defaults
    - node
    - github
    - "https://api.github.com"  # HTTPS-only (should work since api.github.com is in defaults)
    - "http://httpbin.org"       # HTTP-only test endpoint
sandbox:
  agent: awf  # Firewall enabled
tools:
  bash:
    - "*"
  github:
  web-fetch:
safe-outputs:
    add-comment:
      hide-older-comments: true
    add-labels:
      allowed: [smoke-protocol-domains]
    messages:
      footer: "> 🔒 *Protocol Security Test: Report by [{workflow_name}]({run_url})*"
      run-started: "🔒 Protocol filtering test [{workflow_name}]({run_url}) started..."
      run-success: "✅ Protocol filtering test [{workflow_name}]({run_url}) passed. All protocol restrictions working correctly."
      run-failure: "❌ Protocol filtering test [{workflow_name}]({run_url}) failed with {status}."
timeout-minutes: 5
strict: true
---

# Smoke Test: Protocol-Specific Domain Filtering

**IMPORTANT: Keep outputs short and concise.**

## Test Requirements

Test protocol-specific domain filtering with the AWF firewall:

1. **HTTPS-only Domain Test**: Verify that `https://api.github.com` is accessible (included in defaults with protocol prefix)
2. **HTTP-only Domain Test**: Verify that `http://httpbin.org` would be accessible if tested (network allows HTTP-only)
3. **Mixed Protocol Test**: Verify that domains without protocol prefixes (from defaults/ecosystems) work with both HTTP and HTTPS
4. **Firewall Configuration Verification**: Confirm the AWF `--allow-domains` flag includes protocol prefixes in the workflow logs

## Test Actions

1. Use web_fetch to access `https://api.github.com/repos/githubnext/gh-aw` (should succeed)
2. Check `/tmp/gh-aw/agent-stdio.log` for the AWF command line to verify protocol prefixes are passed correctly
3. Look for patterns like `https://api.github.com` and `http://httpbin.org` in the --allow-domains flag

## Output

Add a **brief** comment to the current pull request with:
- ✅ HTTPS-only domain access test result
- ✅ Protocol prefix verification in AWF command
- ✅ Overall protocol filtering status
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-protocol-domains` to the pull request.

**Expected AWF command should include:** `--allow-domains ...,http://httpbin.org,https://api.github.com,...`
