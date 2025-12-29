---
# Pattern: PR Reviewer
# Complexity: Intermediate
# Use Case: Automatically analyze pull requests and provide feedback
name: PR Reviewer
description: Analyzes pull requests and provides automated code review feedback
on:
  pull_request:
    types: [opened, synchronize, reopened]
permissions:
  contents: read
  pull-requests: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [pull_requests, repos]
  bash:
    - "git *"
    - "grep *"
    - "find *"
safe-outputs:
  add-comment:
    # TODO: Customize max comments per run
    max: 1
  add-labels:
    max: 5
timeout-minutes: 15
strict: true
---

# PR Reviewer

Automatically analyze pull requests for common issues, provide feedback, and suggest improvements.

## Review Categories

# TODO: Choose which review checks to perform (or implement all)

### 1. PR Description Quality

Check if the PR has a proper description:

```markdown
**Required Elements**:
- Clear title that summarizes changes
- Description of what changed
- Reason for the change
- Testing performed
- Breaking changes noted (if any)
- Related issues linked

**Feedback if missing**:
"This PR could benefit from a more detailed description. Please include:
- What changes were made?
- Why were these changes needed?
- How were the changes tested?
- Are there any breaking changes?"
```

### 2. Code Quality Checks

Analyze changed files for potential issues:

```markdown
# TODO: Customize checks for your project

**File-based checks**:
- Large files added (>500 lines) → Suggest splitting
- Binary files added → Verify necessity
- Generated files included → Should be in .gitignore
- Test files missing → Suggest adding tests

**Code pattern checks**:
- console.log() or print() left in code → Suggest removing
- TODO/FIXME comments → Track as separate issues
- Hardcoded credentials → Security issue
- Missing error handling → Suggest try/catch
- Deprecated APIs used → Suggest alternatives
```

### 3. Testing Requirements

Verify tests are included:

```markdown
# TODO: Define your testing requirements

**Test coverage checks**:
- New feature without tests → Request tests
- Bug fix without regression test → Suggest test case
- Changed function without updated tests → Verify existing tests
- Test files have meaningful assertions

**Test patterns to check**:
- Test descriptions are clear
- Tests cover edge cases
- Tests don't have hardcoded values where they shouldn't
```

### 4. Documentation Updates

Check if documentation needs updates:

```markdown
# TODO: Customize for your documentation structure

**Documentation checks**:
- New feature → Update relevant docs
- API changes → Update API documentation
- Breaking changes → Update CHANGELOG.md and migration guide
- New configuration → Update config documentation
- New dependencies → Update requirements/dependencies doc
```

### 5. Breaking Change Detection

Identify potential breaking changes:

```markdown
# TODO: Define breaking changes for your project

**Breaking change indicators**:
- Function signature changes
- Removed public APIs
- Changed config file formats
- Modified CLI arguments
- Database schema changes
- Changed API response formats

**Required when breaking change detected**:
- Label PR with "breaking-change"
- Ensure CHANGELOG.md updated
- Request migration guide
- Verify version bump is major
```

### 6. Style and Conventions

Check adherence to project conventions:

```markdown
# TODO: Customize for your project's conventions

**Convention checks**:
- File naming conventions (kebab-case, camelCase, etc.)
- Import order and organization
- Comment style (JSDoc, Python docstrings, etc.)
- Error message format
- Logging format and levels
- Commit message format
```

## Implementation Steps

### Step 1: Get PR Details

```bash
# Fetch PR information
PR_NUMBER=${{ github.event.pull_request.number }}

# Get basic PR details
gh pr view $PR_NUMBER --json title,body,additions,deletions,changedFiles,labels

# Get list of changed files
gh pr view $PR_NUMBER --json files | jq -r '.files[].path'

# Get PR diff for detailed analysis
gh pr diff $PR_NUMBER
```

### Step 2: Analyze Changes

```python
#!/usr/bin/env python3
"""
PR Analysis Script
TODO: Customize analysis logic
"""
import json
import subprocess
import re

# Load PR data
pr_data = json.loads(subprocess.check_output(['gh', 'pr', 'view', pr_number, '--json', 'title,body,files']))

issues = []

# Check 1: PR description quality
if not pr_data['body'] or len(pr_data['body']) < 50:
    issues.append({
        'category': 'description',
        'severity': 'medium',
        'message': 'PR description is too brief. Please provide more details.'
    })

# Check 2: Large PR size
total_changes = sum(f['additions'] + f['deletions'] for f in pr_data['files'])
if total_changes > 500:
    issues.append({
        'category': 'size',
        'severity': 'low',
        'message': f'Large PR with {total_changes} lines changed. Consider splitting into smaller PRs.'
    })

# Check 3: Missing tests
code_files = [f for f in pr_data['files'] if f['path'].endswith(('.py', '.js', '.go', '.ts'))]
test_files = [f for f in pr_data['files'] if 'test' in f['path'].lower()]

if code_files and not test_files:
    issues.append({
        'category': 'testing',
        'severity': 'high',
        'message': 'Code changes detected but no test files included. Please add tests.'
    })

# Check 4: Debug statements
diff_output = subprocess.check_output(['gh', 'pr', 'diff', pr_number]).decode()
debug_patterns = [r'console\.log\(', r'print\(', r'debugger;']
for pattern in debug_patterns:
    if re.search(pattern, diff_output):
        issues.append({
            'category': 'code_quality',
            'severity': 'low',
            'message': f'Debug statement found: {pattern}. Consider removing before merge.'
        })

# Save issues for comment generation
with open('/tmp/pr-review-issues.json', 'w') as f:
    json.dump(issues, f, indent=2)

print(f"Analysis complete. Found {len(issues)} issues.")
```

### Step 3: Generate Feedback

```markdown
# TODO: Customize feedback format

## 🔍 PR Review

Thanks for your contribution! Here's an automated review of this PR:

### Summary
- **Files changed**: [count]
- **Lines added**: [additions]
- **Lines deleted**: [deletions]
- **Size**: [small/medium/large/very large]

### ✅ Looks Good
[List positive aspects:
- Tests included
- Documentation updated
- Clear description
- Follows conventions
]

### ⚠️ Suggestions
[List issues found with severity and suggestions:

**High Priority**:
- Missing tests for new functionality → Please add test coverage

**Medium Priority**:
- PR description could be more detailed → Include testing steps

**Low Priority**:
- Consider extracting large function into smaller ones
]

### 📚 Resources
[Helpful links:
- [Contributing Guide](link)
- [Code Style Guide](link)
- [Testing Guidelines](link)
]

---
*Automated review by [PR Reviewer]({run_url}). Questions? Ask in comments!*
```

### Step 4: Apply Labels

```markdown
# TODO: Define labeling rules

Based on analysis, apply labels:
- "needs-tests" - if tests are missing
- "needs-docs" - if documentation is missing
- "breaking-change" - if breaking changes detected
- "large-pr" - if changes exceed threshold
- "ready-for-review" - if all checks pass
```

### Step 5: Post Comment

```markdown
Use add-comment safe-output to post the review:
- Format as markdown
- Be encouraging and constructive
- Provide specific, actionable feedback
- Include links to relevant guidelines
```

## Customization Guide

### Define File Type Patterns

```python
# TODO: Customize for your project structure

FILE_PATTERNS = {
    'frontend': ['src/components/**', 'ui/**', '*.tsx', '*.jsx'],
    'backend': ['src/api/**', 'server/**', '*.go', '*.py'],
    'tests': ['**/*test.*', '**/*.test.*', '**/tests/**'],
    'docs': ['docs/**', '*.md', 'README*'],
    'config': ['*.json', '*.yaml', '*.toml', '.github/**'],
}
```

### Set Review Thresholds

```yaml
# TODO: Adjust thresholds for your team

THRESHOLDS:
  pr_size_large: 500      # Lines changed
  pr_size_xlarge: 1000
  max_files_changed: 20
  min_description_length: 100
  test_coverage_target: 80
```

### Configure Strictness Level

```python
# TODO: Choose review strictness

STRICTNESS = "moderate"  # "strict", "moderate", or "lenient"

if STRICTNESS == "strict":
    # Require tests, docs, and detailed description
    # Block PR if issues found
elif STRICTNESS == "moderate":
    # Suggest improvements but allow merge
    # Focus on high-priority issues
else:  # lenient
    # Only flag critical issues
    # Most things are suggestions
```

### Add Custom Checks

```python
# TODO: Add project-specific checks

def check_api_versioning(files, diff):
    """Ensure API changes include version bump"""
    api_files = [f for f in files if 'api/' in f['path']]
    version_files = [f for f in files if 'version' in f['path'].lower()]
    
    if api_files and not version_files:
        return {
            'severity': 'high',
            'message': 'API files changed but version not bumped'
        }
    return None

def check_security_patterns(diff):
    """Check for common security issues"""
    issues = []
    
    # Check for hardcoded secrets
    if re.search(r'password\s*=\s*["\'].*["\']', diff):
        issues.append('Potential hardcoded password detected')
    
    # Check for SQL injection risk
    if re.search(r'execute\(["\'].*\+.*["\']', diff):
        issues.append('Potential SQL injection risk - use parameterized queries')
    
    return issues
```

## Example Output

```markdown
## 🔍 Automated PR Review

Thanks for your contribution @author! 🎉

### 📊 Summary
- **Files changed**: 8 files
- **Lines added**: 234
- **Lines deleted**: 67
- **Size**: Medium

### ✅ Looks Good
- ✓ Tests included for new functionality
- ✓ Code follows project style guidelines
- ✓ Clear and descriptive commit messages
- ✓ No obvious security issues detected

### 💡 Suggestions

**High Priority**:
- 📚 **Documentation**: Please update `docs/api.md` to reflect the new endpoint
  - Add examples of the new API usage
  - Document the response format

**Medium Priority**:
- 📝 **PR Description**: Consider adding:
  - Steps to test the changes locally
  - Screenshots/examples of the new feature in action

**Low Priority**:
- 🧹 **Code Quality**: Found a `console.log()` statement in `src/utils/logger.js:42`
  - Consider removing debug logging before merge
- 📏 **Function Length**: The `processData()` function in `src/processor.js` is 85 lines long
  - Consider splitting into smaller, more focused functions

### 🏷️ Labels Applied
- `needs-docs` - Documentation updates needed

### 📚 Helpful Resources
- [API Documentation Guide](link)
- [Code Review Checklist](link)
- [Testing Best Practices](link)

---
*Automated review by [PR Reviewer](run-url) • Please address high-priority items before merge*
```

## Advanced Features

### Integration with CI Results

```bash
# Check if CI passed
gh pr checks $PR_NUMBER --json name,status,conclusion

# Include CI status in review
if [CI failed]; then
  echo "⚠️ CI checks failed. Please fix before merge."
fi
```

### Smart Review Comments

```markdown
Post review comments on specific lines (requires GitHub API):
- Comment on the actual line with the issue
- Provide context-specific suggestions
- Link to relevant documentation
```

### Progressive Review

```markdown
On PR updates (synchronize event):
- Compare with previous review
- Only mention new issues
- Acknowledge fixed issues
```

## Related Examples

- **Production examples**:
  - `.github/workflows/dev-hawk.md` - CI monitoring and PR analysis
  - `.github/workflows/breaking-change-checker.md` - Breaking change detection
  - `.github/workflows/grumpy-reviewer.md` - Opinionated code review

## Tips

- **Be constructive**: Frame feedback positively
- **Be specific**: Point to exact files and lines
- **Be helpful**: Provide resources and examples
- **Be consistent**: Apply rules uniformly
- **Prioritize**: Focus on high-impact issues first
- **Learn**: Adjust rules based on team feedback

## Security Considerations

- This workflow reads PR content and posts comments
- Uses `strict: true` for enhanced security
- No write access to code or protected branches
- All operations validated through safe-outputs

---

**Pattern Info**:
- Complexity: Intermediate
- Trigger: Pull request events
- Safe Outputs: add_comment, add_labels
- Tools: GitHub (pull_requests, repos), bash (git, grep)
