// This file provides the built-in engine definition loader.
//
// Built-in engine definitions are stored as shared agentic workflow Markdown files
// embedded in the binary. Each file uses YAML frontmatter with a top-level "engine:"
// key and is validated against schemas/engine_definition_schema.json before parsing.
//
// # Embedded Resources
//
// Engine Markdown files live in data/engines/*.md and are embedded at compile time
// via the //go:embed directive below. Adding a new built-in engine requires only a
// new .md file in that directory — no Go code changes are needed.
//
// # Builtin Virtual FS
//
// Each embedded .md file is also registered in the parser's builtin virtual FS under
// the path "@builtin:engines/<id>.md". This allows the compiler to inject the file
// as an import when the short-form "engine: <id>" is encountered.
package workflow

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var engineDefinitionLoaderLog = logger.New("workflow:engine_definition_loader")

//go:embed data/engines/*.md
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

// extractMarkdownFrontmatterYAML extracts the YAML content between the first pair of
// "---" delimiters in a Markdown file. Both LF and CRLF line endings are supported.
func extractMarkdownFrontmatterYAML(content []byte) ([]byte, error) {
	s := string(content)
	const sep = "---"

	// Find the opening delimiter
	start := strings.Index(s, sep)
	if start == -1 {
		return nil, fmt.Errorf("no frontmatter opening delimiter found")
	}
	s = s[start+len(sep):]

	// Find the closing delimiter, supporting both LF and CRLF line endings.
	endLF := strings.Index(s, "\n"+sep)
	endCRLF := strings.Index(s, "\r\n"+sep)

	end := -1
	switch {
	case endLF >= 0 && endCRLF >= 0:
		if endLF < endCRLF {
			end = endLF
		} else {
			end = endCRLF
		}
	case endLF >= 0:
		end = endLF
	case endCRLF >= 0:
		end = endCRLF
	}

	if end == -1 {
		return nil, fmt.Errorf("no frontmatter closing delimiter found")
	}
	return []byte(strings.TrimSpace(s[:end])), nil
}

// builtinEnginePath returns the canonical builtin virtual-FS path for an engine id.
func builtinEnginePath(engineID string) string {
	return parser.BuiltinPathPrefix + "engines/" + engineID + ".md"
}

// loadBuiltinEngineDefinitions reads all *.md files from the embedded data/engines/
// directory, validates their frontmatter against the engine definition schema, parses
// each EngineDefinition, and registers the file content in the parser's builtin virtual FS.
// It panics on parse or validation errors to surface misconfigured built-in definitions early.
func loadBuiltinEngineDefinitions() []*EngineDefinition {
	engineDefinitionLoaderLog.Print("Loading built-in engine definitions from embedded Markdown files")

	var definitions []*EngineDefinition

	err := fs.WalkDir(builtinEngineFS, "data/engines", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		data, readErr := builtinEngineFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read embedded engine file %s: %w", path, readErr)
		}

		// Extract the frontmatter YAML from the Markdown file.
		frontmatterYAML, fmErr := extractMarkdownFrontmatterYAML(data)
		if fmErr != nil {
			return fmt.Errorf("failed to extract frontmatter from %s: %w", path, fmErr)
		}

		// Validate the frontmatter YAML against the engine definition schema.
		if validErr := validateEngineDefinitionYAML(frontmatterYAML, path); validErr != nil {
			return validErr
		}

		// Parse the engine definition from the frontmatter.
		var wrapper engineDefinitionFile
		if parseErr := yaml.Unmarshal(frontmatterYAML, &wrapper); parseErr != nil {
			return fmt.Errorf("failed to parse embedded engine file %s: %w", path, parseErr)
		}

		def := wrapper.Engine

		// Default runtime-id to engine id when omitted.
		if def.RuntimeID == "" {
			def.RuntimeID = def.ID
		}

		// Register the full .md content in the parser's builtin virtual FS so the
		// file can be resolved and read during import processing.
		parser.RegisterBuiltinVirtualFile(builtinEnginePath(def.ID), data)

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
