---
title: Dependency Management
description: Managing and analyzing project dependencies with gh aw deps
---

The `gh aw deps` command provides comprehensive dependency management and health analysis tools to help maintain a secure, stable, and up-to-date dependency tree.

## Overview

The dependency management system helps you:

- **Monitor dependency health** - Track v0.x (unstable) dependency ratios
- **Identify outdated packages** - Find dependencies with available updates
- **Check for security vulnerabilities** - Scan against GitHub Advisory database
- **Generate compliance reports** - Create detailed health reports in human or JSON format

## Commands

### `gh aw deps health`

Show dependency health metrics and analysis.

**Usage:**
```bash
gh aw deps health
```

**Output includes:**
- Total dependency count (direct and indirect)
- Version breakdown (v0.x unstable, v1.x stable, v2+ mature)
- Health assessment against target thresholds
- Quick status on outdated and vulnerable dependencies

**Example:**
```bash
$ gh aw deps health

=== Dependency Health Analysis ===

Total dependencies: 277 (22 direct, 255 indirect)

Version breakdown:
  v0.x (unstable): 131 (47.3%)
  v1.x (stable):   114 (41.2%)
  v2+ (mature):    25 (9.0%)

Health assessment:
  ℹ️  Moderate unstable dependency ratio (47.3% v0.x)
      Target: < 30% for improved stability

Outdated dependencies: 5 (23% of direct deps)
```

### `gh aw deps outdated`

List direct dependencies with available updates.

**Usage:**
```bash
gh aw deps outdated [--verbose]
```

**Options:**
- `--verbose`, `-v` - Show detailed progress information

**Output includes:**
- Module name
- Current version
- Latest available version
- Age of the latest version
- Status (with ⚠️ marker for v0.x dependencies)

**Example:**
```bash
$ gh aw deps outdated

Outdated Dependencies
=====================

Module                                  Current      Latest       Age        Status
-------------------------------------------------------------------------------------
github.com/charmbracelet/lipgloss       v1.1.0       v1.2.0       3 months   Update available
github.com/google/jsonschema-go         v0.4.0       v0.4.2       3 days     Update available ⚠️ v0.x

Summary: 2 of 22 dependencies outdated (9%)
```

### `gh aw deps security`

Check dependencies for known security vulnerabilities.

**Usage:**
```bash
gh aw deps security [--verbose]
```

**Options:**
- `--verbose`, `-v` - Show detailed API query progress

**Output includes:**
- Severity level (critical, high, medium, low)
- CVE identifier
- Vulnerability summary
- Fixed versions
- Advisory URL

**Example:**
```bash
$ gh aw deps security

Checking for security vulnerabilities...

✅ No known vulnerabilities
```

**Exit codes:**
- `0` - No vulnerabilities found
- `1` - One or more vulnerabilities found

### `gh aw deps report`

Generate a comprehensive dependency health report.

**Usage:**
```bash
gh aw deps report [--verbose] [--json]
```

**Options:**
- `--verbose`, `-v` - Show detailed progress information
- `--json` - Output report in JSON format

**Output sections:**
1. **Summary** - Total counts and percentages
2. **Outdated Dependencies** - Detailed update information
3. **Security Status** - Vulnerability scan results
4. **Dependency Maturity** - Version distribution analysis
5. **Recommendations** - Actionable improvement suggestions

**Example (Human-readable):**
```bash
$ gh aw deps report

═══════════════════════════════════════
  Dependency Health Report
═══════════════════════════════════════

Summary
-------
Total dependencies: 277 (22 direct, 255 indirect)
Outdated: 5 (23%)
Security advisories: 0
v0.x dependencies: 131 (47%) ⚠️

Outdated Dependencies
---------------------
[... detailed list ...]

Security Status
---------------
✅ No known vulnerabilities

Dependency Maturity
-------------------
v0.x (unstable): 131 (47%) ⚠️
v1.x (stable): 114 (41%)
v2+ (mature): 25 (9%)

Recommendations
---------------
📦 Update 5 outdated dependencies
⚠️  Reduce v0.x exposure from 47% to <30%
```

**Example (JSON format):**
```bash
$ gh aw deps report --json
```

```json
{
  "summary": {
    "total_dependencies": 277,
    "direct_dependencies": 22,
    "indirect_dependencies": 255,
    "outdated_count": 5,
    "outdated_percentage": 22.7,
    "security_advisories": 0,
    "v0_count": 131,
    "v0_percentage": 47.3,
    "v1_count": 114,
    "v1_percentage": 41.2,
    "v2_count": 25,
    "v2_percentage": 9.0
  },
  "outdated": [...],
  "security": [],
  "maturity": {...},
  "recommendations": [
    "Update 5 outdated dependencies",
    "Reduce v0.x exposure from 47% to <30%"
  ]
}
```

## License Compliance

The project also includes license compliance checking via Make targets:

```bash
# Check for license compliance violations
make license-check

# Generate CSV license report
make license-report
```

The license checker validates dependencies against disallowed license types:
- `forbidden` - Licenses that cannot be used
- `reciprocal` - Copyleft licenses requiring source distribution
- `restricted` - Licenses with usage restrictions
- `unknown` - Dependencies without identifiable licenses

## CI/CD Integration

Use the JSON output format for automated dependency monitoring:

```yaml
- name: Check dependency health
  run: |
    gh aw deps report --json > deps-report.json
    
- name: Fail on critical issues
  run: |
    gh aw deps security || echo "Security vulnerabilities found!"
```

## Best Practices

1. **Monitor v0.x ratio** - Target: &lt;30% unstable dependencies
2. **Regular updates** - Check `deps outdated` weekly
3. **Security scanning** - Run `deps security` before releases
4. **License compliance** - Verify with `make license-check`
5. **Track progress** - Use `deps report --json` for trending

## Related Tools

- **go-licenses** - License compliance scanning (included in tools.go)
- **govulncheck** - Official Go vulnerability scanner
- **Dependabot** - Automated dependency updates

## See Also

- [Environment Variables](./environment-variables.md) - Configuration options
- [Troubleshooting](../troubleshooting/) - Common issues and solutions
