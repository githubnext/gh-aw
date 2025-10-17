# Changelog

All notable changes to this project will be documented in this file.

## v0.22.6 - 2025-10-17

### Bug Fixes

#### Updated copilot engine to use --allow-all-paths flag instead of --add-dir / for edit tool support

#### Enable reaction comments for all workflows, not just command-triggered ones

#### Fixed post-steps indentation in generated YAML workflows to match GitHub Actions schema requirements

#### Fixed empty GITHUB_AW_AGENT_OUTPUT in safe output jobs by downloading agent_output.json artifact instead of relying on job outputs

#### Fix Windows path separator issue in workflow resolution

#### Refactored prompt-step generator methods to eliminate duplicate code by introducing shared helper functions, reducing code by 33% while maintaining identical functionality

#### Refactor: Eliminate duplicate code and organize validation into npm.go and pip.go

#### Separate default list of GitHub tools for local and remote servers

#### Sort mermaid graph nodes alphabetically for stable code generation

#### Remove bloat from packaging-imports.md guide (56% reduction)

#### Add update_reaction job to update activation comments when agent fails without producing output

#### Update Claude Code to 2.0.21 and GitHub Copilot CLI to 0.0.343


## v0.22.5 - 2025-10-16

### Bug Fixes

#### Update add_reaction job to always create new comments and add comment-repo output

#### Fix version display in release binaries to show actual release tag instead of "dev"


## v0.22.4 - 2025-10-16

### Bug Fixes

#### Add generic timeout field for tools configuration. Allows configuring operation timeouts (in seconds) for tool/MCP communications in agentic engines. Supports Claude, Codex, and Copilot engines with a unified 60-second default timeout.

#### Add GH_AW_GITHUB_TOKEN secret check for GitHub remote mode in mcp inspect

#### Add GITHUB_AW_ASSETS_BRANCH normalization for upload-assets safe output

#### Reduced bloat in cache-memory documentation (56% reduction)

#### Extract duplicate custom engine step handling into shared helper functions

#### Refactor prompt-step generation to eliminate code duplication by introducing shared helper functions


## v0.22.3 - 2025-10-16

### Bug Fixes

#### Add mcp-inspect tool to mcp-server command with automatic secret validation

#### Add yq to default bash tools

#### Merge check_membership and stop-time jobs into unified pre-activation job

#### Update CLI versions: Claude Code 2.0.15→2.0.19, GitHub Copilot CLI 0.0.340→0.0.342


## v0.22.2 - 2025-10-16

### Bug Fixes

#### Update add command to resolve agentic workflow file from .github/workflows folder

#### Use HTML details/summary for threat detection prompt in step summary

#### Fixed skipped tests in compiler_test.go for MCP format migration

#### Add HTTP MCP header secret support for Copilot engine


## v0.22.1 - 2025-10-15

### Bug Fixes

#### Add Mermaid graph generation to compiled workflow lock file headers

#### Fixed safe outputs MCP server to return stringified JSON results for Copilot CLI compatibility

#### Add strict mode validation for bash tool wildcards and update documentation


## v0.22.0 - 2025-10-15

### Features

#### Add builtin "agentic-workflows" tool for workflow introspection and analysis

Adds a new builtin tool that enables AI agents to analyze GitHub Actions workflow traces and improve workflows based on execution history. The tool exposes the `gh aw mcp-server` command as an MCP server, providing agents with four powerful capabilities:

- **status** - Check compilation status and GitHub Actions state of all workflows
- **compile** - Programmatically compile markdown workflows to YAML
- **logs** - Download and analyze workflow run logs with filtering options
- **audit** - Investigate specific workflow run failures with detailed diagnostics

When enabled in a workflow's frontmatter, the tool automatically installs the gh-aw extension and configures the MCP server for all supported engines (Claude, Copilot, Custom, Codex). This enables continuous workflow improvement driven by AI analysis of actual execution data.

#### Add container image and runtime package validation to --validate flag

Enhances the `--validate` flag to perform additional validation checks beyond GitHub Actions schema validation:

- **Container images**: Validates Docker container images used in MCP configurations are accessible
- **npm packages**: Validates packages referenced with `npx` exist on the npm registry
- **Python packages**: Validates packages referenced with `pip`, `pip3`, `uv`, or `uvx` exist on PyPI

The validator provides early detection of non-existent Docker images, typos in package names, and missing dependencies during compilation, giving immediate feedback to workflow authors before runtime failures occur.

#### Add secret validation steps to agentic engines (Claude, Copilot, Codex)

Added secret validation steps to all agentic engines to fail early with helpful error messages when required API secrets are missing. This includes new helper functions `GenerateSecretValidationStep()` and `GenerateMultiSecretValidationStep()` for single and multi-secret validation with fallback logic.

#### Clear terminal in watch mode before recompiling

Adds automatic terminal clearing when files are modified in `--watch` mode, improving readability by removing cluttered output from previous compilations. The new `ClearScreen()` function uses ANSI escape sequences and only clears when stdout is a TTY, ensuring compatibility with pipes, redirects, and CI/CD environments.


### Bug Fixes

#### Add "Downloading container images" step to predownload Docker images used in MCP configs

#### Add shared agentic workflow for Microsoft Fabric RTI MCP server

Adds a new shared MCP workflow configuration for the Microsoft Fabric Real-Time Intelligence (RTI) MCP Server, enabling AI agents to interact with Fabric RTI services for data querying and analysis. The configuration provides access to Eventhouse (Kusto) queries and Eventstreams management capabilities.

#### Add if condition to custom safe output jobs to check agent output

Custom safe output jobs now automatically include an `if` condition that checks whether the safe output type (job ID) is present in the agent output, matching the behavior of built-in safe output jobs. When users provide a custom `if` condition, it's combined with the safe output type check using AND logic.

#### Add documentation for init command to CLI docs

#### Clarify that edit: tool is required for writing to files

Updated documentation in instruction files to explicitly state that the `edit:` tool is required when workflows need to write to files in the repository. This helps users understand they must include this tool in their workflow configuration to enable file writing capabilities.

#### Treat container image validation as warning instead of error

Container image validation failures during `compile --validate` are now treated as warnings instead of errors. This prevents compilation from failing due to local Docker authentication issues or private registry access problems, while still informing users about potential container validation issues.

#### Fix patch generation to handle underscored safe-output type names

The patch generation script now correctly searches for underscored type names (`push_to_pull_request_branch`, `create_pull_request`) to match the format used by the safe-outputs MCP server. This fixes a mismatch that was causing the `push_to_pull_request_branch` safe-output job to fail when looking for the patch file.

#### Update q.md workflow to use MCP server tools instead of CLI commands

The Q agentic workflow was incorrectly referencing the `gh aw` CLI command, which won't work because the agent doesn't have GitHub token access. Updated all references to explicitly use the gh-aw MCP server's `compile` tool instead.

#### Pretty print unauthorized expressions error message with line breaks

When compilation fails due to unauthorized expressions, the error message now displays each expression on its own line with bullet points, making it much easier to read and identify which expressions are valid. Previously, all expressions were displayed in a single long line that was difficult to scan.

#### Use --allow-all-tools flag for bash wildcards in Copilot engine

#### Optimize changeset-generator workflow for token efficiency

#### Reduce bloat in CLI commands documentation by 72%

Dramatically reduced bloat in CLI commands documentation from 803 lines to 224 lines while preserving all essential information. Removed excessive command examples, redundant explanations, duplicate information, and verbose descriptions to improve documentation clarity and scannability.

#### Reduced bloat in packaging-imports.md documentation (58% reduction)

#### Remove detection of missing tools using error patterns

Removed fragile error pattern matching logic that attempted to detect missing tools from log parsing infrastructure. This detection is now the exclusive responsibility of coding agents. Cleaned up 569 lines of code across Claude, Copilot, and Codex engine implementations while maintaining all error pattern functionality for legitimate use cases (counting and categorizing errors/warnings).

#### Update tool call rendering to include duration and token size

Enhanced log parsers for Claude, Codex, and Copilot to display execution duration and approximate token counts for tool calls. Adds helper functions for token estimation (using chars/token formula) and human-readable duration formatting to provide better visibility into tool execution performance and resource usage.

#### Add Playwright, upload-assets, and documentation build pipeline to unbloat-docs workflow

#### Update Claude Code to version 2.0.15


## v0.21.0 - 2025-10-14

### Features

#### Add support for discussion and discussion_comment events in command trigger

The command trigger now recognizes GitHub Discussions events, allowing agentic workflows to respond to `/mention` commands in discussions just like they do for issues and pull requests. This includes support for both `discussion` (when a discussion is created or edited) and `discussion_comment` (when a comment on a discussion is created or edited) events.

#### Add discussion support to add_reaction_and_edit_comment.cjs

The workflow script now supports GitHub Discussions events (`discussion` and `discussion_comment`), enabling agentic workflows to add reactions and comments to discussions. This extends the existing functionality that previously only supported issues and pull requests. The implementation uses GraphQL API for all discussion operations and includes comprehensive test coverage.

#### Add entrypointArgs field to container-type MCP configuration

This adds a new `entrypointArgs` field that allows specifying arguments to be added after the container image in Docker run commands. This provides greater flexibility when configuring containerized MCP servers, following the standard Docker CLI pattern where arguments can be placed before the image (via `args`) or after the image (via `entrypointArgs`).

#### Extract and display premium model information and request consumption from Copilot CLI logs

Enhanced the Copilot log parser to extract and display premium request information from agent stdio logs. Users can now see which AI model was used, whether it requires a premium subscription, any cost multipliers that apply, and how many premium requests were consumed. This information is now surfaced directly in the GitHub Actions step summary, making it easily accessible without needing to download and manually parse log files.

#### Add --json flag to logs command for structured JSON output

Reorganized the logs command to support both JSON and console output formats using the same structured data collection approach. The implementation follows the architecture pattern established by the audit command, with structured data types (LogsData, LogsSummary, RunData) and separate rendering functions for JSON and console output. The MCP server logs tool now also supports the --json flag with jq filtering capabilities.

#### Add support for multiple cache-memory configurations with array notation and optional descriptions

Implemented support for multiple cache-memory configurations with a simplified, unified array-based structure. This feature allows workflows to define multiple caches using array notation, each with a unique ID and optional description. The implementation maintains full backward compatibility with existing single-cache configurations (boolean, nil, or object notation).

Key features:
- Unified array structure for all cache configurations
- Support for multiple caches with explicit IDs
- Optional description field for each cache
- Backward compatibility with existing workflows
- Smart path handling for single cache with ID "default"
- Duplicate ID validation at compile time
- Import support for shared workflows

#### Reorganize audit command with structured output and JSON support

Added `--json` flag to the audit command for machine-readable output. Enhanced audit reports with comprehensive information including per-job durations, file sizes with descriptions, and improved error/warning categorization. Updated MCP server integration to use JSON output for programmatic access.

Key improvements:
- New `--json` flag for structured JSON output
- Per-job duration tracking from GitHub API
- Enhanced file information with sizes and intelligent descriptions
- Better error and warning categorization
- Dual rendering: human-readable console tables or machine-readable JSON
- MCP server now returns structured JSON instead of console-formatted text

#### Update status command JSON output structure

The status command with --json flag now:
- Replaces `agent` field with `engine_id` for clarity
- Removes `frontmatter` and `prompt` fields
- Adds `on` field from workflow frontmatter to show trigger configuration

#### Add workflow run logs download and extraction to audit/logs commands

The `gh aw logs` and `gh aw audit` commands now automatically download and extract GitHub Actions workflow run logs in addition to artifacts, providing complete audit trail information by including the actual console output from workflow executions. The implementation includes security protection against zip slip vulnerability and graceful error handling for missing or expired logs.


### Bug Fixes

#### Add Datadog MCP shared workflow configuration

Adds a new shared Datadog MCP server configuration at `.github/workflows/shared/mcp/datadog.md` that enables agentic workflows to interact with Datadog's observability and monitoring platform. The configuration provides 10 tools for comprehensive Datadog access including monitors, dashboards, metrics, logs, events, and incidents with container-based deployment and multi-region support.

#### Add JSON schema helper for MCP tool outputs

Implements a reusable `GenerateOutputSchema[T]()` helper function that generates JSON schemas from Go structs using `github.com/google/jsonschema-go`. Enhanced MCP tool documentation by inlining schema information in tool descriptions for better LLM discoverability. Added comprehensive unit and integration tests for schema generation.

#### Add Sentry MCP Integration for Agentic Workflows (Read-Only)

Adds comprehensive Sentry MCP integration to enable agentic workflows to interact with Sentry for application monitoring and debugging. The integration provides 14 read-only Sentry tools including organization/project management, release management, issue/event analysis, AI-powered search, and documentation access. Configuration is available as a shared MCP setup at `.github/workflows/shared/mcp/sentry.md` that can be imported into any workflow for safe, non-destructive monitoring operations.

#### Add SST OpenCode shared agentic workflow and smoke test

Added support for SST OpenCode as a custom agentic engine with:
- Shared workflow configuration at `.github/workflows/shared/opencode.md`
- Smoke test workflow for validation
- Test workflow example
- Documentation for customizing environment variables (agent version and AI model)
- Simplified 2-step workflow (Install and Run) with direct prompt reading

#### Apply struct-based rendering to status command

Refactored the `status` command to use the struct tag-based console rendering system, following the guidelines in `.github/instructions/console-rendering.instructions.md`. The change reduces code duplication by eliminating manual table construction and improves maintainability by defining column headers once in struct tags. JSON output continues to work exactly as before.

#### Remove bloat from coding-development.md documentation

Cleaned up the coding and development workflows documentation by eliminating repetitive bullet structures and converting 12 bullet points to concise prose descriptions. This change preserves all essential information while reducing the file size by 35% and improving readability.

#### Extract shared engine installation and permission error helpers

Refactors engine-specific implementations to eliminate ~165 lines of duplicated code by extracting shared installation scaffolding and permission error handling into reusable helper functions. Creates `BuildStandardNpmEngineInstallSteps()` and permission error detection helpers, maintaining backward compatibility with no breaking changes.

#### Fix logs command to fetch all runs when date filters are specified

The `logs` command's `--count` parameter was limiting the number of logs downloaded, not the number of matching logs returned after filtering. This caused incomplete results when using date filters like `--start-date -24h`.

Modified the algorithm to always limit downloads inline based on remaining count needed, ensuring the count parameter correctly limits the final output after applying all filters. Also increased the default count from 20 to 100 for better coverage and updated documentation to clarify the behavior.

#### Fix threat detection CLI overflow by using file access instead of inlining agent output

The threat detection job was passing the entire agent output to the detection agent via environment variables, which could cause CLI argument overflow errors when the agent output was large. Modified the threat detection system to use a file-based approach where the agent reads the output file directly using bash tools (cat, head, tail, wc, grep, ls, jq) instead of inlining the full content into the prompt.

#### Fix: Add setup-python dependency for uv tool in workflow compilation

The workflow compiler now correctly adds the required `setup-python` step when the `uv` tool is detected via MCP server configurations. Previously, the runtime detection system would skip all runtime setup when ANY setup action existed in custom steps, causing workflows using `uv` or `uvx` commands to fail.

The fix refactors runtime detection to:
- Always run runtime detection and process all sources
- Automatically inject Python as a dependency when uv is detected
- Selectively filter out only runtimes that already have setup actions, rather than skipping all detection

#### Add GitHub Actions workflow commands error pattern detector

Adds support for detecting common GitHub Actions workflow command error syntax (::error, ::warning, ::notice) across all agentic engines. This improves error detection for GitHub Actions workflows by recognizing standard workflow command formats.

#### Merge "Create prompt" and "Print prompt to step summary" workflow steps

Consolidates the prompt generation workflow by moving the "Print prompt to step summary" step to appear immediately after prompt creation, making the workflow more logical and easier to understand. The functionality remains identical - this is purely a reorganization for better code structure.

#### Fix Copilot MCP configuration tools field population

Updates the `renderGitHubCopilotMCPConfig` function to correctly populate the "tools" field in MCP configuration based on allowed tools from the configuration. Adds helper function `getGitHubAllowedTools` to extract allowed tools and defaults to `["*"]` when no allowed list is specified.

#### Refactor: Extract duplicate safe-output environment setup logic into helper functions

Extracted duplicated safe-output environment setup code from multiple workflow engines and job builders into reusable helper functions in `pkg/workflow/safe_output_helpers.go`. This eliminates ~123 lines of duplicated code across 4 engine implementations and 5 safe-output job builders, improving maintainability and consistency while maintaining 100% backward compatibility.

#### Remove workflow cancellation API calls from compiler

The compiler no longer uses the GitHub Actions cancellation API. Workflow cancellation is now handled through job dependencies and `if` conditions, resulting in a cleaner architecture. This removes the need for `actions: write` permission in the `add_reaction` job and eliminates 125 lines of legacy code.

#### Rename check-membership job to check_membership with constant

Refactored the check-membership job name to use underscores (check_membership) for consistency with Go naming conventions. Introduced CheckMembershipJobName constant in constants.go to centralize the job name and eliminate hardcoded strings throughout the codebase. Updated all references including step IDs, job dependencies, step outputs, tests, and recompiled all workflow files.

#### Add rocket reaction to Q workflow

Changed the Q agentic workflow optimizer to use a rocket emoji (🚀) reaction instead of the default "eyes" (👀) reaction when triggered via `/q` comments. The rocket emoji better represents Q's mission as a workflow optimizer and performance enhancer.

#### Replace channel_id input with GH_AW_SLACK_CHANNEL_ID environment variable in Slack shared workflow

Updates the Slack shared workflow to use a required environment variable `GH_AW_SLACK_CHANNEL_ID` instead of accepting the channel ID as a `channel_id` input parameter. This simplifies the interface and aligns with best practices for configuration management. Workflows using the Slack shared workflow will need to set `GH_AW_SLACK_CHANNEL_ID` as an environment variable or repository variable instead of passing `channel_id` as an input.

#### Refactor logs command to use struct-based console rendering system

Updated the logs command to use the same struct-based rendering approach as the audit command, improving code maintainability and consistency. All data structures now use unified types for both console and JSON output with proper struct tags.

#### Add temporary folder usage instructions to agentic workflow prompts

Agentic workflows now include explicit instructions for AI agents to use `/tmp/gh-aw/agent/` for temporary files instead of the root `/tmp/` directory. This improves file organization and prevents conflicts between workflow runs.

#### Update GitHub Copilot CLI to version 0.0.340 and implement ${} syntax for MCP environment variables

This update upgrades the GitHub Copilot CLI from version 0.0.339 to 0.0.340 and implements the breaking change for MCP server environment variable configuration. The safe-outputs MCP server now uses the new `${VAR}` syntax for environment variable references instead of direct variable names.


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

