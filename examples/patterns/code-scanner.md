---
# Pattern: Code Scanner
# Complexity: Advanced
# Use Case: Analyze codebase for patterns, issues, or improvements
name: Code Scanner
description: Analyzes codebase for patterns, security issues, and improvement opportunities
on:
  schedule:
    # TODO: Customize scan schedule
    - cron: "0 2 * * 1"  # Weekly on Monday at 2 AM
  push:
    branches:
      # TODO: Specify branches to scan on push
      - main
  workflow_dispatch:
permissions:
  contents: read
  issues: write
  pull-requests: write
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [repos, issues, pull_requests]
  bash:
    - "git *"
    - "grep *"
    - "find *"
    - "python3 *"
safe-outputs:
  create-issue:
    max: 5
  create-pull-request:
    max: 1
timeout-minutes: 30
strict: true
---

# Code Scanner

Analyze your codebase for patterns, potential issues, security vulnerabilities, and improvement opportunities.

## Scan Types

# TODO: Choose which scans to implement

### 1. Breaking Change Detection

Scan for changes that could break backward compatibility:

```yaml
# TODO: Define breaking change patterns for your project

BREAKING_PATTERNS:
  - Removed public APIs or functions
  - Changed function signatures
  - Modified API response formats
  - Changed config file schemas
  - Removed CLI commands or flags
  - Changed database schemas
  - Modified external interfaces
```

**Output**: Create issue listing breaking changes found

### 2. Security Vulnerability Scan

Scan for common security issues:

```yaml
# TODO: Customize security patterns

SECURITY_PATTERNS:
  - Hardcoded credentials or API keys
  - SQL injection vulnerabilities
  - Cross-site scripting (XSS) risks
  - Insecure random number generation
  - Weak cryptographic algorithms
  - Unvalidated input handling
  - Path traversal vulnerabilities
```

**Output**: Create high-priority issues for each vulnerability

### 3. Code Quality Scan

Analyze code quality and maintainability:

```yaml
# TODO: Define quality metrics

QUALITY_CHECKS:
  - Functions exceeding complexity threshold
  - Duplicated code blocks
  - Dead code (unused functions/variables)
  - Missing error handling
  - Missing documentation
  - Inconsistent naming conventions
  - Code smells (long parameter lists, etc.)
```

**Output**: Create issue with quality improvement suggestions

### 4. Dependency Audit

Scan dependencies for issues:

```yaml
# TODO: Configure dependency checks

DEPENDENCY_CHECKS:
  - Outdated dependencies (>6 months old)
  - Dependencies with known vulnerabilities
  - Unused dependencies
  - Duplicate dependencies
  - Missing license information
  - Deprecated packages
```

**Output**: Create PR updating dependencies or issue listing concerns

### 5. Consistency Check

Ensure codebase follows project conventions:

```yaml
# TODO: Define consistency rules

CONSISTENCY_RULES:
  - File naming conventions
  - Import statement order
  - Error message format
  - Logging patterns
  - Test file structure
  - Documentation format
```

**Output**: Create PR fixing inconsistencies or issue listing them

## Implementation Steps

### Step 1: Clone and Analyze Repository

```bash
# Repository is already cloned in the workflow environment
cd $GITHUB_WORKSPACE

# Get file statistics
echo "=== Repository Statistics ==="
echo "Total files: $(find . -type f ! -path '*/.*' | wc -l)"
echo "Go files: $(find . -name '*.go' ! -path '*/.*' | wc -l)"
echo "Python files: $(find . -name '*.py' ! -path '*/.*' | wc -l)"
echo "JavaScript files: $(find . -name '*.js' -o -name '*.ts' ! -path '*/.*' | wc -l)"
```

### Step 2: Run Scanners

```python
#!/usr/bin/env python3
"""
Code Scanner
TODO: Implement scanners for your specific needs
"""
import os
import re
import json
from pathlib import Path
from collections import defaultdict

class CodeScanner:
    def __init__(self, repo_path):
        self.repo_path = Path(repo_path)
        self.findings = defaultdict(list)
    
    def scan_security(self):
        """Scan for security issues"""
        # TODO: Customize security patterns
        patterns = {
            'hardcoded_secrets': [
                r'password\s*=\s*["\'][^"\']+["\']',
                r'api[_-]?key\s*=\s*["\'][^"\']+["\']',
                r'secret\s*=\s*["\'][^"\']+["\']',
            ],
            'sql_injection': [
                r'execute\(["\'].*\+.*["\']',
                r'query\(["\'].*\+.*["\']',
            ],
            'path_traversal': [
                r'open\(.*\+.*\)',
                r'read_file\(.*\+.*\)',
            ]
        }
        
        for file_path in self.repo_path.rglob('*'):
            if file_path.is_file() and file_path.suffix in ['.py', '.js', '.go']:
                content = file_path.read_text()
                
                for issue_type, pattern_list in patterns.items():
                    for pattern in pattern_list:
                        matches = re.finditer(pattern, content, re.MULTILINE)
                        for match in matches:
                            line_num = content[:match.start()].count('\n') + 1
                            self.findings[issue_type].append({
                                'file': str(file_path.relative_to(self.repo_path)),
                                'line': line_num,
                                'code': match.group(0),
                                'severity': 'high'
                            })
    
    def scan_code_quality(self):
        """Scan for code quality issues"""
        # TODO: Implement quality checks
        
        for file_path in self.repo_path.rglob('*.py'):
            if not file_path.is_file():
                continue
                
            content = file_path.read_text()
            lines = content.split('\n')
            
            # Check for long functions (>50 lines)
            in_function = False
            function_start = 0
            
            for i, line in enumerate(lines):
                if line.strip().startswith('def '):
                    if in_function and (i - function_start) > 50:
                        self.findings['long_function'].append({
                            'file': str(file_path.relative_to(self.repo_path)),
                            'line': function_start,
                            'length': i - function_start
                        })
                    in_function = True
                    function_start = i
            
            # Check for missing docstrings on public functions
            for i, line in enumerate(lines):
                if line.strip().startswith('def ') and not line.strip().startswith('def _'):
                    # Check if next non-empty line is a docstring
                    next_line = lines[i + 1].strip() if i + 1 < len(lines) else ''
                    if not next_line.startswith('"""') and not next_line.startswith("'''"):
                        self.findings['missing_docstring'].append({
                            'file': str(file_path.relative_to(self.repo_path)),
                            'line': i + 1,
                            'function': line.strip()
                        })
    
    def scan_dependencies(self):
        """Scan dependency files for issues"""
        # TODO: Customize for your package managers
        
        # Python dependencies
        req_file = self.repo_path / 'requirements.txt'
        if req_file.exists():
            content = req_file.read_text()
            for line in content.split('\n'):
                if line and not line.startswith('#'):
                    # Check for unpinned versions
                    if '==' not in line:
                        self.findings['unpinned_dependency'].append({
                            'file': 'requirements.txt',
                            'dependency': line.strip(),
                            'message': 'Version not pinned'
                        })
        
        # Node.js dependencies
        package_json = self.repo_path / 'package.json'
        if package_json.exists():
            data = json.loads(package_json.read_text())
            
            # Check for outdated patterns
            deps = {**data.get('dependencies', {}), **data.get('devDependencies', {})}
            for dep, version in deps.items():
                if version.startswith('^') or version.startswith('~'):
                    self.findings['loose_dependency'].append({
                        'file': 'package.json',
                        'dependency': dep,
                        'version': version,
                        'message': 'Using loose version range'
                    })
    
    def generate_report(self):
        """Generate findings report"""
        return dict(self.findings)

# Run scanner
scanner = CodeScanner('.')
scanner.scan_security()
scanner.scan_code_quality()
scanner.scan_dependencies()

# Save findings
findings = scanner.generate_report()
with open('/tmp/scan-findings.json', 'w') as f:
    json.dump(findings, f, indent=2)

print(f"Scan complete. Found {sum(len(v) for v in findings.values())} issues across {len(findings)} categories.")
```

### Step 3: Analyze Findings

```python
#!/usr/bin/env python3
"""
Analyze and prioritize findings
"""
import json

with open('/tmp/scan-findings.json') as f:
    findings = json.load(f)

# Categorize by severity
critical = []
high = []
medium = []
low = []

for category, issues in findings.items():
    for issue in issues:
        severity = issue.get('severity', 'medium')
        
        item = {
            'category': category,
            **issue
        }
        
        if category in ['hardcoded_secrets', 'sql_injection']:
            critical.append(item)
        elif severity == 'high':
            high.append(item)
        elif severity == 'medium':
            medium.append(item)
        else:
            low.append(item)

# Save prioritized findings
report = {
    'summary': {
        'critical': len(critical),
        'high': len(high),
        'medium': len(medium),
        'low': len(low),
        'total': len(critical) + len(high) + len(medium) + len(low)
    },
    'critical': critical,
    'high': high,
    'medium': medium,
    'low': low
}

with open('/tmp/scan-report.json', 'w') as f:
    json.dump(report, f, indent=2)

print(f"Report generated: {report['summary']['total']} total findings")
```

### Step 4: Create Issues

```markdown
# TODO: Customize issue creation logic

For each finding category:

**Critical Issues** (create immediately):
- Create separate issue for each security vulnerability
- Label: "security", "critical"
- Assign to security team
- Include reproduction steps and fix suggestions

**High Priority** (create if >5 findings):
- Create single issue summarizing all high-priority findings
- Label: "code-quality", "high-priority"
- Group by category

**Medium/Low** (create weekly summary):
- Create single issue with all findings
- Label: "code-quality", "technical-debt"
- Include statistics and trends
```

### Step 5: Generate Report

```markdown
## 🔍 Code Scan Report - [Date]

### Summary

Found **[total]** issues across [categories] categories:
- 🚨 Critical: [count]
- ⚠️ High: [count]
- 💡 Medium: [count]
- ℹ️ Low: [count]

### Critical Issues

[If any critical issues found, create separate issues for each]

### Security Findings

**Hardcoded Secrets** ([count]):
- `auth.py:42` - Hardcoded API key detected
- `config.js:18` - Password in configuration file

[Fix]: Use environment variables or secret management

**SQL Injection Risks** ([count]):
- `database.py:156` - String concatenation in SQL query

[Fix]: Use parameterized queries

### Code Quality

**Long Functions** ([count] files):
- `processor.py:45` - Function is 87 lines (threshold: 50)
- `handler.go:234` - Function is 120 lines

[Fix]: Extract into smaller, focused functions

**Missing Documentation** ([count] functions):
- [count] public functions without docstrings

[Fix]: Add docstrings following project conventions

### Dependencies

**Outdated** ([count]):
- `requests` - Current: 2.25.0, Latest: 2.31.0 (18 months old)
- `express` - Current: 4.17.1, Latest: 4.18.2

[Fix]: Update dependencies and test

**Vulnerabilities** ([count]):
- `lodash@4.17.19` - CVE-2020-8203 (Prototype pollution)

[Fix]: Update to lodash@4.17.21

### Recommendations

1. **Immediate**: Address critical security issues
2. **This Week**: Review high-priority findings
3. **This Month**: Plan technical debt reduction
4. **Ongoing**: Establish code quality standards

### Trend

Compared to last scan:
- Issues decreased by 15% 📉
- Security findings: -2
- Code quality: -8

---
*Scan performed by [Code Scanner]({run_url})*
```

## Customization Guide

### Add Language-Specific Scanners

```python
# TODO: Add scanners for your languages

class GoScanner:
    def scan_error_handling(self):
        """Check for proper error handling in Go"""
        pattern = r'if err != nil'
        # Look for functions that should check errors but don't
        
    def scan_goroutine_leaks(self):
        """Check for potential goroutine leaks"""
        # Look for goroutines without proper cleanup

class JavaScriptScanner:
    def scan_async_await(self):
        """Check for missing await on promises"""
        pattern = r'\basync\s+function.*{[^}]*return\s+\w+\('
        
    def scan_console_logs(self):
        """Find console.log statements"""
        # Should be removed in production code
```

### Configure Scan Triggers

```yaml
# TODO: Choose when to run scans

# Option 1: Scheduled (recommended for comprehensive scans)
on:
  schedule:
    - cron: "0 2 * * 1"  # Weekly

# Option 2: On push to main (for quick checks)
on:
  push:
    branches: [main]

# Option 3: On PR (for diff-only scans)
on:
  pull_request:
    types: [opened, synchronize]

# Option 4: Manual (for ad-hoc scans)
on:
  workflow_dispatch:
```

### Set Thresholds

```python
# TODO: Configure thresholds for your project

THRESHOLDS = {
    'max_function_lines': 50,
    'max_file_lines': 500,
    'max_complexity': 10,
    'min_test_coverage': 80,
    'max_dependency_age_months': 12,
    'max_critical_issues': 0,
    'max_high_issues': 5,
}
```

## Example Outputs

### Security Issue Example

```markdown
## 🚨 Security: Hardcoded API Key Detected

**Severity**: Critical  
**Category**: Security  
**File**: `src/config/api.py`  
**Line**: 42

### Issue

Hardcoded API key found in source code:

```python
api_key = "sk_live_abc123xyz789"  # Line 42
```

### Risk

- API key is exposed in version control history
- Anyone with access to the repository can use the key
- Key rotation requires code changes and deployment

### Recommended Fix

1. Remove the hardcoded key from the file
2. Use environment variables:

```python
import os
api_key = os.environ.get('API_KEY')
if not api_key:
    raise ValueError('API_KEY environment variable not set')
```

3. Rotate the compromised API key
4. Store the new key in your secrets management system

### References

- [OWASP: Use of Hard-coded Credentials](link)
- [Project Secret Management Guide](link)

---
*Detected by [Code Scanner](run-url)*
```

### Code Quality Issue Example

```markdown
## 💡 Code Quality: Technical Debt Summary

**Priority**: Medium  
**Found**: 23 issues across 8 files

### Long Functions (8 files)

Functions exceeding 50 lines:

1. `process_data()` in `processor.py` - **87 lines**
   - Consider extracting: validation logic, transformation logic, saving logic

2. `handle_request()` in `api/handler.go` - **120 lines**
   - Consider extracting: request parsing, business logic, response formatting

[View all 8 files](link)

### Missing Documentation (15 functions)

Public functions without docstrings:

- `calculate_metrics()` in `analyzer.py`
- `format_output()` in `formatter.py`
- `validate_input()` in `validator.py`

[View all 15 functions](link)

### Recommendations

1. **This Sprint**: Address top 3 longest functions
2. **Next Sprint**: Add docstrings to most-used functions
3. **Ongoing**: Add pre-commit hooks to enforce standards

### Progress

Compared to last month:
- Long functions: 8 (-2) 📉
- Missing docs: 15 (-5) 📉

---
*Found by [Code Scanner](run-url)*
```

## Advanced Features

### Diff-Only Scanning

```bash
# Only scan changed files in a PR
git fetch origin main
git diff origin/main...HEAD --name-only > /tmp/changed-files.txt

# Pass to scanner
python3 scanner.py --files-from /tmp/changed-files.txt
```

### Custom Rules Engine

```yaml
# TODO: Define custom rules
custom-rules:
  - id: no-console-log
    pattern: "console\\.log\\("
    message: "Remove console.log statements"
    severity: low
    
  - id: require-error-handling
    pattern: "throw new Error"
    message: "Use custom error classes"
    severity: medium
    
  - id: no-any-type
    pattern: ": any"
    message: "Avoid 'any' type, use specific types"
    severity: medium
```

### Auto-Fix PRs

```bash
# For certain issues, create PR with fixes
if [ "$AUTO_FIX_ENABLED" = "true" ]; then
  # Run auto-fixers (linters, formatters)
  npm run lint --fix
  go fmt ./...
  
  # Create PR if changes made
  git diff --quiet || create_pr_with_fixes
fi
```

## Related Examples

- **Production examples**:
  - `.github/workflows/breaking-change-checker.md` - Breaking change detection
  - `.github/workflows/cli-consistency-checker.md` - CLI consistency checks
  - `.github/workflows/blog-auditor.md` - Content auditing

## Tips

- **Start small**: Begin with a few high-value scans
- **Tune thresholds**: Adjust based on your project's needs
- **Automate fixes**: Where possible, create PRs with fixes
- **Track trends**: Monitor improvement over time
- **Educate team**: Share findings in team meetings
- **Integrate with CI**: Run quick scans on every PR

## Security Considerations

- Scanner only reads code, doesn't modify it
- Uses `strict: true` for enhanced security
- Findings are stored temporarily, not persisted
- Sensitive findings can be marked private

---

**Pattern Info**:
- Complexity: Advanced
- Trigger: Scheduled, push, or manual
- Safe Outputs: create_issue, create_pull_request
- Tools: GitHub (repos, issues), bash (git, grep, python)
