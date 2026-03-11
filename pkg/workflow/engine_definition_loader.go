// This file provides the built-in engine definition loader.
//
// Built-in engine definitions are stored as YAML files embedded in the binary.
// Each file uses a top-level "engine:" key and is validated against
// schemas/engine_definition_schema.json before the definition is registered.
//
// # Embedded Resources
//
// Engine definition files live in data/engines/*.yml and are embedded at compile
// time via the //go:embed directive below. Adding a new built-in engine requires
// only a new .yml file in that directory — no Go code changes are needed.
package workflow

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var engineDefinitionLoaderLog = logger.New("workflow:engine_definition_loader")

//go:embed data/engines/*.yml
var builtinEngineFS embed.FS

//go:embed schemas/engine_definition_schema.json
var engineDefinitionSchemaJSON []byte

// engineDefinitionFile is the on-disk wrapper that holds the engine definition
// under the top-level "engine" key.
type engineDefinitionFile struct {
	Engine EngineDefinition `yaml:"engine"`
}

// Compiled engine definition schema, initialised once.
var (
	engineDefSchemaOnce  sync.Once
	engineDefSchema      *jsonschema.Schema
	engineDefSchemaError error
)

// getEngineDefinitionSchema returns the compiled JSON schema used to validate
// engine definition YAML files.
func getEngineDefinitionSchema() (*jsonschema.Schema, error) {
	engineDefSchemaOnce.Do(func() {
		engineDefinitionLoaderLog.Print("Compiling engine definition schema (first time)")

		var schemaDoc any
		if err := json.Unmarshal(engineDefinitionSchemaJSON, &schemaDoc); err != nil {
			engineDefSchemaError = fmt.Errorf("failed to parse engine definition schema JSON: %w", err)
			return
		}

		compiler := jsonschema.NewCompiler()
		schemaURL := "https://github.com/github/gh-aw/schemas/engine_definition_schema.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			engineDefSchemaError = fmt.Errorf("failed to add engine definition schema resource: %w", err)
			return
		}

		schema, err := compiler.Compile(schemaURL)
		if err != nil {
			engineDefSchemaError = fmt.Errorf("failed to compile engine definition schema: %w", err)
			return
		}

		engineDefSchema = schema
		engineDefinitionLoaderLog.Print("Engine definition schema compiled successfully")
	})

	return engineDefSchema, engineDefSchemaError
}

// validateEngineDefinitionYAML validates raw YAML bytes against the engine definition
// JSON schema.
func validateEngineDefinitionYAML(data []byte, path string) error {
	schema, err := getEngineDefinitionSchema()
	if err != nil {
		return fmt.Errorf("engine definition schema unavailable: %w", err)
	}

	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse %s for schema validation: %w", path, err)
	}

	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("engine definition file %s failed schema validation: %w", path, err)
	}

	return nil
}

// loadBuiltinEngineDefinitions reads all *.yml files from the embedded
// data/engines/ directory, validates them against the engine definition schema,
// and returns a slice of parsed EngineDefinition values.
// It panics on parse or validation errors to surface misconfigured built-in definitions early.
func loadBuiltinEngineDefinitions() []*EngineDefinition {
	engineDefinitionLoaderLog.Print("Loading built-in engine definitions from embedded YAML files")

	var definitions []*EngineDefinition

	err := fs.WalkDir(builtinEngineFS, "data/engines", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yml" {
			return nil
		}

		data, readErr := builtinEngineFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read embedded engine file %s: %w", path, readErr)
		}

		if validErr := validateEngineDefinitionYAML(data, path); validErr != nil {
			return validErr
		}

		var wrapper engineDefinitionFile
		if parseErr := yaml.Unmarshal(data, &wrapper); parseErr != nil {
			return fmt.Errorf("failed to parse embedded engine file %s: %w", path, parseErr)
		}

		def := wrapper.Engine

		// Default runtime-id to engine id when omitted
		if def.RuntimeID == "" {
			def.RuntimeID = def.ID
		}

		engineDefinitionLoaderLog.Printf("Loaded built-in engine definition: id=%s runtime-id=%s", def.ID, def.RuntimeID)
		definitions = append(definitions, &def)
		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("failed to walk embedded engine definitions directory: %v", err))
	}

	engineDefinitionLoaderLog.Printf("Loaded %d built-in engine definitions", len(definitions))
	return definitions
}
