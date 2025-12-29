---
# Pattern: CI Monitor
# Complexity: Intermediate
# Use Case: Monitor CI/CD pipeline execution and analyze failures
name: CI Monitor
description: Monitors workflow runs and provides failure analysis and alerts
on:
  workflow_run:
    # TODO: Specify which workflows to monitor
    workflows:
      - "CI"
      - "Tests"
      - "Build"
    types:
      - completed
permissions:
  contents: read
  actions: read
  pull-requests: write
  issues: write
engine: copilot
tools:
  agentic-workflows:
  github:
    mode: remote
    toolsets: [actions, pull_requests, issues, repos]
  bash:
    - "gh *"
safe-outputs:
  add-comment:
    max: 1
    target: "*"
  create-issue:
    max: 1
timeout-minutes: 15
strict: true
---

# CI Monitor

Monitor CI/CD pipeline execution, analyze failures, and provide actionable insights when workflows fail.

## Monitoring Modes

# TODO: Choose ONE mode or combine multiple

### Mode 1: Failure Alert & Analysis

Alert when workflows fail and provide root cause analysis:

**When to Use**: Monitor critical workflows that should rarely fail

**Actions**:
1. Detect workflow failure
2. Analyze logs and error messages
3. Correlate with recent code changes
4. Post analysis to related PR or create issue

### Mode 2: Flaky Test Detection

Track and report tests that fail intermittently:

**When to Use**: Identify unreliable tests affecting CI stability

**Actions**:
1. Track test failure patterns
2. Identify tests failing >20% but <80% of time
3. Report flaky tests with statistics
4. Create issues for investigation

### Mode 3: Performance Monitoring

Track CI/CD performance metrics and trends:

**When to Use**: Monitor build times and resource usage

**Actions**:
1. Collect workflow duration data
2. Identify slow-running jobs
3. Track performance trends
4. Alert on significant regressions

### Mode 4: Success Rate Dashboard

Generate periodic reports on CI health:

**When to Use**: Weekly/monthly CI health reports

**Actions**:
1. Calculate success rates by workflow
2. Identify most problematic workflows
3. Track improvement trends
4. Create dashboard discussion

## Implementation Steps

### Step 1: Get Workflow Run Details

```bash
# Get the completed workflow run details
WORKFLOW_RUN_ID=${{ github.event.workflow_run.id }}
CONCLUSION=${{ github.event.workflow_run.conclusion }}

# Fetch detailed run information
gh run view $WORKFLOW_RUN_ID --json jobs,conclusion,url,headSha,createdAt
```

### Step 2: Analyze Failure (if failed)

```bash
# Only proceed if workflow failed
if [ "$CONCLUSION" != "success" ]; then
  # Use agentic-workflows MCP server audit tool
  # This provides comprehensive failure analysis
  
  # The audit tool gives you:
  # - Root cause analysis
  # - Error messages and stack traces
  # - Failed jobs and steps
  # - Tool usage patterns
  # - Performance metrics
  
  # TODO: Use the audit tool from agentic-workflows MCP server
  # Example: audit --run-id $WORKFLOW_RUN_ID
fi
```

### Step 3: Correlate with Code Changes

```bash
# Find related PR for the commit
HEAD_SHA=${{ github.event.workflow_run.head_sha }}

# Search for PR
gh pr list --repo ${{ github.repository }} \
  --search "sha:$HEAD_SHA" \
  --json number,title,url

# If PR found, analyze the changes
if [ -n "$PR_NUMBER" ]; then
  # Get PR diff
  gh pr diff $PR_NUMBER
  
  # Get changed files
  gh pr view $PR_NUMBER --json files
fi
```

### Step 4: Identify Root Cause

```python
#!/usr/bin/env python3
"""
Failure Root Cause Analysis
TODO: Customize for your project's error patterns
"""
import json
import re

# Load audit data (from agentic-workflows audit tool)
with open('/tmp/audit-data.json') as f:
    audit = json.load(f)

# Load PR changes (if available)
with open('/tmp/pr-changes.json') as f:
    pr_data = json.load(f)

root_cause = None

# Pattern 1: Syntax/Compile Errors
if 'SyntaxError' in audit['errors'] or 'compile error' in audit['errors']:
    # Find which file was recently changed and caused syntax error
    error_file = extract_file_from_error(audit['errors'])
    if error_file in [f['path'] for f in pr_data['files']]:
        root_cause = {
            'type': 'syntax_error',
            'file': error_file,
            'message': 'Syntax error in recently modified file',
            'fix': f'Review changes in {error_file}'
        }

# Pattern 2: Test Failures
elif 'test failed' in audit['errors'].lower():
    test_name = extract_test_name(audit['errors'])
    root_cause = {
        'type': 'test_failure',
        'test': test_name,
        'message': f'Test {test_name} failed',
        'fix': 'Review test expectations or implementation'
    }

# Pattern 3: Dependency Issues
elif 'module not found' in audit['errors'] or 'cannot import' in audit['errors']:
    root_cause = {
        'type': 'dependency_issue',
        'message': 'Missing or incompatible dependency',
        'fix': 'Check requirements.txt, package.json, or go.mod'
    }

# Pattern 4: Timeout
elif audit['conclusion'] == 'timeout':
    root_cause = {
        'type': 'timeout',
        'message': 'Workflow exceeded time limit',
        'fix': 'Optimize slow tests or increase timeout'
    }

# Save analysis
with open('/tmp/root-cause.json', 'w') as f:
    json.dump(root_cause, f, indent=2)
```

### Step 5: Generate Report

```markdown
# TODO: Customize report format

## 🔴 CI Failure Alert

Workflow **[${{ github.event.workflow_run.name }}](${{ github.event.workflow_run.html_url }})** failed.

### Details
- **Run**: #${{ github.event.workflow_run.run_number }}
- **Commit**: ${{ github.event.workflow_run.head_sha }}
- **Branch**: ${{ github.event.workflow_run.head_branch }}
- **Conclusion**: ${{ github.event.workflow_run.conclusion }}
- **Duration**: [duration]

### Root Cause Analysis

**Error Type**: [type]

**What Happened**:
[Detailed explanation of the failure]

**Which Files/Tests**:
- [file1.py] - [specific issue]
- [test_feature.js] - [failure reason]

**Error Messages**:
```
[Key error messages from logs]
```

### Recent Changes

**Related PR**: #[number] ([title](url))

**Files Changed**:
- [file1] - [additions/deletions]
- [file2] - [additions/deletions]

**Correlation**:
The failure appears to be caused by changes in [specific file/function].

### Recommended Actions

1. **Immediate**: [What to do first]
2. **Investigation**: [What to check]
3. **Fix**: [Suggested solution]

### Helpful Commands

```bash
# Reproduce locally
[commands to reproduce the failure]

# View full logs
gh run view ${{ github.event.workflow_run.id }} --log

# Re-run the workflow
gh run rerun ${{ github.event.workflow_run.id }}
```

---
*Analysis by [CI Monitor]({run_url})*
```

### Step 6: Post Alert

```markdown
# TODO: Choose where to post the alert

**Option A**: Comment on related PR (preferred)
- Use add-comment safe-output
- Target: PR associated with the commit
- Provides immediate feedback to author

**Option B**: Create issue for persistent failures
- Use create-issue safe-output
- For failures on main branch or recurring issues
- Track resolution separately

**Option C**: Both
- Comment on PR for immediate feedback
- Create issue if failure is critical or repeated
```

## Customization Guide

### Configure Monitored Workflows

```yaml
# TODO: List workflows to monitor
on:
  workflow_run:
    workflows:
      - "CI"                    # Main CI pipeline
      - "Tests"                 # Test suite
      - "Build and Deploy"      # Build process
      - "Security Scan"         # Security checks
      # - "Lint"               # Optionally monitor linting
    types:
      - completed               # Trigger when workflow completes
    # Optional: Only monitor specific branches
    # branches:
    #   - main
    #   - develop
```

### Define Failure Severity

```python
# TODO: Categorize failures by severity

def determine_severity(workflow_name, conclusion, branch):
    """Determine how critical the failure is"""
    
    # Critical: Main branch failures in core workflows
    if branch == 'main' and workflow_name in ['CI', 'Build']:
        return 'critical'
    
    # High: Test failures on main or deploy failures
    if (branch == 'main' and 'test' in workflow_name.lower()) or \
       'deploy' in workflow_name.lower():
        return 'high'
    
    # Medium: Branch failures in core workflows
    if workflow_name in ['CI', 'Tests']:
        return 'medium'
    
    # Low: Branch failures in optional workflows
    return 'low'

# Adjust alerting based on severity
if severity == 'critical':
    # Post issue + comment + notify team
    create_issue()
    comment_on_pr()
    send_team_notification()
elif severity == 'high':
    # Post issue or comment
    create_issue_or_comment()
else:
    # Just comment on PR
    comment_on_pr()
```

### Add Custom Error Patterns

```python
# TODO: Add patterns specific to your tech stack

ERROR_PATTERNS = {
    'go': {
        'pattern': r'undefined: (\w+)',
        'message': 'Undefined identifier: {match}',
        'fix': 'Check if package is imported or variable is declared'
    },
    'python': {
        'pattern': r'ModuleNotFoundError: No module named \'(\w+)\'',
        'message': 'Missing Python module: {match}',
        'fix': 'Add {match} to requirements.txt'
    },
    'javascript': {
        'pattern': r'Cannot find module \'([\w-]+)\'',
        'message': 'Missing npm package: {match}',
        'fix': 'Run: npm install {match}'
    },
    'docker': {
        'pattern': r'Error: No such image: ([\w:]+)',
        'message': 'Docker image not found: {match}',
        'fix': 'Pull image: docker pull {match}'
    }
}
```

### Track Flaky Tests

```python
# TODO: Implement flaky test tracking

def track_test_failure(test_name, passed):
    """Track test pass/fail history"""
    history = load_test_history()
    
    if test_name not in history:
        history[test_name] = {'passes': 0, 'failures': 0}
    
    if passed:
        history[test_name]['passes'] += 1
    else:
        history[test_name]['failures'] += 1
    
    save_test_history(history)
    
    # Check if test is flaky
    total = history[test_name]['passes'] + history[test_name]['failures']
    failure_rate = history[test_name]['failures'] / total
    
    # Flaky if fails between 20% and 80% of time
    if 0.2 <= failure_rate <= 0.8 and total >= 10:
        report_flaky_test(test_name, failure_rate)
```

## Example Outputs

### Example 1: Syntax Error Alert

```markdown
## 🔴 CI Failure Alert

Workflow **[CI](run-url)** failed on commit `abc1234`.

### Root Cause Analysis

**Error Type**: Syntax Error

**What Happened**:
A syntax error was introduced in `src/api/handler.go` preventing compilation.

**Error Message**:
```
src/api/handler.go:42:15: syntax error: unexpected newline, expecting comma or }
```

**Files Changed**:
- `src/api/handler.go` - Added new endpoint (45 lines)

**Correlation**:
The error is in line 42 of the recently modified file. The new `CreateUser` function is missing a closing brace.

### Recommended Actions

1. **Fix**: Add missing `}` at line 42 in `src/api/handler.go`
2. **Test**: Run `go build` locally before pushing
3. **Prevent**: Enable pre-commit hooks or local CI checks

### Reproduce Locally

```bash
git checkout ${{ github.event.workflow_run.head_branch }}
go build ./...
```

---
*Analysis by [CI Monitor](run-url) • [View Full Logs](logs-url)*
```

### Example 2: Test Failure Alert

```markdown
## 🔴 Test Failure Alert

Workflow **[Tests](run-url)** failed - 2 tests failed.

### Failed Tests

1. **`test_user_authentication`** (tests/test_auth.py)
   - Expected: 200 OK
   - Actual: 401 Unauthorized
   - Duration: 0.3s

2. **`test_user_profile_update`** (tests/test_users.py)
   - Error: KeyError: 'email'
   - Duration: 0.2s

### Analysis

**Related PR**: #123 ([Update user authentication](pr-url))

**Files Changed**:
- `src/auth.py` - Modified token validation (23 lines)
- `src/models/user.py` - Added email field (12 lines)

**Root Cause**:
The authentication changes in `src/auth.py` appear to have broken token validation logic. The `email` field addition requires database migration.

### Recommended Actions

1. Review token validation changes in `src/auth.py`
2. Run database migration for new `email` field
3. Update test fixtures with email field
4. Re-run tests locally: `pytest tests/test_auth.py tests/test_users.py`

---
*Analysis by [CI Monitor](run-url)*
```

## Advanced Features

### Auto-Retry Transient Failures

```yaml
# TODO: Configure auto-retry
retry-strategy:
  transient-patterns:
    - "connection timeout"
    - "temporary failure"
    - "rate limit exceeded"
  max-retries: 2
  delay-seconds: 300
```

### Performance Regression Detection

```python
# TODO: Track performance over time

def check_performance_regression(current_duration, workflow_name):
    """Alert if workflow is significantly slower"""
    history = load_duration_history(workflow_name)
    avg_duration = statistics.mean(history[-10:])  # Last 10 runs
    
    # Alert if 50% slower than average
    if current_duration > avg_duration * 1.5:
        return {
            'alert': True,
            'message': f'Workflow took {current_duration}s (avg: {avg_duration}s)',
            'regression': f'{(current_duration/avg_duration - 1)*100:.1f}%'
        }
    
    return {'alert': False}
```

### Integration with Issue Tracking

```markdown
Create issues for persistent failures:
- Check if issue already exists for this failure type
- Add labels: "ci-failure", "bug", severity level
- Link to failed workflow run
- Assign to code owner of failed files
```

## Related Examples

- **Production examples**:
  - `.github/workflows/dev-hawk.md` - Comprehensive CI monitoring
  - `.github/workflows/ci-doctor.md` - CI health diagnostics

## Tips

- **Be selective**: Only monitor critical workflows
- **Be actionable**: Provide specific fix suggestions
- **Be timely**: Alert immediately on failures
- **Be smart**: Correlate failures with code changes
- **Track patterns**: Identify recurring issues
- **Reduce noise**: Don't alert on known flaky tests

## Security Considerations

- This workflow reads CI logs and posts comments
- Uses `strict: true` for enhanced security
- No write access to code or workflows
- All operations validated through safe-outputs

---

**Pattern Info**:
- Complexity: Intermediate
- Trigger: workflow_run completed
- Safe Outputs: add_comment, create_issue
- Tools: GitHub (actions, pull_requests, issues), agentic-workflows (audit)
