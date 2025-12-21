package cli

import (
	"github.com/google/jsonschema-go/jsonschema"
)

// GenerateOutputSchema generates a JSON schema from a Go struct type for MCP tool outputs.
// The schema conforms to JSON Schema draft 2020-12 and draft-07.
//
// Schema generation rules:
//   - json tags define property names
//   - jsonschema tags define descriptions
//   - omitempty/omitzero mark optional fields
//   - Pointer types include null in their type array
//   - Slices allow null values (jsonschema-go v0.4.0+)
//   - PropertyOrder maintains deterministic field ordering (v0.4.0+)
//
// MCP Requirements:
//   - Tool output schemas must be objects (not arrays or primitives)
//   - All properties should have descriptions for better LLM understanding
//   - Required vs optional fields must be correctly specified
//
// Example:
//
//	type Output struct {
//	    Name string `json:"name" jsonschema:"Name of the user"`
//	    Age  int    `json:"age,omitempty" jsonschema:"Age in years"`
//	}
//	schema, err := GenerateOutputSchema[Output]()
func GenerateOutputSchema[T any]() (*jsonschema.Schema, error) {
	return jsonschema.For[T](nil)
}
