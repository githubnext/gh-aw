---
title: Security Scanner Workflow Example
name: Security Scanner
description: Scan repositories for security vulnerabilities
on:
  schedule:
    - cron: "0 9 * * 1" # Every Monday at 9 AM
permissions:
  contents: read
  security-events: write
safe-outputs:
  create-issue:
    max: 5
  add-comment:
    max: 3
engine: copilot
---

# Security Scanner

Scan the repository for security vulnerabilities and create issues for any findings.

## Instructions

1. Run security scans using available tools
2. Identify vulnerabilities by severity (critical, high, medium, low)
3. For each critical or high-severity vulnerability:
   - Create an issue with:
     - Title: "[Security] <Vulnerability Name> in <Component>"
     - Description including:
       - Severity level
       - Affected component/file
       - CVE ID (if available)
       - Recommended fix
       - References and resources
     - Labels: security, <severity-level>
     - Body should include tracker-id marker for campaign discovery
4. For medium and low-severity findings:
   - Group similar findings into a single issue
   - Include all details in the issue description
5. Add comments to existing security issues if new information is discovered

## Output Format

When creating issues, always include the tracker-id in the issue body:

```
tracker-id: security-scanner
```

This allows campaign orchestrators to discover and track the work items you create.

## Example Issue

**Title**: [Security] SQL Injection vulnerability in user authentication

**Body**:
```markdown
## Vulnerability Details

**Severity**: High
**CVE**: CVE-2025-12345
**Component**: `src/auth/login.js`
**Line**: 42-45

## Description

SQL injection vulnerability in user authentication logic allows attackers to bypass authentication by injecting malicious SQL code.

## Recommended Fix

Use parameterized queries instead of string concatenation:

\```javascript
const query = 'SELECT * FROM users WHERE username = ? AND password = ?';
db.query(query, [username, hashedPassword]);
\```

## References

- https://cwe.mitre.org/data/definitions/89.html
- https://owasp.org/www-community/attacks/SQL_Injection

---
tracker-id: security-scanner
```
