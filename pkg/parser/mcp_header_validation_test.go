package parser

import (
	"strings"
	"testing"
)

// TestHTTPHeaderLiteralValueValidation tests that literal values in HTTP headers are rejected
func TestHTTPHeaderLiteralValueValidation(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		mcpSection  map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name:     "Literal value in Authorization header should be rejected",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"Authorization": "Bearer hardcoded-token-123",
				},
			},
			wantErr:     true,
			errContains: "must use a GitHub Actions expression",
		},
		{
			name:     "Multiple literal values should be rejected",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"Authorization": "literal-value",
					"X-API-Key":     "another-literal",
				},
			},
			wantErr:     true,
			errContains: "must use a GitHub Actions expression",
		},
		{
			name:     "Valid GitHub Actions expression should be accepted",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"Authorization": "${{ secrets.API_KEY }}",
				},
			},
			wantErr: false,
		},
		{
			name:     "Valid expression with default value should be accepted",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"X-Site": "${{ secrets.DD_SITE || 'datadoghq.com' }}",
				},
			},
			wantErr: false,
		},
		{
			name:     "Valid vars expression should be accepted",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"X-Custom": "${{ vars.CUSTOM_HEADER }}",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMCPConfig(tt.toolName, tt.mcpSection, map[string]any{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
