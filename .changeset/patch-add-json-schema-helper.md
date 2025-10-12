---
"gh-aw": patch
---

Add JSON schema helper for MCP tool outputs

Implements a reusable `GenerateOutputSchema[T]()` helper function that generates JSON schemas from Go structs using `github.com/google/jsonschema-go`. Enhanced MCP tool documentation by inlining schema information in tool descriptions for better LLM discoverability. Added comprehensive unit and integration tests for schema generation.
