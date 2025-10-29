# GitHub Agentic Workflows - Design Specifications

This directory contains design specifications and implementation documentation for key features of GitHub Agentic Workflows.

## Documents in this Directory

### 1. [Safe Output Messages Design System](./safe-output-messages.md)

**Status**: ✅ Implemented  
**Implementation**: `pkg/workflow/safe_outputs.go`, `pkg/workflow/js/safe_outputs_mcp_client.cjs`

A comprehensive design system that catalogs all messages, footers, and UI patterns used across GitHub Agentic Workflows safe output functions. This document defines:

- AI attribution footers and workflow installation instructions
- Related items references and staged mode preview messages
- Message patterns for all safe output types (issues, discussions, comments, PRs, etc.)
- Design principles: consistency, clarity, discoverability, and safety

**Key Features Documented**:
- Create Issues, Discussions, Comments, Pull Requests
- Add PR Review Comments and Update Issues
- Staged mode previews with 🎭 emoji indicator
- Patch preview messages with size limits and truncation
- Fallback messages for failure scenarios

### 2. [MCP Logs Guardrail](./MCP_LOGS_GUARDRAIL.md)

**Status**: ✅ Implemented  
**Implementation**: `pkg/cli/mcp_logs_guardrail.go`

Documentation for the output size guardrail implemented for the MCP server's `logs` command. This feature prevents overwhelming responses by:

- Checking output size before returning results (default: 12000 tokens)
- Returning schema descriptions and suggested jq queries when limit exceeded
- Supporting configurable token limits via `max_tokens` parameter

**Implementation Details**:
- Token estimation: ~4 characters per token
- Suggested jq queries for common use cases
- Test coverage in `pkg/cli/mcp_logs_guardrail_test.go`
- Integration tests in `pkg/cli/mcp_logs_guardrail_integration_test.go`

**Benefits**:
- Prevents token limit errors in AI models
- Provides actionable guidance for filtering large datasets
- Self-documenting with schema information

### 3. [YAML Version Compatibility](./yaml-version-gotchas.md)

**Status**: ✅ Documented  
**Implementation**: `pkg/workflow/compiler.go` (uses `goccy/go-yaml` v1.18.0)

Critical documentation on YAML 1.1 vs YAML 1.2 parser compatibility and their impact on GitHub Agentic Workflows. Key points:

- **YAML 1.1** treats `on`, `off`, `yes`, `no` as booleans → causes false positives
- **YAML 1.2** treats these as strings (gh-aw uses this correctly)
- GitHub Actions also uses YAML 1.2, ensuring compatibility

**Important for**:
- Workflow authors validating with Python tools
- Tool developers integrating with gh-aw
- CI/CD pipeline validation

**Recommendation**: Always use `gh aw compile` for validation, not Python's `yaml.safe_load`

## Implementation Status

All documents in this directory describe **implemented features**. These are not future plans, but rather comprehensive documentation of existing functionality.

## Related Documentation

For user-facing documentation, see the main [docs/](../docs/) directory which contains:
- User guides and tutorials
- CLI reference and MCP server documentation  
- Workflow examples and best practices

## Contributing

When adding new specifications:

1. **Document implementation details**: Include file paths and function names
2. **Mark status clearly**: Use ✅ Implemented, 🚧 In Progress, or 📋 Planned
3. **Provide examples**: Show code samples and usage patterns
4. **Link to tests**: Reference test files that verify the implementation
5. **Update this README**: Add new documents to the list above

## Maintenance

These specifications should be updated when:
- Implementation details change significantly
- New features are added to the documented systems
- Best practices or recommendations evolve
- Related issues or discussions provide new insights

---

**Last Updated**: 2025-10-29  
**Maintained by**: GitHub Next Team
