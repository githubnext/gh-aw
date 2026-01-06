---
title: CI Validation Setup
description: Guide for setting up continuous integration validation for GitHub Agentic Workflows, including local validation, CI/CD integration, and pre-commit hooks.
sidebar:
  order: 5
---

This guide explains how to integrate workflow validation into your development process, from local pre-commit checks to automated CI/CD pipelines.

## Overview

Validation should happen at multiple stages:

1. **Local development** - Pre-commit hooks catch issues before commit
2. **Pull request** - CI validation blocks merges with security issues
3. **Scheduled scans** - Regular audits catch emerging vulnerabilities

## Local Validation

### Manual Validation

Validate workflows during development:

```bash
# Compile a single workflow
gh aw compile workflow-name.md

# Compile with validation
gh aw compile workflow-name.md --actionlint

# Compile all workflows
make recompile

# Run actionlint on all workflows
make actionlint
```

### Validation Script

Create a reusable validation script at `scripts/validate-workflow.sh`:

```bash
#!/bin/bash
# Validate a workflow before commit

set -euo pipefail

workflow="${1:-}"

if [ -z "$workflow" ]; then
  echo "Usage: $0 <workflow.md>"
  exit 1
fi

echo "Validating $workflow..."

# Extract workflow name without extension
workflow_name="${workflow%.md}"
lock_file="${workflow_name}.lock.yml"

# Compile the workflow
echo "→ Compiling workflow..."
if ! gh aw compile "$workflow"; then
  echo "✗ Compilation failed"
  exit 1
fi

# Run actionlint (if available)
if command -v actionlint &> /dev/null; then
  echo "→ Running actionlint..."
  if ! actionlint "$lock_file"; then
    echo "✗ actionlint found issues"
    exit 1
  fi
else
  echo "⚠ actionlint not installed, skipping..."
fi

# Run zizmor (if available)
if command -v zizmor &> /dev/null; then
  echo "→ Running zizmor..."
  if ! zizmor "$lock_file"; then
    echo "⚠ zizmor found issues (non-blocking)"
  fi
else
  echo "⚠ zizmor not installed, skipping..."
fi

echo "✅ Validation passed"
```

Make it executable:

```bash
chmod +x scripts/validate-workflow.sh
```

Use it:

```bash
./scripts/validate-workflow.sh .github/workflows/my-workflow.md
```

## Pre-Commit Hooks

### Git Hooks Setup

Create a pre-commit hook at `.githooks/pre-commit`:

```bash
#!/bin/bash
# Pre-commit hook to validate changed workflows

set -euo pipefail

# Check if gh-aw is available
if ! command -v gh &> /dev/null || ! gh aw --help &> /dev/null 2>&1; then
  echo "⚠ Warning: gh-aw not installed, skipping workflow validation"
  exit 0
fi

# Get list of changed workflow files
workflows=$(git diff --cached --name-only --diff-filter=ACM | grep '^\.github/workflows/.*\.md$' || true)

if [ -z "$workflows" ]; then
  # No workflow files changed
  exit 0
fi

echo "Validating changed workflows..."

failed=0
for workflow in $workflows; do
  echo ""
  echo "Checking $workflow..."
  
  # Validate using script if available
  if [ -x "./scripts/validate-workflow.sh" ]; then
    if ! ./scripts/validate-workflow.sh "$workflow"; then
      failed=1
    fi
  else
    # Fallback to direct compilation
    if ! gh aw compile "$workflow"; then
      failed=1
    fi
  fi
done

if [ $failed -eq 1 ]; then
  echo ""
  echo "✗ Workflow validation failed"
  echo "Fix the issues above or use 'git commit --no-verify' to skip validation"
  exit 1
fi

echo ""
echo "✅ All workflows validated successfully"
exit 0
```

Make it executable:

```bash
chmod +x .githooks/pre-commit
```

### Configuring Git to Use Custom Hooks Directory

Tell git to use the `.githooks` directory:

```bash
# One-time setup per repository
git config core.hooksPath .githooks
```

Or add this to your repository setup instructions in `CONTRIBUTING.md`:

```markdown
## Development Setup

After cloning the repository:

```bash
# Configure git hooks
git config core.hooksPath .githooks
```
```

### Alternative: Husky (Node.js Projects)

If your project uses Node.js, you can use Husky:

```bash
# Install Husky
npm install --save-dev husky

# Enable Git hooks
npx husky install

# Add pre-commit hook
npx husky add .husky/pre-commit "make validate-workflows"
```

Then add the validation command to `package.json`:

```json
{
  "scripts": {
    "validate-workflows": "./scripts/validate-workflow.sh"
  }
}
```

## CI/CD Integration

### GitHub Actions Workflow

Create `.github/workflows/validate-workflows.yml`:

```yaml
name: Validate Workflows

on:
  pull_request:
    paths:
      - '.github/workflows/**.md'
      - '.github/workflows/**.lock.yml'
  push:
    branches:
      - main
    paths:
      - '.github/workflows/**.md'

jobs:
  validate:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write  # For posting comments
    
    steps:
      - name: Checkout code
        uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1
      
      - name: Setup Go
        uses: actions/setup-go@0c52d547c9bc32b1aa3301fd7a9cb496313a4491 # v5.0.0
        with:
          go-version-file: 'go.mod'
      
      - name: Install gh-aw
        run: |
          gh extension install githubnext/gh-aw
        env:
          GH_TOKEN: ${{ github.token }}
      
      - name: Compile workflows
        run: make recompile
      
      - name: Run actionlint
        run: make actionlint
      
      - name: Install zizmor (optional)
        run: |
          cargo install zizmor || echo "zizmor installation failed, skipping"
        continue-on-error: true
      
      - name: Run zizmor
        run: |
          if command -v zizmor &> /dev/null; then
            zizmor .github/workflows/*.lock.yml || echo "zizmor found issues"
          else
            echo "zizmor not available, skipping"
          fi
        continue-on-error: true
      
      - name: Upload validation results
        if: always()
        uses: actions/upload-artifact@5d5d22a31266ced268874388b861e4b58bb5c2f3 # v4.3.1
        with:
          name: validation-results
          path: |
            .github/workflows/*.lock.yml
          retention-days: 7
```

### Required Status Checks

Configure branch protection rules to require validation:

1. Go to repository **Settings** → **Branches**
2. Add or edit branch protection rule for `main`
3. Enable **Require status checks to pass before merging**
4. Select **Validate Workflows** as required check

### Matrix Testing (Multiple Workflow Versions)

For large repositories with many workflows:

```yaml
name: Validate Workflows

on:
  pull_request:
    paths:
      - '.github/workflows/**.md'

jobs:
  list-workflows:
    runs-on: ubuntu-latest
    outputs:
      workflows: ${{ steps.list.outputs.workflows }}
    steps:
      - uses: actions/checkout@v4
      
      - id: list
        run: |
          workflows=$(find .github/workflows -name "*.md" -type f | jq -R -s -c 'split("\n")[:-1]')
          echo "workflows=$workflows" >> $GITHUB_OUTPUT
  
  validate:
    needs: list-workflows
    runs-on: ubuntu-latest
    strategy:
      matrix:
        workflow: ${{ fromJson(needs.list-workflows.outputs.workflows) }}
      fail-fast: false  # Continue validating all workflows
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Install gh-aw
        run: gh extension install githubnext/gh-aw
        env:
          GH_TOKEN: ${{ github.token }}
      
      - name: Validate ${{ matrix.workflow }}
        run: ./scripts/validate-workflow.sh "${{ matrix.workflow }}"
```

## Automated Security Scanning

### Scheduled Vulnerability Scans

Create `.github/workflows/security-scan.yml`:

```yaml
name: Security Scan

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
  workflow_dispatch:  # Manual trigger

jobs:
  scan:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      issues: write  # For creating security issues
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Install security tools
        run: |
          cargo install zizmor
          brew install actionlint
      
      - name: Scan workflows
        id: scan
        run: |
          # Run zizmor and save results
          zizmor --format json .github/workflows/*.lock.yml > zizmor-results.json || true
          
          # Run actionlint
          actionlint .github/workflows/*.lock.yml > actionlint-results.txt 2>&1 || true
          
          # Check for High/Critical findings
          if grep -q '"level": "High\|Critical"' zizmor-results.json; then
            echo "has_critical=true" >> $GITHUB_OUTPUT
          fi
      
      - name: Create issue for critical findings
        if: steps.scan.outputs.has_critical == 'true'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const results = JSON.parse(fs.readFileSync('zizmor-results.json', 'utf8'));
            
            const critical = results.findings.filter(f => 
              f.rule.level === 'High' || f.rule.level === 'Critical'
            );
            
            const body = `## Security Scan Alert\n\n` +
              `Found ${critical.length} high/critical security issues:\n\n` +
              critical.map(f => 
                `- **${f.rule.id}** (${f.rule.level}): ${f.rule.desc}\n` +
                `  Location: ${f.location}\n`
              ).join('\n');
            
            await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: '🚨 Security Scan: Critical Workflow Issues Found',
              body: body,
              labels: ['security', 'workflows']
            });
      
      - name: Upload scan results
        uses: actions/upload-artifact@v4
        with:
          name: security-scan-results
          path: |
            zizmor-results.json
            actionlint-results.txt
          retention-days: 90
```

## Installing Validation Tools

### Actionlint

**macOS:**
```bash
brew install actionlint
```

**Linux:**
```bash
# Download latest release
bash <(curl -sSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash)

# Add to PATH
sudo mv actionlint /usr/local/bin/
```

**Via Go:**
```bash
go install github.com/rhysd/actionlint/cmd/actionlint@latest
```

**Docker:**
```bash
# Use Docker image (no installation needed)
docker run --rm -v "$(pwd):/repo" rhysd/actionlint:latest -color /repo/.github/workflows/
```

### Zizmor

**Via Cargo (Rust):**
```bash
cargo install zizmor
```

**Via Homebrew:**
```bash
# Not yet available, but may be added in future
```

**Docker:**
```bash
# Check zizmor documentation for Docker images
```

### Shellcheck

**macOS:**
```bash
brew install shellcheck
```

**Linux:**
```bash
sudo apt-get install shellcheck
```

**Via Docker:**
```bash
docker run --rm -v "$PWD:/mnt" koalaman/shellcheck:stable script.sh
```

## IDE Integration

### VS Code

Install extensions for real-time validation:

1. **GitHub Actions** - Syntax highlighting and validation
2. **shellcheck** - Shell script linting

Add to `.vscode/settings.json`:

```json
{
  "files.associations": {
    "*.md": "markdown"
  },
  "yaml.schemas": {
    "https://json.schemastore.org/github-workflow.json": ".github/workflows/*.yml"
  }
}
```

### Pre-Save Validation

Configure VS Code to run validation on save:

```json
{
  "runOnSave.commands": [
    {
      "match": "\\.github/workflows/.*\\.md$",
      "command": "./scripts/validate-workflow.sh ${file}",
      "runIn": "terminal"
    }
  ]
}
```

## Troubleshooting

### Common Issues

**Issue:** Pre-commit hook not running

**Solution:**
```bash
# Check hooks directory configuration
git config core.hooksPath

# Should output: .githooks

# If not set:
git config core.hooksPath .githooks
```

**Issue:** Validation fails in CI but passes locally

**Solution:**
- Ensure you're using same tool versions
- Check for uncommitted changes to workflows
- Review CI logs for environment differences

**Issue:** Actionlint not found in CI

**Solution:**
```yaml
# Ensure actionlint is installed in CI
- name: Install actionlint
  run: |
    bash <(curl -sSL https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash)
    sudo mv actionlint /usr/local/bin/
```

## Best Practices

1. **Validate early and often** - Run validation before every commit
2. **Block merges on failures** - Use required status checks
3. **Keep tools updated** - New versions catch new vulnerabilities
4. **Document suppressions** - Explain why validation rules are disabled
5. **Regular audits** - Schedule weekly/monthly security scans
6. **Team training** - Ensure all contributors understand validation

## Additional Resources

### Related Guides
- [Secure Workflow Authoring Guide](/guides/workflow-security-guide/)
- [Validation Rules Reference](/reference/validation-rules/)
- [Security Best Practices](/guides/security/)

### Tool Documentation
- [actionlint](https://github.com/rhysd/actionlint)
- [zizmor](https://github.com/woodruffw/zizmor)
- [shellcheck](https://www.shellcheck.net/)

### Internal Documentation
- `CONTRIBUTING.md` - Contribution guidelines
- `DEVGUIDE.md` - Development guide
- `specs/validation-architecture.md` - Validation system architecture
