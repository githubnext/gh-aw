package cli

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// GenerateOutputSchema generates a JSON schema from a Go struct type for MCP tool outputs.
// This helper uses the github.com/google/jsonschema-go package to automatically generate
// schemas from Go types, leveraging struct tags for descriptions and constraints.
//
// The schema is generated with default options which respect:
// - json tags for field names
// - jsonschema tags for descriptions and constraints
// - omitempty to mark optional fields
//
// Example usage:
//
//	type MyOutput struct {
//	    Name  string `json:"name" jsonschema:"description=Name of the item"`
//	    Count int    `json:"count,omitempty" jsonschema:"description=Number of items"`
//	}
//
//	schema, err := GenerateOutputSchema[MyOutput]()
//	tool := &mcp.Tool{
//	    Name:         "my-tool",
//	    Description:  "My tool description",
//	    OutputSchema: schema,
//	}
func GenerateOutputSchema[T any]() (*jsonschema.Schema, error) {
	// Get the type of T
	var zero T
	typ := reflect.TypeOf(zero)

	// Use jsonschema.ForType to generate schema from Go type
	schema, err := jsonschema.ForType(typ, &jsonschema.ForOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	return schema, nil
}
