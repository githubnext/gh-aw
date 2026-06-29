package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema compiles a JSON schema from a JSON string and caches it at the given URL.
// It is a shared helper used by all schema-compilation sites in this package to avoid
// repeating the NewCompiler → AddResource → Compile boilerplate.
func compileSchema(schemaJSON, schemaURL string) (*jsonschema.Schema, error) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON for %s: %w", schemaURL, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource %s: %w", schemaURL, err)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema %s: %w", schemaURL, err)
	}

	return schema, nil
}
