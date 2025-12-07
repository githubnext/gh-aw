# GitHub Agentic Workflows (gh-aw) Constitution

## Core Principles

### I. Go-First Architecture
Every feature is built using Go as the primary language. The codebase follows Go best practices and idioms. Code organization prefers many smaller files grouped by functionality over large monolithic files.

### II. Minimal Changes Philosophy
When making changes, the smallest possible modifications should be made to achieve the goal. Surgical and precise changes are preferred. Never delete or remove working code unless absolutely necessary or when fixing security vulnerabilities.

### III. Test-Driven Development (NON-NEGOTIABLE)
- Unit tests must be written for all new functionality
- Tests should be consistent with existing test patterns in the repository
- Run targeted tests during development; full test suite only at completion
- Integration tests verify command behavior and binary compilation
- Manual validation is required after changes

### IV. Console Output Standards
All user-facing CLI output must use the console formatting package (`github.com/githubnext/gh-aw/pkg/console`). Never use plain `fmt.*` for CLI output. All logging must go to `os.Stderr`, never to `stdout`. Use the logger package for debug logging with proper namespacing.

### V. Workflow Compilation
All workflow markdown files must be compiled to YAML lock files using `make recompile` before committing. Lock files are NOT build artifacts and must be tracked in git. Schema changes require rebuilding the binary with `make build`.

### VI. Build & Test Discipline
- Always run `make agent-finish` before final commits (runs build, test, recompile, fmt, lint)
- Use `make test-unit` for fast iteration during development
- Use `make test` for complete validation before completion
- Format code with `make fmt` before linting
- Never cancel long-running build processes

### VII. Security & Quality
- Always validate changes don't introduce security vulnerabilities
- Run CodeQL checker before completion
- Fix vulnerabilities related to your changes
- Use `gh-advisory-database` for dependency vulnerability checking
- Store important codebase facts using the memory system

## GitHub Actions Integration

### JavaScript Code Standards
For JavaScript files in `pkg/workflow/js/*.cjs`:
- Use GitHub Actions `core.*` methods (not console.log)
- Avoid `any` type; use specific types or `unknown`
- Run `make js` and `make lint-cjs` for validation
- Follow the shared helper pattern for common operations

### Workflow Security
- Safe-outputs provide secure PR/issue operations
- MCP servers enable tool access with proper authentication
- Network access follows allowlist patterns
- Secrets are never committed to source code

## Development Workflow

### Repository-Specific Tools
- Custom agents are specialized for specific tasks - always prefer delegation when available
- Use ecosystem tools (scaffolding, package managers) to automate tasks
- Run linters, builds, and tests frequently and iteratively
- Document changes only when directly related to modifications

### Git Workflow
- Use `report_progress` tool for commits and pushes (never use git commands directly)
- Include issue numbers in PR titles when fixing issues
- Use conventional commit messages
- Review files committed and use `.gitignore` to exclude build artifacts

### Code Organization
- Prefer many smaller files grouped by functionality
- Use `any` instead of `interface{}` (Go 1.18+)
- Add new files for new features rather than extending existing ones
- Extract shared logic into helper functions

## Governance

This constitution supersedes all other practices and guides all development decisions. When in doubt, consult AGENTS.md for additional guidance on tools, workflows, and repository-specific patterns.

**Version**: 1.0.0 | **Ratified**: 2025-12-07 | **Last Amended**: 2025-12-07
