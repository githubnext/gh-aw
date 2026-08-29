//go:build !integration

package workflow

import (
	"encoding/json"
	"maps"
	"os"
	"reflect"
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
			"port":    8080,
			"domain":  "localhost",
			"agentId": "test-agent",
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

func TestMCPGatewayConfigSchemaAgentIdentifiers(t *testing.T) {
	schemaPaths := []string{
		"schemas/mcp-gateway-config.schema.json",
		"../../docs/public/schemas/mcp-gateway-config.schema.json",
	}
	tests := []struct {
		name       string
		identity   map[string]any
		shouldPass bool
	}{
		{name: "single agent ID", identity: map[string]any{"agentId": "agent-1"}, shouldPass: true},
		{name: "multiple agent IDs", identity: map[string]any{"agentIds": []any{"agent-1", "agent-2"}}, shouldPass: true},
		{name: "missing agent ID", identity: map[string]any{}, shouldPass: false},
		{name: "both identity forms", identity: map[string]any{"agentId": "agent-1", "agentIds": []any{"agent-2"}}, shouldPass: false},
		{name: "empty single agent ID", identity: map[string]any{"agentId": ""}, shouldPass: false},
		{name: "empty agent IDs", identity: map[string]any{"agentIds": []any{}}, shouldPass: false},
		{name: "empty member", identity: map[string]any{"agentIds": []any{"agent-1", ""}}, shouldPass: false},
		{name: "removed API key", identity: map[string]any{"apiKey": "legacy-key"}, shouldPass: false},
	}

	for _, schemaPath := range schemaPaths {
		t.Run(schemaPath, func(t *testing.T) {
			schemaJSON, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("failed to read schema: %v", err)
			}
			var schemaDocument map[string]any
			if err := json.Unmarshal(schemaJSON, &schemaDocument); err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			definitions := requireObject(t, schemaDocument, "definitions")
			gatewaySchemaJSON, err := json.Marshal(map[string]any{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"$ref":    "#/definitions/gatewayConfig",
				"definitions": map[string]any{
					"gatewayConfig":       requireObject(t, definitions, "gatewayConfig"),
					"opentelemetryConfig": requireObject(t, definitions, "opentelemetryConfig"),
				},
			})
			if err != nil {
				t.Fatalf("failed to marshal gateway schema: %v", err)
			}
			schema, err := compileSchema(string(gatewaySchemaJSON), "https://docs.github.com/gh-aw/schemas/mcp-gateway-config-test.schema.json")
			if err != nil {
				t.Fatalf("failed to compile schema: %v", err)
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					gateway := map[string]any{
						"port":   8080,
						"domain": "localhost",
					}
					maps.Copy(gateway, tt.identity)
					err := schema.Validate(gateway)
					if tt.shouldPass && err != nil {
						t.Fatalf("expected config to validate: %v", err)
					}
					if !tt.shouldPass && err == nil {
						t.Fatal("expected config validation to fail")
					}
				})
			}
		})
	}
}

func TestMCPGatewayConfigSchemasHaveMatchingGatewayConfig(t *testing.T) {
	packageSchema := readMCPGatewayConfigDefinition(t, "schemas/mcp-gateway-config.schema.json")
	publishedSchema := readMCPGatewayConfigDefinition(t, "../../docs/public/schemas/mcp-gateway-config.schema.json")
	if !reflect.DeepEqual(packageSchema, publishedSchema) {
		t.Fatal("package and published gateway configuration schemas differ")
	}
}

func readMCPGatewayConfigDefinition(t *testing.T, schemaPath string) map[string]any {
	t.Helper()

	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	definitions := requireObject(t, schema, "definitions")
	return requireObject(t, definitions, "gatewayConfig")
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

	definitions := requireObject(t, schema, "definitions")
	otlpConfig := requireObject(t, definitions, "opentelemetryConfig")
	properties := requireObject(t, otlpConfig, "properties")
	endpoint := requireObject(t, properties, "endpoint")
	pattern, ok := endpoint["pattern"].(string)
	if !ok {
		t.Fatalf("expected endpoint pattern to be a string")
	}
	return pattern
}

func requireObject(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("missing %q in schema", key)
	}
	nested, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be an object", key)
	}
	return nested
}
