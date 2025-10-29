# JSON Schema Design Documentation

This directory contains JSON Schema files for GitHub Agentic Workflows validation.

## Schema Files

- **`main_workflow_schema.json`**: Main workflow frontmatter schema
- **`included_file_schema.json`**: Schema for imported/included workflow files
- **`mcp_config_schema.json`**: MCP server configuration schema

## Schema Design Patterns

### Fields with `oneOf` but No Top-Level Description

Some fields in the schema intentionally omit top-level `description` properties when using `oneOf` patterns. This is by design.

**Fields following this pattern:**
- `runs-on`
- `concurrency`

**Rationale**: These fields use `oneOf` to support multiple valid types (string, array, object). Each variant in the `oneOf` array provides its own specific description that accurately describes that particular type's purpose and usage. Adding a top-level description would be redundant or overly generic, as it would need to encompass all possible variants.

**Example - `runs-on` field:**
```json
{
  "runs-on": {
    "oneOf": [
      {
        "type": "string",
        "description": "Runner type as string"
      },
      {
        "type": "array",
        "description": "Runner type as array",
        "items": { "type": "string" }
      },
      {
        "type": "object",
        "description": "Runner type as object",
        "properties": {
          "group": {
            "type": "string",
            "description": "Runner group name for self-hosted runners"
          },
          "labels": {
            "type": "array",
            "description": "List of runner labels for self-hosted runners",
            "items": { "type": "string" }
          }
        }
      }
    ]
  }
}
```

Each variant provides a clear, type-specific description. A top-level description like "Runner configuration" would be less useful than the variant-specific descriptions that explain what each type represents.

**Example - `concurrency` field:**
```json
{
  "concurrency": {
    "oneOf": [
      {
        "type": "string",
        "description": "Simple concurrency group name to prevent multiple runs. Agentic workflows automatically generate enhanced concurrency policies."
      },
      {
        "type": "object",
        "description": "Concurrency configuration object with group isolation and cancellation control",
        "properties": {
          "group": {
            "type": "string",
            "description": "Concurrency group name. Workflows in the same group cannot run simultaneously."
          },
          "cancel-in-progress": {
            "type": "boolean",
            "description": "Whether to cancel in-progress workflows in the same concurrency group when a new one starts"
          }
        }
      }
    ]
  }
}
```

The string variant describes the simple use case, while the object variant describes the advanced configuration. A single top-level description cannot capture both use cases as clearly.

### Schema Validation

The schema files are embedded in the Go binary using `//go:embed` directives. After modifying any schema file, you must rebuild the binary:

```bash
make build
```

### Testing

Run schema validation tests:

```bash
make test-unit
```

## References

- [JSON Schema Specification](https://json-schema.org/)
- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [GitHub Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)
