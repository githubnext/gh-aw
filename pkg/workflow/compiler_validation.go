package workflow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// validateWorkflowSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateWorkflowSchema(yamlContent string) error {
	// Convert YAML to JSON for validation
	var workflowData interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse generated YAML: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	// Load GitHub Actions workflow schema from SchemaStore
	schemaURL := "https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json"

	// Create compiler with HTTP loader
	loader := jsonschema.NewCompiler()
	httpLoader := &httpURLLoader{
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// Configure the compiler to use HTTP loader for https and http schemes
	schemeLoader := jsonschema.SchemeURLLoader{
		"https": httpLoader,
		"http":  httpLoader,
	}
	loader.UseLoader(schemeLoader)

	schema, err := loader.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to load GitHub Actions schema from %s: %w", schemaURL, err)
	}

	// Validate the JSON data against the schema
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("workflow schema validation failed: %w", err)
	}

	return nil
}

// validateHTTPTransportSupport validates that HTTP MCP servers are only used with engines that support HTTP transport
func (c *Compiler) validateHTTPTransportSupport(tools map[string]any, engine AgenticEngine) error {
	if engine.SupportsHTTPTransport() {
		// Engine supports HTTP transport, no validation needed
		return nil
	}

	// Engine doesn't support HTTP transport, check for HTTP MCP servers
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(config); hasMcp && mcpType == "http" {
				return fmt.Errorf("tool '%s' uses HTTP transport which is not supported by engine '%s' (only stdio transport is supported)", toolName, engine.GetID())
			}
		}
	}

	return nil
}

// validateMaxTurnsSupport validates that max-turns is only used with engines that support this feature
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine AgenticEngine) error {
	// Check if max-turns is specified in the frontmatter
	_, hasMaxTurns := frontmatter["max-turns"]
	if !hasMaxTurns {
		// No max-turns specified, no validation needed
		return nil
	}

	// max-turns is specified, check if the engine supports it
	if !engine.SupportsMaxTurns() {
		return fmt.Errorf("max-turns not supported: engine '%s' does not support the max-turns feature", engine.GetID())
	}

	// Engine supports max-turns - additional validation could be added here if needed
	// For now, we rely on JSON schema validation for format checking

	return nil
}
