---
title: Validation Rules Reference
description: Comprehensive reference for validation rules used in GitHub Agentic Workflows, including actionlint, zizmor, and shellcheck rules.
sidebar:
  order: 20
---

This reference documents all validation rules used to ensure workflow security and correctness. Understanding these rules helps you write better workflows and quickly resolve validation errors.

## Overview

GitHub Agentic Workflows uses multiple static analysis tools to validate workflows:

- **actionlint** - Lints GitHub Actions workflows and validates shell scripts
- **zizmor** - Security vulnerability scanner for GitHub Actions
- **shellcheck** - Shell script static analysis (integrated via actionlint)

## Actionlint Rules

Actionlint validates workflow syntax, type checking, and shell script correctness.

### Expression Validation

#### Undefined Step References

**Rule:** All step references must exist in the current job.

**Error Example:**
```text
workflow.yml:15:10: property "build" is not defined in object type {checkout,test}
```

**Problematic Code:**
```yaml
jobs:
  test:
    steps:
      - id: checkout
        uses: actions/checkout@v4
      - id: test
        run: npm test
      # Error: 'build' step doesn't exist
      - run: echo "${{ steps.build.outputs.result }}"
```

**Fix:**
```yaml
jobs:
  test:
    steps:
      - id: checkout
        uses: actions/checkout@v4
      - id: build
        run: npm run build
      - id: test
        run: npm test
      - run: echo "${{ steps.build.outputs.result }}"
```

#### Undefined Job Outputs

**Rule:** Referenced job outputs must be declared in the job's `outputs` section.

**Error Example:**
```text
workflow.yml:25:15: property "version" is not defined in object type {}
```

**Problematic Code:**
```yaml
jobs:
  build:
    # Missing outputs declaration
    steps:
      - id: version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  deploy:
    needs: build
    steps:
      - run: echo "${{ needs.build.outputs.version }}"
```

**Fix:**
```yaml
jobs:
  build:
    outputs:
      version: ${{ steps.version.outputs.version }}
    steps:
      - id: version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  deploy:
    needs: build
    steps:
      - run: echo "${{ needs.build.outputs.version }}"
```

### Syntax Validation

#### Missing Required Fields

**Rule:** Required workflow fields must be present.

**Error Example:**
```text
workflow.yml:5:3: "runs-on" is required but missing
```

**Problematic Code:**
```yaml
jobs:
  test:
    # Missing runs-on
    steps:
      - run: echo "test"
```

**Fix:**
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
```

#### Invalid Field Types

**Rule:** Field values must match expected types.

**Error Example:**
```text
workflow.yml:8:5: expected number but got string for "timeout-minutes"
```

**Problematic Code:**
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: "30"  # Should be number
```

**Fix:**
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 30  # Number, not string
```

### Deprecated Features

**Rule:** Deprecated GitHub Actions features should not be used.

**Error Example:**
```text
workflow.yml:10:7: set-output command is deprecated. Use $GITHUB_OUTPUT instead
```

**Problematic Code:**
```yaml
- run: echo "::set-output name=value::result"
```

**Fix:**
```yaml
- run: echo "value=result" >> $GITHUB_OUTPUT
```

## Shellcheck Integration

Actionlint integrates shellcheck to validate shell scripts in `run:` steps.

### SC2086: Quote Variables

**Rule:** Variables should be quoted to prevent word splitting and globbing.

**Error Example:**
```text
workflow.yml:15:5: shellcheck reported issue: SC2086:info:3:10: Double quote to prevent globbing and word splitting
```

**Problematic Code:**
```yaml
- run: |
    FILES=$(ls *.txt)
    for file in $FILES; do
      echo $file
    done
```

**Why it's problematic:**
- Files with spaces in names will break
- Glob patterns in variables are expanded unexpectedly
- Can lead to command injection if variable contains malicious content

**Fix:**
```yaml
- run: |
    while IFS= read -r file; do
      echo "$file"
    done < <(find . -name "*.txt")
```

### SC2016: Wrong Quote Type for Variables

**Rule:** Variables don't expand in single quotes.

**Error Example:**
```text
workflow.yml:12:5: shellcheck reported issue: SC2016:info:2:6: Expressions don't expand in single quotes
```

**Problematic Code:**
```yaml
- run: echo 'Value is $HOME'
  # Output: Value is $HOME (literal)
```

**Fix:**
```yaml
- run: echo "Value is $HOME"
  # Output: Value is /home/runner
```

### SC2129: Consolidate Redirects

**Rule:** Multiple redirects to the same file should be consolidated.

**Error Example:**
```text
workflow.yml:10:5: shellcheck reported issue: SC2129:style:3:1: Consider using { cmd1; cmd2; } >> file instead of individual redirects
```

**Problematic Code:**
```yaml
- run: |
    echo "line1" >> output.txt
    echo "line2" >> output.txt
    echo "line3" >> output.txt
```

**Fix:**
```yaml
- run: |
    {
      echo "line1"
      echo "line2"
      echo "line3"
    } >> output.txt
```

### SC2046: Quote Command Substitution

**Rule:** Command substitutions should be quoted to prevent word splitting.

**Error Example:**
```text
workflow.yml:8:5: shellcheck reported issue: SC2046:warning:2:10: Quote this to prevent word splitting
```

**Problematic Code:**
```yaml
- run: |
    for file in $(find . -name "*.txt"); do
      echo "$file"
    done
```

**Fix:**
```yaml
- run: |
    find . -name "*.txt" -print0 | while IFS= read -r -d '' file; do
      echo "$file"
    done
```

### SC2006: Use $() Instead of Backticks

**Rule:** Use `$()` for command substitution instead of backticks.

**Error Example:**
```text
workflow.yml:10:5: shellcheck reported issue: SC2006:style:2:8: Use $(...) notation instead of legacy backticked `...`
```

**Problematic Code:**
```yaml
- run: |
    RESULT=`command arg`
```

**Fix:**
```yaml
- run: |
    RESULT=$(command arg)
```

## Zizmor Rules

Zizmor detects security vulnerabilities in GitHub Actions workflows.

### Template Injection

**Rule ID:** `template-injection`  
**Severity:** Medium to Critical (context-dependent)

**Description:** Untrusted user input must not be used directly in GitHub Actions expressions.

**Detection Example:**
```text
finding: template-injection
  rule:
    id: template-injection
    level: High
    desc: Dangerous expression with user-controlled input
  location: workflow.yml:15
  details: github.event.issue.title used directly in expression
```

**Vulnerable Code:**
```yaml
on:
  issues:
    types: [opened]

jobs:
  process:
    steps:
      - run: echo "${{ github.event.issue.title }}"
```

**Fix:**
```yaml
on:
  issues:
    types: [opened]

jobs:
  process:
    steps:
      - env:
          ISSUE_TITLE: ${{ github.event.issue.title }}
        run: echo "$ISSUE_TITLE"
```

### Excessive Permissions

**Rule ID:** `excessive-permissions`  
**Severity:** Medium

**Description:** Workflows should use minimal required permissions following the principle of least privilege.

**Detection Example:**
```text
finding: excessive-permissions
  rule:
    id: excessive-permissions
    level: Medium
    desc: Workflow has unnecessarily broad permissions
  location: workflow.yml:5
  details: permissions set to write-all
```

**Problematic Code:**
```yaml
permissions: write-all  # Too broad
```

**Fix:**
```yaml
permissions:
  contents: read
  issues: write  # Only what's needed
```

### Artipacked (Missing Retention)

**Rule ID:** `artipacked`  
**Severity:** Low

**Description:** Artifacts should have explicit retention periods to prevent indefinite storage.

**Detection Example:**
```text
finding: artipacked
  rule:
    id: artipacked
    level: Low
    desc: Artifact upload without retention period
  location: workflow.yml:25
```

**Problematic Code:**
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: build-output
    path: dist/
    # Missing retention-days
```

**Fix:**
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: build-output
    path: dist/
    retention-days: 7  # Explicit retention
```

### Unpinned Actions

**Rule ID:** `unpinned-action`  
**Severity:** High

**Description:** Actions should be pinned to immutable SHA commits, not mutable tags or branches.

**Detection Example:**
```text
finding: unpinned-action
  rule:
    id: unpinned-action
    level: High
    desc: Action referenced by mutable tag
  location: workflow.yml:10
  details: actions/checkout@v4
```

**Problematic Code:**
```yaml
- uses: actions/checkout@v4  # Mutable tag
```

**Fix:**
```yaml
- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1
```

### Dangerous Workflow Triggers

**Rule ID:** `dangerous-triggers`  
**Severity:** High

**Description:** Some workflow triggers can be exploited if not properly secured.

**Detection Example:**
```text
finding: dangerous-triggers
  rule:
    id: dangerous-triggers
    level: High
    desc: pull_request_target with write permissions
  location: workflow.yml:2
```

**Problematic Code:**
```yaml
on:
  pull_request_target:  # Runs with base repo permissions

permissions:
  contents: write  # Dangerous combination
```

**Fix:**
```yaml
on:
  pull_request:  # Safer trigger

permissions:
  contents: read  # Read-only
```

## Validation Severity Levels

Understanding severity levels helps prioritize fixes:

### Critical
- **Impact:** Immediate security risk, data loss, or system compromise
- **Action:** Must fix before merge
- **Examples:** Template injection in production workflows, hardcoded secrets

### High
- **Impact:** Significant security vulnerability or major functional issue
- **Action:** Fix as soon as possible, ideally before merge
- **Examples:** Unpinned actions, excessive permissions, dangerous triggers

### Medium
- **Impact:** Security weakness or functional degradation
- **Action:** Fix within the development cycle
- **Examples:** Missing input validation, suboptimal permission scoping

### Low
- **Impact:** Code quality issue or minor inefficiency
- **Action:** Fix when convenient, can be addressed in cleanup
- **Examples:** Missing artifact retention, style inconsistencies

### Informational
- **Impact:** Best practice deviation, no immediate risk
- **Action:** Consider fixing for code quality improvement
- **Examples:** Deprecated but still functional syntax

## Running Validation

### Local Validation

```bash
# Compile and validate a single workflow
gh aw compile workflow-name.md

# Run actionlint
gh aw compile --actionlint

# For lock files directly
actionlint .github/workflows/*.lock.yml
```

### CI Validation

```yaml
# .github/workflows/validate.yml
name: Validate Workflows

on:
  pull_request:
    paths:
      - '.github/workflows/**.md'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@sha
      
      - name: Setup gh-aw
        run: |
          gh extension install githubnext/gh-aw
      
      - name: Compile workflows
        run: make recompile
      
      - name: Run actionlint
        run: make actionlint
```

## Suppressing False Positives

### Actionlint

Use inline comments to suppress specific warnings:

```yaml
- run: |
    # shellcheck disable=SC2086
    echo $SAFE_VARIABLE
```

### Zizmor

Use configuration file to suppress rules:

```yaml
# .zizmor.yml
ignore:
  - rule: artipacked
    path: .github/workflows/test.yml
    reason: Test workflow doesn't need retention
```

## Best Practices

1. **Fix High/Critical findings immediately** - Don't merge with critical security issues
2. **Use validation in CI/CD** - Catch issues before they reach main branch
3. **Keep tools updated** - New rules catch emerging vulnerabilities
4. **Document suppressions** - Always explain why a rule is suppressed
5. **Review regularly** - Periodic audits catch issues that slip through

## Additional Resources

### Tools
- [actionlint documentation](https://github.com/rhysd/actionlint)
- [zizmor documentation](https://github.com/woodruffw/zizmor)
- [shellcheck wiki](https://www.shellcheck.net/wiki/)

### Related Guides
- [Secure Workflow Authoring Guide](/guides/workflow-security-guide/)
- [CI Validation Setup](/guides/ci-validation-setup/)
- [Security Best Practices](/guides/security/)

### Internal Specs
- `specs/template-injection-prevention.md`
- `specs/github-actions-security-best-practices.md`
- `specs/validation-architecture.md`
