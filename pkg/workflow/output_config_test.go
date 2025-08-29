package workflow

import (
	"testing"
)

func TestAllowedDomainsParsing(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedDomains []string
	}{
		{
			name: "no output config",
			frontmatter: map[string]any{
				"engine": "claude",
			},
			expectedDomains: nil,
		},
		{
			name: "output config with allowed-domains",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"allowed-domains": []any{"example.com", "trusted.org"},
				},
			},
			expectedDomains: []string{"example.com", "trusted.org"},
		},
		{
			name: "output config with create-issue and allowed-domains",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"title-prefix": "[auto] ",
					},
					"allowed-domains": []any{"github.com", "api.github.com"},
				},
			},
			expectedDomains: []string{"github.com", "api.github.com"},
		},
		{
			name: "output config without allowed-domains",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"title-prefix": "[auto] ",
					},
				},
			},
			expectedDomains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(false, "", "test")
			config := c.extractSafeOutputsConfig(tt.frontmatter)

			if tt.expectedDomains == nil {
				if config == nil {
					return // expected case
				}
				if len(config.AllowedDomains) == 0 {
					return // expected case
				}
				t.Errorf("Expected no allowed domains, but got %v", config.AllowedDomains)
				return
			}

			if config == nil {
				t.Errorf("Expected output config, but got nil")
				return
			}

			if len(config.AllowedDomains) != len(tt.expectedDomains) {
				t.Errorf("Expected %d allowed domains, but got %d", len(tt.expectedDomains), len(config.AllowedDomains))
				return
			}

			for i, expected := range tt.expectedDomains {
				if config.AllowedDomains[i] != expected {
					t.Errorf("Expected domain %s at index %d, but got %s", expected, i, config.AllowedDomains[i])
				}
			}
		})
	}
}

func TestAllowedDomainsInWorkflow(t *testing.T) {
	// Create a test compiler with verbose output to check generated workflow
	c := NewCompiler(true, "", "test")

	// Test workflow with allowed domains
	frontmatter := map[string]any{
		"engine": "claude",
		"safe-outputs": map[string]any{
			"allowed-domains": []any{"example.com", "trusted.org"},
		},
	}

	config := c.extractSafeOutputsConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected output config, but got nil")
	}

	if len(config.AllowedDomains) != 2 {
		t.Errorf("Expected 2 allowed domains, but got %d", len(config.AllowedDomains))
	}

	expectedDomains := []string{"example.com", "trusted.org"}
	for i, expected := range expectedDomains {
		if config.AllowedDomains[i] != expected {
			t.Errorf("Expected domain %s at index %d, but got %s", expected, i, config.AllowedDomains[i])
		}
	}
}
