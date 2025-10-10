# Changelog

All notable changes to this project will be documented in this file.

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

