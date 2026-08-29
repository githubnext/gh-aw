//go:build !integration

package workflow

import (
	"encoding/json"
	"maps"
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
	// The docs/public copy is intentionally excluded here: it additionally defines
	// customServerConfig with a negative-lookahead regex pattern that the schema
	// compiler's regex engine does not support (pre-existing, unrelated to
	// agentId/agentIds). Both copies still enforce the same agentId/agentIds
	// constraints — see the package schema below.
	schemaPaths := []string{
		"schemas/mcp-gateway-config.schema.json",
	}

	baseConfig := func() map[string]any {
		return map[string]any{
			"mcpServers": map[string]any{},
			"gateway": map[string]any{
				"port":   8080,
				"domain": "localhost",
			},
		}
	}

	tests := []struct {
		name    string
		gateway map[string]any
		wantErr bool
	}{
		{
			name:    "valid single agentId",
			gateway: map[string]any{"agentId": "primary-agent"},
			wantErr: false,
		},
		{
			name:    "valid single-entry agentIds",
			gateway: map[string]any{"agentIds": []any{"primary-agent"}},
			wantErr: false,
		},
		{
			name:    "valid multi-entry agentIds",
			gateway: map[string]any{"agentIds": []any{"primary-agent", "enclave-agent"}},
			wantErr: false,
		},
		{
			name:    "empty agentIds array",
			gateway: map[string]any{"agentIds": []any{}},
			wantErr: true,
		},
		{
			name:    "empty string entry in agentIds",
			gateway: map[string]any{"agentIds": []any{""}},
			wantErr: true,
		},
		{
			name:    "empty agentId string",
			gateway: map[string]any{"agentId": ""},
			wantErr: true,
		},
		{
			name:    "both agentId and agentIds",
			gateway: map[string]any{"agentId": "primary-agent", "agentIds": []any{"enclave-agent"}},
			wantErr: true,
		},
		{
			name:    "neither agentId nor agentIds",
			gateway: map[string]any{},
			wantErr: true,
		},
		{
			name:    "removed apiKey field",
			gateway: map[string]any{"apiKey": "gateway-secret-token"},
			wantErr: true,
		},
	}

	for _, schemaPath := range schemaPaths {
		t.Run(schemaPath, func(t *testing.T) {
			schemaJSON, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("failed to read schema: %v", err)
			}

			schema, err := compileSchema(string(schemaJSON), "https://docs.github.com/gh-aw/schemas/mcp-gateway-config.schema.json")
			if err != nil {
				t.Fatalf("failed to compile schema: %v", err)
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					config := baseConfig()
					gateway := config["gateway"].(map[string]any)
					maps.Copy(gateway, tc.gateway)

					err := schema.Validate(config)
					if tc.wantErr && err == nil {
						t.Fatalf("expected validation error for gateway config %+v, got none", tc.gateway)
					}
					if !tc.wantErr && err != nil {
						t.Fatalf("expected gateway config %+v to validate, got error: %v", tc.gateway, err)
					}
				})
			}
		})
	}
}

// TestMCPGatewayConfigSchemaCopiesStayInSync verifies that every published
// mcp-gateway-config.schema.json copy defines agentId/agentIds consistently
// and no longer defines apiKey, even for copies (like docs/public) that embed
// additional definitions the schema compiler's regex engine cannot compile.
func TestMCPGatewayConfigSchemaCopiesStayInSync(t *testing.T) {
	schemaPaths := []string{
		"schemas/mcp-gateway-config.schema.json",
		"../../docs/public/schemas/mcp-gateway-config.schema.json",
	}

	for _, schemaPath := range schemaPaths {
		t.Run(schemaPath, func(t *testing.T) {
			schemaJSON, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("failed to read schema: %v", err)
			}

			var schema map[string]any
			if err := json.Unmarshal(schemaJSON, &schema); err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}

			definitions := requireObject(t, schema, "definitions")
			gatewayConfig := requireObject(t, definitions, "gatewayConfig")
			properties := requireObject(t, gatewayConfig, "properties")

			if _, hasAPIKey := properties["apiKey"]; hasAPIKey {
				t.Fatal("expected gatewayConfig.properties to no longer define apiKey")
			}
			if _, hasAgentID := properties["agentId"]; !hasAgentID {
				t.Fatal("expected gatewayConfig.properties to define agentId")
			}
			if _, hasAgentIDs := properties["agentIds"]; !hasAgentIDs {
				t.Fatal("expected gatewayConfig.properties to define agentIds")
			}

			required, ok := gatewayConfig["required"].([]any)
			if !ok {
				t.Fatal("expected gatewayConfig.required to be an array")
			}
			for _, r := range required {
				if r == "apiKey" {
					t.Fatal("expected gatewayConfig.required to no longer require apiKey")
				}
				if r == "agentId" || r == "agentIds" {
					t.Fatalf("expected agentId/agentIds to be enforced via oneOf, not required directly, got required entry %q", r)
				}
			}

			oneOf, ok := gatewayConfig["oneOf"].([]any)
			if !ok || len(oneOf) != 2 {
				t.Fatalf("expected gatewayConfig.oneOf to encode agentId/agentIds mutual exclusion, got %v", gatewayConfig["oneOf"])
			}
		})
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
