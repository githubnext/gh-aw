---
# Semgrep MCP Server
# Container-based MCP server for static analysis and security scanning
#
# Documentation: https://semgrep.dev/
# MCP Server: https://github.com/semgrep/semgrep
# Docker Image: https://hub.docker.com/r/semgrep/semgrep
#
# Available tools:
#   - semgrep_rule_schema: Get the schema for writing Semgrep rules
#   - get_supported_languages: List languages supported by Semgrep
#   - semgrep_scan: Scan code files for security vulnerabilities and bugs
#   - semgrep_scan_local: Perform scans on local files and return JSON results
#   - semgrep_scan_with_custom_rule: Run a scan using a custom rule on provided code
#   - semgrep_findings: Fetch findings from the Semgrep AppSec Platform API
#
# Usage:
#   imports:
#     - shared/mcp/semgrep.md

mcp-servers:
  semgrep:
    container: "semgrep/semgrep"
    version: "latest"
    entrypointArgs: ["semgrep", "mcp", "-t", "stdio"]
    allowed: ["*"]
---

<!--

# Semgrep MCP Server
# Static analysis and security scanning tool

Semgrep is a fast, open-source static analysis tool for finding bugs, detecting security vulnerabilities, and enforcing code standards. It supports multiple languages and custom rules.

Documentation: https://semgrep.dev/

## Available Tools

The Semgrep MCP server provides the following tools for code analysis:

- semgrep_rule_schema: Retrieves the schema required to write a Semgrep rule, lists available fields, and verifies rule syntax
- get_supported_languages: Lists all programming languages supported by Semgrep
- semgrep_scan: Scans code files for security vulnerabilities, bugs, and code quality issues using default rules
- semgrep_scan_local: Performs scans on local files and returns results in JSON format
- semgrep_scan_with_custom_rule: Runs a scan using a custom rule on provided code and returns findings
- semgrep_findings: Fetches findings from the Semgrep AppSec Platform API (requires Semgrep Cloud authentication)

## Basic Usage

To use Semgrep in your workflow, simply import this shared configuration:

```yaml
imports:
  - shared/mcp/semgrep.md
```

The MCP server will enable the agent to perform static analysis on code, including:
- Security vulnerability detection
- Bug finding
- Code quality checks
- Custom rule validation
- Language-specific pattern matching

## Example Patterns

Security Scanning:
```
Use semgrep to scan the repository for SQL injection vulnerabilities
```

Custom Rule Testing:
```
Test this custom Semgrep rule against the code in src/auth.go
```

Language Support:
```
Check which languages are supported by Semgrep for our multi-language repository
```

## More Information

- Official Documentation: https://semgrep.dev/docs/
- Rule Examples: https://semgrep.dev/explore
- Writing Rules: https://semgrep.dev/docs/writing-rules/overview
- Supported Languages: https://semgrep.dev/docs/supported-languages
- Docker Image: https://hub.docker.com/r/semgrep/semgrep

-->
