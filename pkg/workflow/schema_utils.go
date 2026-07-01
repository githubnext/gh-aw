package workflow

import (
	"github.com/github/gh-aw/pkg/parser"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema parses schemaJSON and compiles it as a jsonschema.Schema registered
// under schemaURL.
// It is a shared helper used by all schema-compilation sites in this package to avoid
// repeating the NewCompiler → AddResource → Compile boilerplate.
func compileSchema(schemaJSON, schemaURL string) (*jsonschema.Schema, error) {
	return parser.CompileSchema(schemaJSON, schemaURL)
}
