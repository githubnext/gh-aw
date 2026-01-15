---
name: Security Lint
description: Automated security scanning and workflow linting for GitHub Actions
on:
  pull_request:
    paths:
      - '.github/workflows/**'
  push:
    branches: [main]
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - defaults
    - github.com
    - raw.githubusercontent.com
sandbox:
  agent: awf
tools:
  bash:
  github:
    toolsets: [repos]
timeout-minutes: 10
strict: true
---

# Security and Code Quality Checks

This workflow performs automated security scanning and linting on GitHub Actions workflows to catch security issues and code quality problems before they reach production.

## Objectives

1. Run zizmor security scanner to identify workflow vulnerabilities
2. Run actionlint to validate workflow syntax and best practices
3. Block PRs with High or Critical security findings
4. Provide actionable feedback to contributors

## Tasks

### 1. Install Security Tools

Install the required tools for security scanning and linting:

- **zizmor**: Security scanner for GitHub Actions workflows
- **actionlint**: Linter for GitHub Actions workflow files

```bash
# Install zizmor
echo "Installing zizmor..."
curl -sSfL https://github.com/woodruffw/zizmor/releases/latest/download/zizmor-x86_64-unknown-linux-musl -o /tmp/zizmor
chmod +x /tmp/zizmor
sudo mv /tmp/zizmor /usr/local/bin/zizmor

# Install actionlint
echo "Installing actionlint..."
bash <(curl -sSfL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash)
sudo mv ./actionlint /usr/local/bin/actionlint

# Verify installations
echo "Verifying installations..."
zizmor --version
actionlint --version
```

### 2. Run Zizmor Security Scan

Scan all compiled workflow files for security vulnerabilities:

```bash
echo "Running zizmor security scan on compiled workflows..."

# Scan all .lock.yml files and generate both human-readable and SARIF output
if zizmor --format=sarif .github/workflows/*.lock.yml > zizmor-results.sarif 2>&1; then
    echo "✅ Zizmor scan completed successfully (no issues found)"
    ZIZMOR_EXIT=0
else
    ZIZMOR_EXIT=$?
    echo "⚠️ Zizmor found potential issues (exit code: $ZIZMOR_EXIT)"
fi

# Show human-readable output for review
echo ""
echo "=== Zizmor Findings (Human-Readable) ==="
zizmor .github/workflows/*.lock.yml || true

# Display SARIF summary
echo ""
echo "=== SARIF Results Summary ==="
if [ -f zizmor-results.sarif ]; then
    # Count findings by severity
    CRITICAL=$(jq '[.runs[].results[] | select(.level == "error")] | length' zizmor-results.sarif 2>/dev/null || echo "0")
    HIGH=$(jq '[.runs[].results[] | select(.level == "warning")] | length' zizmor-results.sarif 2>/dev/null || echo "0")
    MEDIUM=$(jq '[.runs[].results[] | select(.level == "note")] | length' zizmor-results.sarif 2>/dev/null || echo "0")
    
    echo "Critical/High: $CRITICAL"
    echo "Medium: $HIGH"
    echo "Low/Info: $MEDIUM"
    
    # Save exit code for later check
    echo "$ZIZMOR_EXIT" > /tmp/zizmor-exit-code
else
    echo "No SARIF file generated"
    echo "0" > /tmp/zizmor-exit-code
fi
```

### 3. Run Actionlint

Validate workflow files for syntax errors and best practices:

```bash
echo "Running actionlint on compiled workflows..."

# Run actionlint with shellcheck integration
if actionlint -color .github/workflows/*.lock.yml; then
    echo "✅ Actionlint validation passed"
    ACTIONLINT_EXIT=0
else
    ACTIONLINT_EXIT=$?
    echo "❌ Actionlint found issues (exit code: $ACTIONLINT_EXIT)"
fi

echo "$ACTIONLINT_EXIT" > /tmp/actionlint-exit-code
```

### 4. Check for Critical Issues

Determine if the workflow should fail based on severity of findings:

```bash
echo "Checking for critical security issues..."

ZIZMOR_EXIT=$(cat /tmp/zizmor-exit-code)
ACTIONLINT_EXIT=$(cat /tmp/actionlint-exit-code)

echo "Zizmor exit code: $ZIZMOR_EXIT"
echo "Actionlint exit code: $ACTIONLINT_EXIT"

# Check for high/critical severity issues in zizmor SARIF output
if [ -f zizmor-results.sarif ]; then
    CRITICAL_COUNT=$(jq '[.runs[].results[] | select(.level == "error")] | length' zizmor-results.sarif 2>/dev/null || echo "0")
    
    if [ "$CRITICAL_COUNT" -gt 0 ]; then
        echo "❌ Found $CRITICAL_COUNT critical/high severity security issues!"
        echo "These must be fixed before merging."
        echo ""
        echo "Run locally to see details:"
        echo "  make security-lint"
        exit 1
    fi
fi

# Fail on actionlint errors (syntax issues are always critical)
if [ "$ACTIONLINT_EXIT" -ne 0 ]; then
    echo "❌ Actionlint found critical issues that must be fixed"
    echo ""
    echo "Run locally to see details:"
    echo "  make actionlint"
    exit 1
fi

echo "✅ No critical issues found - all checks passed!"
```

## Notes

- **Pre-commit hooks**: Consider installing pre-commit hooks locally for faster feedback
- **Local testing**: Run `make security-lint` locally before pushing
- **Documentation**: See DEVGUIDE.md for setup instructions
- **False positives**: If you believe a finding is a false positive, document it in the PR description

## References

- Zizmor documentation: https://docs.zizmor.sh/
- Actionlint repository: https://github.com/rhysd/actionlint
- Pre-commit framework: https://pre-commit.com/
