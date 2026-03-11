// This file provides the built-in engine definition loader.
//
// Built-in engine definitions are stored as YAML files embedded in the binary.
// The loader reads each file from the embedded filesystem, parses it into an
// EngineDefinition, and registers it with the catalog.
//
// # Embedded Resources
//
// Engine definition files live in data/engines/*.yml and are embedded at compile
// time via the //go:embed directive below. Adding a new built-in engine requires
// only a new .yml file in that directory — no Go code changes are needed.
package workflow

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var engineDefinitionLoaderLog = logger.New("workflow:engine_definition_loader")

//go:embed data/engines/*.yml
var builtinEngineFS embed.FS

// loadBuiltinEngineDefinitions reads all *.yml files from the embedded
// data/engines/ directory and returns a slice of parsed EngineDefinition values.
// It panics on parse errors to surface misconfigured built-in definitions early.
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

		var def EngineDefinition
		if parseErr := yaml.Unmarshal(data, &def); parseErr != nil {
			return fmt.Errorf("failed to parse embedded engine file %s: %w", path, parseErr)
		}

		if def.ID == "" {
			return fmt.Errorf("embedded engine file %s is missing required 'id' field", path)
		}
		// ID is the only field strictly required for catalog resolution.
		// DisplayName, Description, Provider, and other fields are optional metadata
		// that enrich the catalog but are not needed for engine dispatch.

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
