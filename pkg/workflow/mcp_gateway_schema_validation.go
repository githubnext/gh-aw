// Package workflow provides MCP Gateway configuration schema validation.
//
// # MCP Gateway Configuration Schema Validation
//
// This file validates that generated MCP gateway configurations conform to the
// MCP Gateway Configuration schema before being sent to the gateway script.
// It ensures the compiler generates correct configurations.
//
// # Validation Functions
//
//   - ValidateMCPGatewayConfig() - Validates gateway config JSON against schema
//   - getCompiledMCPGatewaySchema() - Returns cached compiled schema
//
// # Validation Pattern: Schema Validation with Caching
//
// Schema validation uses a singleton pattern for efficiency:
//   - sync.Once ensures schema is compiled only once
//   - Schema is embedded in the binary as mcpGatewayConfigSchema
//   - Cached compiled schema is reused across all validations
//   - JSON is validated directly against the schema
//
// # Schema Source
//
// The MCP Gateway configuration schema is embedded from:
//
//	docs/public/schemas/mcp-gateway-config.schema.json
//
// # When to Use This Validation
//
// This validation should be called:
//   - Before writing MCP gateway configuration to the gateway script
//   - In RenderJSONMCPConfig() after building the configuration
//   - To catch compiler bugs that produce invalid configurations
//
// For general validation, see validation.go.
// For schema validation architecture, see schema_validation.go.
package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var mcpGatewaySchemaValidationLog = logger.New("workflow:mcp_gateway_schema_validation")

//go:embed schemas/mcp-gateway-config.schema.json
var mcpGatewayConfigSchema string

// Cached compiled MCP gateway schema to avoid recompiling on every validation
var (
	compiledMCPGatewaySchemaOnce sync.Once
	compiledMCPGatewaySchema     *jsonschema.Schema
	mcpGatewaySchemaCompileError error
)

// getCompiledMCPGatewaySchema returns the compiled MCP Gateway schema, compiling it once and caching
func getCompiledMCPGatewaySchema() (*jsonschema.Schema, error) {
	compiledMCPGatewaySchemaOnce.Do(func() {
		mcpGatewaySchemaValidationLog.Print("Compiling MCP Gateway configuration schema (first time)")
		// Parse the embedded schema
		var schemaDoc any
		if err := json.Unmarshal([]byte(mcpGatewayConfigSchema), &schemaDoc); err != nil {
			mcpGatewaySchemaCompileError = fmt.Errorf("failed to parse embedded MCP Gateway configuration schema: %w", err)
			return
		}

		// Create compiler and add the schema as a resource
		loader := jsonschema.NewCompiler()
		schemaURL := "https://docs.github.com/gh-aw/schemas/mcp-gateway-config.schema.json"
		if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
			mcpGatewaySchemaCompileError = fmt.Errorf("failed to add MCP Gateway schema resource: %w", err)
			return
		}

		// Compile the schema once
		schema, err := loader.Compile(schemaURL)
		if err != nil {
			mcpGatewaySchemaCompileError = fmt.Errorf("failed to compile MCP Gateway configuration schema: %w", err)
			return
		}

		compiledMCPGatewaySchema = schema
		mcpGatewaySchemaValidationLog.Print("MCP Gateway configuration schema compiled successfully")
	})

	return compiledMCPGatewaySchema, mcpGatewaySchemaCompileError
}

// ValidateMCPGatewayConfig validates the MCP gateway configuration JSON against the schema
// This should be called before the configuration is sent to the gateway script
// Returns an error if the configuration is invalid (indicates a compiler bug)
func ValidateMCPGatewayConfig(configJSON string) error {
	mcpGatewaySchemaValidationLog.Print("Validating MCP gateway configuration against schema")

	// Parse JSON configuration
	var configData any
	if err := json.Unmarshal([]byte(configJSON), &configData); err != nil {
		return fmt.Errorf("internal compiler error: generated MCP gateway configuration is not valid JSON: %w", err)
	}

	// Get compiled schema (cached after first call)
	schema, err := getCompiledMCPGatewaySchema()
	if err != nil {
		return fmt.Errorf("internal compiler error: failed to load MCP Gateway configuration schema: %w", err)
	}

	// Validate the configuration against the schema
	if err := schema.Validate(configData); err != nil {
		// Format validation error with details
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			var errMsg strings.Builder
			errMsg.WriteString("internal compiler error: generated MCP gateway configuration does not conform to schema:\n")
			errMsg.WriteString(formatMCPGatewayValidationError(ve))
			return fmt.Errorf("%s", errMsg.String())
		}
		return fmt.Errorf("internal compiler error: MCP gateway configuration validation failed: %w", err)
	}

	mcpGatewaySchemaValidationLog.Print("MCP gateway configuration is valid")
	return nil
}

// formatMCPGatewayValidationError formats a jsonschema validation error into a readable message
func formatMCPGatewayValidationError(ve *jsonschema.ValidationError) string {
	var result strings.Builder

	// Main error - use Error() method to get formatted message
	result.WriteString("  - ")
	result.WriteString(ve.Error())
	result.WriteString("\n")

	// Add causes (nested validation errors) recursively
	for _, cause := range ve.Causes {
		result.WriteString("    - ")
		result.WriteString(cause.Error())
		result.WriteString("\n")
	}

	return result.String()
}
