---
on:
  workflow_dispatch:
  schedule:
    - cron: "0 14 * * 1-5" # 2 PM UTC, weekdays only
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
safe-outputs:
  create-issue:
    title-prefix: "[linter] "
    labels: [automation, code-quality]
engine: copilot
name: Super Linter Report
timeout-minutes: 15
imports:
  - shared/reporting.md
jobs:
  super_linter:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v5
        with:
          fetch-depth: 0
      
      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
      
      - name: Set up Node.js
        uses: actions/setup-node@v6
        with:
          node-version: "24"
          cache: npm
          cache-dependency-path: pkg/workflow/js/package-lock.json
      
      - name: Install Dependencies
        run: |
          go mod download
          cd pkg/workflow/js && npm ci
      
      - name: Run Super Linter
        id: super-linter
        continue-on-error: true
        uses: super-linter/super-linter/slim@v8
        env:
          DEFAULT_BRANCH: main
          FILTER_REGEX_EXCLUDE: dist/**/*
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          LINTER_RULES_PATH: .
          VALIDATE_ALL_CODEBASE: "true"
          # Disable linters that are covered by other workflows or not applicable
          VALIDATE_GO: "false"                # golangci-lint is used instead in CI
          VALIDATE_GO_MODULES: "false"        # Go mod verification in CI
          VALIDATE_JAVASCRIPT_ES: "false"     # ESLint/npm test handles JS linting
          VALIDATE_TYPESCRIPT_ES: "false"     # Not using TypeScript
          VALIDATE_JSCPD: "false"              # Copy-paste detection not required
          VALIDATE_JSON: "false"               # Not strictly enforced
          VALIDATE_GITHUB_ACTIONS: "true"     # Keep GitHub Actions validation
          VALIDATE_MARKDOWN: "true"            # Keep Markdown validation
          VALIDATE_YAML: "true"                # Keep YAML validation
          VALIDATE_SHELL_SHFMT: "true"         # Keep shell script formatting
          VALIDATE_BASH: "true"                # Keep bash validation
          LOG_FILE: /tmp/gh-aw/super-linter.log
          CREATE_LOG_FILE: "true"
      
      - name: Check for linting issues
        id: check-results
        run: |
          if [ -f "/tmp/gh-aw/super-linter.log" ] && [ -s "/tmp/gh-aw/super-linter.log" ]; then
            # Check if there are actual errors (not just the header)
            if grep -qE "ERROR|WARN|FAIL" /tmp/gh-aw/super-linter.log; then
              echo "needs-linting=true" >> $GITHUB_OUTPUT
            else
              echo "needs-linting=false" >> $GITHUB_OUTPUT
            fi
          else
            echo "needs-linting=false" >> $GITHUB_OUTPUT
          fi
      
      - name: Upload super-linter log
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: super-linter-log
          path: /tmp/gh-aw/super-linter.log
          retention-days: 7
steps:
  - name: Download super-linter log
    uses: actions/download-artifact@v4
    with:
      name: super-linter-log
      path: /tmp/gh-aw/
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
---

# Super Linter Analysis Report

You are an expert code quality analyst for a Go-based GitHub CLI extension project. Your task is to analyze the super-linter output and create a comprehensive issue report.

## Context

- **Repository**: ${{ github.repository }}
- **Project Type**: Go CLI tool (GitHub Agentic Workflows extension)
- **Triggered by**: @${{ github.actor }}
- **Run ID**: ${{ github.run_id }}

## Your Task

1. **Read the linter output** from `/tmp/gh-aw/super-linter.log` using the bash tool
2. **Analyze the findings**:
   - Categorize errors by severity (critical, high, medium, low)
   - Group errors by file type or linter (Markdown, YAML, Shell, GitHub Actions)
   - Identify patterns in the errors
   - Determine which errors are most important to fix first
   - Note: Go and JavaScript linting are handled by dedicated CI jobs (golangci-lint, npm test)
3. **Create a detailed issue** with the following structure:

### Issue Title
Use format: "Code Quality Report - [Date] - [X] issues found"

### Issue Body Structure

```markdown
## 🔍 Super Linter Analysis Summary

**Date**: [Current date]
**Total Issues Found**: [Number]
**Run ID**: ${{ github.run_id }}

## 📊 Breakdown by Severity

- **Critical**: [Count and brief description]
- **High**: [Count and brief description]  
- **Medium**: [Count and brief description]
- **Low**: [Count and brief description]

## 📁 Issues by Category

### [Category/Linter Name]
- **File**: `path/to/file`
  - Line [X]: [Error description]
  - Impact: [Why this matters]
  - Suggested fix: [How to resolve]

[Repeat for other categories]

## 🎯 Priority Recommendations

1. [Most critical issue to address first]
2. [Second priority]
3. [Third priority]

## 📋 Full Linter Output

<details>
<summary>Click to expand complete linter log</summary>

```
[Include the full linter output here]
```

</details>

## 🔗 References

- [Link to workflow run](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})
- [Super Linter Documentation](https://github.com/super-linter/super-linter)
- [Project CI Configuration](${{ github.server_url }}/${{ github.repository }}/blob/main/.github/workflows/ci.yml)
```

## Important Guidelines

- **Be concise but thorough**: Focus on actionable insights
- **Prioritize issues**: Not all linting errors are equal
- **Provide context**: Explain why each type of error matters for a CLI tool project
- **Suggest fixes**: Give practical recommendations
- **Use proper formatting**: Make the issue easy to read and navigate
- **If no errors found**: Create a positive report celebrating clean code
- **Remember**: This is a Go project with separate Go/JS linting in CI, so focus on Markdown, YAML, Shell, and GitHub Actions files

## Validating Fixes with Super Linter

When suggesting fixes for linting errors, you can provide instructions for running super-linter locally to validate the fixes before committing. Include this section in your issue report when relevant:

### Running Super Linter Locally

To validate your fixes locally before committing, run super-linter using Docker:

```bash
# Run super-linter on the entire repository
docker run --rm \
  -e DEFAULT_BRANCH=main \
  -e RUN_LOCAL=true \
  -e VALIDATE_ALL_CODEBASE=true \
  -e VALIDATE_GO=false \
  -e VALIDATE_GO_MODULES=false \
  -e VALIDATE_JAVASCRIPT_ES=false \
  -e VALIDATE_TYPESCRIPT_ES=false \
  -e VALIDATE_JSCPD=false \
  -e VALIDATE_JSON=false \
  -v $(pwd):/tmp/lint \
  ghcr.io/super-linter/super-linter:slim-v8

# Run super-linter on specific file types only
# For example, to validate only Markdown files:
docker run --rm \
  -e RUN_LOCAL=true \
  -e VALIDATE_ALL_CODEBASE=true \
  -e VALIDATE_MARKDOWN=true \
  -v $(pwd):/tmp/lint \
  ghcr.io/super-linter/super-linter:slim-v8
```

**Note**: The Docker command uses the same super-linter configuration as this workflow. Files are mounted from your current directory to `/tmp/lint` in the container.

## Security Note

Treat linter output as potentially sensitive. Do not expose credentials, API keys, or other secrets that might appear in file paths or error messages.
