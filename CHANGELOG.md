# Changelog

All notable changes to this project will be documented in this file.

## v0.20.0 - 2025-10-12

### Features

#### Add --json flag to status command and jq filtering to MCP server

Adds new command-line flags to the status command:
- `--json` flag renders the entire output as JSON
- Optional `jq` parameter allows filtering JSON output through jq tool

The jq filtering functionality has been refactored into dedicated files (jq.go) with comprehensive test coverage.


### Bug Fixes

#### Fix content truncation message priority in sanitizeContent function

Fixed a bug where the `sanitizeContent` function was applying truncation checks in the wrong order. When content exceeded both line count and byte length limits, the function would incorrectly report "Content truncated due to length" instead of the more specific "Content truncated due to line count" message. The truncation logic now prioritizes line count truncation, ensuring users get the most accurate truncation message based on which limit was hit first.

#### Fix HTTP transport usage of go-sdk

Fixed the MCP server HTTP transport implementation to use the correct `NewStreamableHTTPHandler` API from go-sdk instead of the deprecated SSE handler. Also added request/response logging middleware and changed configuration validation errors to warnings to allow server startup in test environments.

#### Fix single-file artifact directory nesting in logs command

When downloading artifacts with a single file, the file is now moved to the parent directory and the unnecessary nested folder is removed. This implements the "artifact unfold rule" which simplifies artifact access by removing unnecessary nesting for single-file artifacts while preserving multi-file artifact directories.

#### Update MCP server workflow for toolset comparison with cache-memory

Enhanced the github-mcp-tools-report workflow to track and compare changes to the GitHub MCP toolset over time. Added cache-memory configuration to enable persistent storage across workflow runs, allowing the workflow to detect new and removed tools since the last report. The workflow now loads previous tools data, compares it with the current toolset, and includes a changes section in the generated report.


## v0.19.0 - 2025-10-12

### Features

#### Add validation step to mcp-server command startup

The `mcp-server` command now validates configuration before starting the server. It runs `gh aw status` to verify that the gh CLI and gh-aw extension are properly installed, and that the working directory is a valid git repository with `.github/workflows`. This provides immediate, actionable feedback to users about configuration issues instead of cryptic errors when tools are invoked.


### Bug Fixes

#### Add git patch preview in fallback issue messages

When the create_pull_request safe output handler fails to push changes or create a PR, it now includes a preview of the git patch (max 500 lines) in the fallback issue message. This improves debugging by providing immediate visibility into the changes that failed to be pushed or converted to a PR.

#### Add lockfile statistics analysis workflow for nightly audits

Adds a new agentic workflow that performs comprehensive statistical and structural analysis of all `.lock.yml` files in the repository, publishing insights to the "audits" discussion category. The workflow runs nightly at 3am UTC and provides valuable visibility into workflow usage patterns, trigger types, safe outputs, file sizes, and structural characteristics.

#### Fix false positives in error validation from environment variable dumps in logs

The audit workflow was failing due to false positives in error pattern matching. The error validation script was matching error pattern definitions that appeared in GitHub Actions logs as environment variable dumps, creating a recursive false positive issue. Added a `shouldSkipLine()` function that filters out GitHub Actions metadata lines (environment variable declarations and section headers) before validation, allowing the audit workflow to successfully parse agent logs without false positives.

#### Fix YAML boolean keyword quoting to prevent workflow validation failures

Fixed the compiler to prevent unquoting the "on" key in generated workflow YAML files. This prevents YAML parsers from misinterpreting "on" as the boolean value `True` instead of a string key, which was causing GitHub Actions workflow validation failures. The fix ensures all compiled workflows generate valid YAML that passes GitHub Actions validation.

#### Mark permission-related error patterns as warnings to reduce false positives

Permission-related error patterns were being classified as fatal errors, causing workflow runs to fail unnecessarily when encountering informational messages about permissions, authentication, or authorization. This change introduces a `Severity` field to the `ErrorPattern` struct that allows explicit override of the automatic level detection logic, enabling fine-grained control over which patterns should be treated as errors versus warnings.

Updated 26 permission and authentication-related patterns across the Codex and Copilot engines to be classified as warnings instead of errors, improving workflow reliability while maintaining visibility of permission issues for troubleshooting.


## v0.18.2 - 2025-10-11

### Bug Fixes

#### Add GitHub Copilot agent setup workflow

Adds a `.github/workflows/copilot-setup-steps.yml` workflow file to configure the GitHub Copilot coding agent environment with preinstalled tools and dependencies. The workflow mirrors the setup steps from the CI workflow's build job, including Node.js, Go, JavaScript dependencies, development tools, and build step. This provides Copilot agents with a fully configured development environment and speeds up agent workflows.

#### Add compiler validation for GitHub Actions 21KB expression size limit

The compiler now validates that expressions in generated YAML files don't exceed GitHub Actions' 21KB limit. This prevents silent failures at runtime by catching oversized environment variables and expressions during compilation. When violations are detected, compilation fails with a descriptive error message and saves the invalid YAML to `*.invalid.yml` for debugging.

#### Enhance CLI version checker workflow with comprehensive version analysis

Enhanced the CLI version checker workflow to perform deeper research summaries when updates are detected. The workflow now includes:

- Version-by-version analysis for all intermediate versions
- Categorized change tracking (breaking changes, features, bugs, security, performance)
- Impact assessment on gh-aw workflows
- Timeline analysis with release dates
- Risk assessment (Low/Medium/High)
- Enhanced research sources and methods documentation
- Improved PR description templates with comprehensive version progression documentation

This internal tooling improvement helps maintainers make more informed decisions about CLI dependency updates.

#### Fix compiler issue generating invalid lock files due to heredoc delimiter

Fixed a critical bug in the workflow compiler where using single-quoted heredoc delimiters (`<< 'EOF'`) prevented GitHub Actions expressions from being evaluated in MCP server configuration files. Changed to unquoted delimiters (`<< EOF`) to allow proper expression evaluation at runtime. This fix affects all generated workflow lock files and ensures MCP configurations are correctly populated with environment variables.

#### Move init command to pkg/cli folder

Refactored the init command structure by moving `NewInitCommand()` from `cmd/gh-aw/init.go` to `pkg/cli/init_command.go` to follow the established pattern for command organization used by other commands in the repository.

#### Remove push trigger from repo-tree-map agentic workflow

The workflow now only triggers via manual `workflow_dispatch`, preventing unnecessary automatic runs when the workflow lock file is modified.

#### Update documentation unbloater workflow with cache-memory and PR checking

Enhanced the unbloat-docs workflow to improve coordination and avoid duplicate work:
- Added cache-memory tool for persistent storage of cleanup notes across runs
- Added search_pull_requests GitHub API tool to check for conflicting PRs
- Updated workflow instructions to check cache and open PRs before selecting files to clean


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

