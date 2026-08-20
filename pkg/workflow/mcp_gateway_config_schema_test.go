//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

func TestMCPGatewayConfigSchemaAcceptsHTTPOTLPEndpoint(t *testing.T) {
	schemaPaths := []string{
		"schemas/mcp-gateway-config.schema.json",
		"../../docs/public/schemas/mcp-gateway-config.schema.json",
	}

	for _, schemaPath := range schemaPaths {
		t.Run(schemaPath, func(t *testing.T) {
			pattern := mcpGatewayOTLPEndpointPattern(t, schemaPath)
			matched, err := regexp.MatchString(pattern, "http://127.0.0.1:4318/v1/traces")
			if err != nil {
				t.Fatalf("invalid endpoint pattern: %v", err)
			}
			if !matched {
				t.Fatal("expected endpoint pattern to accept HTTP OTLP endpoint")
			}
			matched, err = regexp.MatchString(pattern, "ftp://127.0.0.1:4318/v1/traces")
			if err != nil {
				t.Fatalf("invalid endpoint pattern: %v", err)
			}
			if matched {
				t.Fatal("expected endpoint pattern to reject non-HTTP(S) OTLP endpoint")
			}
		})
	}

	schemaJSON, err := os.ReadFile("schemas/mcp-gateway-config.schema.json")
	if err != nil {
		t.Fatalf("failed to read package schema: %v", err)
	}

	schema, err := compileSchema(string(schemaJSON), "https://docs.github.com/gh-aw/schemas/mcp-gateway-config.schema.json")
	if err != nil {
		t.Fatalf("failed to compile package schema: %v", err)
	}

	config := map[string]any{
		"mcpServers": map[string]any{},
		"gateway": map[string]any{
			"port":   8080,
			"domain": "localhost",
			"apiKey": "test-key",
			"opentelemetry": map[string]any{
				"endpoint": "http://127.0.0.1:4318/v1/traces",
			},
		},
	}
	if err := schema.Validate(config); err != nil {
		t.Fatalf("expected HTTP OTLP endpoint to validate: %v", err)
	}

	config["gateway"].(map[string]any)["opentelemetry"].(map[string]any)["endpoint"] = "ftp://127.0.0.1:4318/v1/traces"
	if err := schema.Validate(config); err == nil {
		t.Fatal("expected non-HTTP(S) OTLP endpoint to fail validation")
	}
}

func mcpGatewayOTLPEndpointPattern(t *testing.T, schemaPath string) string {
	t.Helper()

	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	definitions := schema["definitions"].(map[string]any)
	otlpConfig := definitions["opentelemetryConfig"].(map[string]any)
	properties := otlpConfig["properties"].(map[string]any)
	endpoint := properties["endpoint"].(map[string]any)
	return endpoint["pattern"].(string)
}
