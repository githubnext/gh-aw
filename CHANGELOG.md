# Changelog

All notable changes to this project will be documented in this file.

## v0.18.1 - 2025-10-11

### Bug Fixes

#### Security Fix: Allocation Size Overflow in Bash Tool Merging (Alert #7)

Fixed a potential allocation size overflow vulnerability (CWE-190) in the workflow compiler's bash tool merging logic. The fix implements input validation, overflow detection, and reasonable limits to prevent integer overflow when computing capacity for merged command arrays. This is a preventive security fix that maintains backward compatibility with no breaking changes.

#### Security Fix: Allocation Size Overflow in Domain List Merging (Alert #6)

Fixed CWE-190 (Integer Overflow or Wraparound) vulnerability in the `EnsureLocalhostDomains` function. The function was vulnerable to allocation size overflow when computing capacity for the merged domain list. The fix eliminates the overflow risk by removing pre-allocation and relying on Go's append function to handle capacity growth automatically, preventing potential denial-of-service issues with extremely large domain configurations.

#### Fixed unsafe quoting vulnerability in network hook generation (CodeQL Alert #9)

Implemented proper quote escaping using `strconv.Quote()` when embedding JSON-encoded domain data into Python script templates. This prevents potential code injection vulnerabilities (CWE-78, CWE-89, CWE-94) that could occur if domain data contained special characters. The fix uses Go's standard library for safe string escaping and adds `json.loads()` parsing in the generated Python scripts for defense in depth.

#### Refactor: Extract duplicate MCP config renderers to shared functions

Eliminated 124 lines of duplicate code by extracting MCP configuration rendering logic into shared functions. The Playwright, safe outputs, and custom MCP configuration renderers are now centralized in `mcp-config.go`, ensuring consistency between Claude and Custom engines while maintaining 100% backward compatibility.

#### Update agentic CLI versions

Updates the default versions for agentic CLIs:
- Claude Code: 2.0.13 → 2.0.14
- GitHub Copilot CLI: 0.0.338 → 0.0.339

These are patch version updates and should not contain breaking changes. Users of gh-aw will automatically use these newer versions when they are specified in workflows.


## v0.18.0 - 2025-10-11

### Features

#### Add simonw/llm CLI integration with issue triage workflow

This adds support for using the simonw/llm CLI tool as a custom agentic engine in GitHub Agentic Workflows, with a complete issue triage workflow example. The integration includes:

- A reusable shared component (`.github/workflows/shared/simonw-llm.md`) that enables any workflow to use simonw/llm CLI as its execution engine
- Support for multiple LLM providers: OpenAI, Anthropic Claude, and GitHub Models (free tier)
- Automatic configuration and plugin management
- Safe-outputs integration for GitHub API operations
- An example workflow (`issue-triage-llm.md`) demonstrating automated issue triage
- Comprehensive documentation with setup instructions and examples
- Support for both automatic triggering (on issue opened) and manual workflow dispatch


### Bug Fixes

#### Add repo-tree-map workflow for visualizing repository structure

This introduces a new agentic workflow that generates an ASCII tree map visualization of the repository file structure and publishes it as a GitHub Discussion. The workflow uses bash tools to gather repository statistics and create a formatted report with directory hierarchy, file size distributions, and repository metadata.

#### Add security-fix-pr workflow for automated security issue remediation

This adds a new agentic workflow that automatically generates pull requests to fix code security issues detected by GitHub Code Scanning. The workflow can be triggered manually via workflow_dispatch and will identify the first open security alert, analyze the vulnerability, generate a fix, and create a draft pull request for review.

#### Improve Copilot error detection to treat permission denied messages as warnings

Updated error pattern classification in the Copilot engine to correctly identify "Permission denied and could not request permission from user" messages as warnings instead of errors. This change improves error reporting accuracy and reduces false positives in workflow execution metrics.

#### Fix error pattern false positives in workflow validation

The error validation step was incorrectly flagging false positives when workflow output contained filenames or text with "error" as a substring. Updated error patterns across all AI engines (Copilot, Claude, and Codex) to use word boundaries (`\berror\b`) instead of matching any occurrence of "error", ensuring validation correctly distinguishes between actual error messages and informational text.

#### Fix import directive parsing for new {{#import}} syntax

Fixed a bug in `processIncludesWithWorkflowSpec` where the new `{{#import}}` syntax was incorrectly parsed using manual regex group extraction, causing malformed workflowspec paths. The function now uses the `ParseImportDirective` helper that correctly handles both legacy `@include` and new `{{#import}}` syntax. Also added safety checks for empty file paths and comprehensive unit tests.

#### Add security-events permission to security workflow

Fixed a permissions error in the security-fix-pr workflow that prevented it from accessing code scanning alerts. The workflow now includes the required `security-events: read` permission to successfully query GitHub's Code Scanning API for vulnerability analysis and automated fix generation.

#### Security Fix: Unsafe Quoting in Import Directive Warning (Alert #8)

Fixed unsafe string quoting in the `processIncludesWithVisited` function that could lead to potential injection vulnerabilities. The fix applies Go's `%q` format specifier to safely escape special characters in deprecation warning messages, replacing the unsafe `'%s'` pattern. This addresses CodeQL alert #8 (go/unsafe-quoting) related to CWE-78 (OS Command Injection), CWE-89 (SQL Injection), and CWE-94 (Code Injection).

#### Security fix: Prevent injection vulnerability in secret redaction YAML generation

Fixed a critical security vulnerability (CodeQL go/unsafe-quoting) where secret names containing single quotes could break out of enclosing quotes in generated YAML strings, potentially leading to command injection, SQL injection, or code injection attacks. Added proper escaping via a new `escapeSingleQuote()` helper function that sanitizes secret references before embedding them in YAML.

#### Fix XML comment removal in imported workflows and update GenAI prompt generation

- Fixed a bug where code blocks within XML comments were incorrectly preserved instead of being removed during workflow parsing
- Refactored GenAI prompt generation to use echo commands instead of sed for better readability and maintainability
- Removed the Issue Summarizer workflow
- Updated workflow trigger configurations to run on lock file changes
- Added comprehensive test suite for XML comment handling
- Simplified repository tree map workflow by reducing timeout and streamlining tool permissions

#### Update Codex remote GitHub MCP configuration to new streamable HTTP format

Updated the Codex engine's remote GitHub MCP server configuration to use the new streamable HTTP format with `bearer_token_env_var` instead of deprecated HTTP headers. This includes adding the `experimental_use_rmcp_client` flag, using the `/mcp-readonly/` endpoint for read-only mode, and standardizing on `GH_AW_GITHUB_TOKEN` across workflows. The configuration now aligns with OpenAI Codex documentation requirements.


## v0.17.0 - 2025-10-10

### Features

- Add GenAIScript shared workflow configuration and example
- Add support for GitHub toolsets configuration in agentic workflows
- Add GraphQL sub-issue linking and optional parent parameter to create-issue safe output
- Add mcp-server command to expose CLI tools via Model Context Protocol
- Add top-level `runtimes` field for runtime version overrides
- Add automatic runtime setup detection and insertion for workflow steps
- Display individual errors and warnings in audit command output
- Remove instruction file writing from compile command and remove --no-instructions flag
- Remove timeout requirement for strict mode and set default timeout to 20 minutes
- Add support for common GitHub URL formats in workflow specifications

### Bug Fixes

- Add arXiv MCP server integration to scout workflow
- Add Context7 MCP server integration to scout workflow
- Add comprehensive logging to validate_errors.cjs for infinite loop detection
- Add test coverage for shorthand write permissions in strict mode
- Add verbose logging for artifact download and metric extraction
- Add workflow installation instructions to safe output footers with enterprise support
- Add GITHUB_AW_WORKFLOW_NAME environment variable to add_reaction job
- Add cache-memory support to included workflow schema
- Configure Copilot log parsing to use debug logs from /tmp/gh-aw/.copilot/logs/
- Update duplicate finder workflow to ignore test files
- Fix copilot log parser to show tool success status instead of question marks
- Fix `mcp inspect` to apply imports before extracting MCP configurations
- Fix: Correct MCP server command in .vscode/mcp.json
- Use GITHUB_SERVER_URL instead of hardcoded https://github.com in safe output JavaScript files
- Update ast-grep shared workflow to use mcp/ast-grep docker image
- Organize MCP server shared workflows into dedicated mcp/ subdirectory with cleaner naming
- Organize temp file locations under /tmp/gh-aw/ directory
- Extract common GitHub Script step builder for safe output jobs
- Remove "Print Safe Outputs" step from generated lock files
- Remove Agentic Run Information from step summary
- Update comment message format for add_reaction job
- Update CLI version updater to support GitHub MCP server version monitoring
- Update Codex error patterns to support new Rust-based format
- Update Codex log parser to render tool calls using HTML details with 6 backticks
- Update Codex log parser to support new Rust-based format
- Update error patterns for copilot agentic engine
- Update Copilot log parser to render tool calls using HTML details with 6 backticks
- Update Copilot log parser to render tool calls with 6 backticks and structured format
- Update rendering of tools in summary tag to use HTML code elements
- Update workflow to create discussions instead of issues and adjust related configurations

## v0.15.0 - 2025-10-08

### Features

- Add PR branch checkout when pull request context is available
- Add comment creation for issue/PR reactions with workflow run links
- Add secret redaction step before artifact upload in agentic workflows
- Implement internal changeset script for version management with safety checks

### Bug Fixes

- Convert TypeScript safe output files to CommonJS and remove TypeScript compilation
- Update Claude Code CLI to version 2.0.10

