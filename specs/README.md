# GitHub Agentic Workflows - Design Specifications

This directory contains design specifications and implementation documentation for key features of GitHub Agentic Workflows.

## Specifications

| Document | Status | Implementation |
|----------|--------|----------------|
| [Safe Output Messages Design System](./safe-output-messages.md) | ✅ Implemented | `pkg/workflow/safe_outputs.go` |
| [MCP Logs Guardrail](./MCP_LOGS_GUARDRAIL.md) | ✅ Implemented | `pkg/cli/mcp_logs_guardrail.go` |
| [YAML Version Compatibility](./yaml-version-gotchas.md) | ✅ Documented | `pkg/workflow/compiler.go` |
| [Validation Architecture](./validation-architecture.md) | ✅ Documented | `pkg/workflow/validation.go` and domain-specific files |

## Related Documentation

For user-facing documentation, see [docs/](../docs/).

## Contributing

When adding new specifications:

1. Document implementation details with file paths
2. Mark status clearly: ✅ Implemented, 🚧 In Progress, or 📋 Planned
3. Provide code samples and usage patterns
4. Link to test files
5. Update this README's table

---

**Last Updated**: 2025-11-03
