// Package workflow provides GitHub Actions schema validation for compiled workflows.
//
// # GitHub Actions Schema Validation
//
// This file validates that compiled workflow YAML conforms to the official
// GitHub Actions workflow schema. It uses JSON Schema validation with caching
// to avoid recompiling the schema on every validation.
//
// # Validation Functions
//
//   - validateGitHubActionsSchema() - Validates YAML against GitHub Actions schema
//   - getCompiledSchema() - Returns cached compiled schema (compiled once)
//
// # Validation Pattern: Schema Validation with Caching
//
// Schema validation uses a singleton pattern for efficiency:
//   - sync.Once ensures schema is compiled only once
//   - Schema is embedded in the binary as githubWorkflowSchema
//   - Cached compiled schema is reused across all validations
//   - YAML is converted to JSON for schema validation
//
// # Schema Source
//
// The GitHub Actions workflow schema is embedded from:
//
//	https://json.schemastore.org/github-workflow.json
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates against JSON schemas
//   - It checks GitHub Actions YAML structure
//   - It verifies workflow syntax correctness
//   - It requires schema compilation and caching
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Cached compiled schema to avoid recompiling on every validation
var (
	compiledSchemaOnce sync.Once
	compiledSchema     *jsonschema.Schema
	schemaCompileError error
)

// getCompiledSchema returns the compiled GitHub Actions schema, compiling it once and caching
func getCompiledSchema() (*jsonschema.Schema, error) {
	compiledSchemaOnce.Do(func() {
		// Parse the embedded schema
		var schemaDoc any
		if err := json.Unmarshal([]byte(githubWorkflowSchema), &schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to parse embedded GitHub Actions schema: %w", err)
			return
		}

		// Create compiler and add the schema as a resource
		loader := jsonschema.NewCompiler()
		schemaURL := "https://json.schemastore.org/github-workflow.json"
		if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to add schema resource: %w", err)
			return
		}

		// Compile the schema once
		schema, err := loader.Compile(schemaURL)
		if err != nil {
			schemaCompileError = fmt.Errorf("failed to compile GitHub Actions schema: %w", err)
			return
		}

		compiledSchema = schema
	})

	return compiledSchema, schemaCompileError
}

// validateGitHubActionsSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateGitHubActionsSchema(yamlContent string) error {
	// Convert YAML to any for JSON conversion
	var workflowData any
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse YAML for schema validation: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON for validation: %w", err)
	}

	// Get the cached compiled schema
	schema, err := getCompiledSchema()
	if err != nil {
		return err
	}

	// Validate the JSON data against the schema
	var jsonObj any
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("GitHub Actions schema validation failed: %w", err)
	}

	return nil
}
