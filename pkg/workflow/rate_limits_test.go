package workflow

import (
	"testing"
)

func TestExtractRateLimitsConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name              string
		frontmatter       map[string]any
		expectNil         bool
		expectedGitHubAPI string
		expectedMCP       string
		expectedNetwork   string
		expectedFileRead  string
	}{
		{
			name:        "no rate-limits",
			frontmatter: map[string]any{},
			expectNil:   true,
		},
		{
			name: "all rate-limits",
			frontmatter: map[string]any{
				"rate-limits": map[string]any{
					"github-api":       "100/hour",
					"mcp-requests":     "50/minute",
					"network-requests": "60/minute",
					"file-read":        "1000/minute",
				},
			},
			expectNil:         false,
			expectedGitHubAPI: "100/hour",
			expectedMCP:       "50/minute",
			expectedNetwork:   "60/minute",
			expectedFileRead:  "1000/minute",
		},
		{
			name: "partial rate-limits",
			frontmatter: map[string]any{
				"rate-limits": map[string]any{
					"github-api":   "500/day",
					"mcp-requests": "100/minute",
				},
			},
			expectNil:         false,
			expectedGitHubAPI: "500/day",
			expectedMCP:       "100/minute",
			expectedNetwork:   "",
			expectedFileRead:  "",
		},
		{
			name: "empty rate-limits object",
			frontmatter: map[string]any{
				"rate-limits": map[string]any{},
			},
			expectNil: true,
		},
		{
			name: "non-map rate-limits",
			frontmatter: map[string]any{
				"rate-limits": "invalid",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractRateLimitsConfig(tt.frontmatter)

			if tt.expectNil {
				if config != nil {
					t.Errorf("Expected nil config, got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("Expected non-nil config")
			}

			if config.GitHubAPI != tt.expectedGitHubAPI {
				t.Errorf("GitHubAPI = %q, want %q", config.GitHubAPI, tt.expectedGitHubAPI)
			}
			if config.MCPRequests != tt.expectedMCP {
				t.Errorf("MCPRequests = %q, want %q", config.MCPRequests, tt.expectedMCP)
			}
			if config.NetworkRequests != tt.expectedNetwork {
				t.Errorf("NetworkRequests = %q, want %q", config.NetworkRequests, tt.expectedNetwork)
			}
			if config.FileRead != tt.expectedFileRead {
				t.Errorf("FileRead = %q, want %q", config.FileRead, tt.expectedFileRead)
			}
		})
	}
}

func TestMergeRateLimits(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name              string
		topConfig         *RateLimitsConfig
		importedJSON      string
		expectNil         bool
		expectedGitHubAPI string
		expectedMCP       string
		expectedNetwork   string
		expectedFileRead  string
	}{
		{
			name:         "no imports",
			topConfig:    nil,
			importedJSON: "",
			expectNil:    true,
		},
		{
			name: "top config only",
			topConfig: &RateLimitsConfig{
				GitHubAPI:   "100/hour",
				MCPRequests: "50/minute",
			},
			importedJSON:      "",
			expectNil:         false,
			expectedGitHubAPI: "100/hour",
			expectedMCP:       "50/minute",
		},
		{
			name:              "imports only",
			topConfig:         nil,
			importedJSON:      `{"github-api":"200/hour","mcp-requests":"100/minute"}`,
			expectNil:         false,
			expectedGitHubAPI: "200/hour",
			expectedMCP:       "100/minute",
		},
		{
			name: "top config takes precedence",
			topConfig: &RateLimitsConfig{
				GitHubAPI: "100/hour",
			},
			importedJSON:      `{"github-api":"200/hour","mcp-requests":"100/minute"}`,
			expectNil:         false,
			expectedGitHubAPI: "100/hour",
			expectedMCP:       "100/minute",
		},
		{
			name: "multiple import lines",
			topConfig: &RateLimitsConfig{
				GitHubAPI: "100/hour",
			},
			importedJSON:      "{\"mcp-requests\":\"50/minute\"}\n{\"network-requests\":\"60/minute\"}",
			expectNil:         false,
			expectedGitHubAPI: "100/hour",
			expectedMCP:       "50/minute",
			expectedNetwork:   "60/minute",
		},
		{
			name:         "empty JSON object",
			topConfig:    nil,
			importedJSON: "{}",
			expectNil:    true,
		},
		{
			name: "invalid JSON skipped",
			topConfig: &RateLimitsConfig{
				GitHubAPI: "100/hour",
			},
			importedJSON:      "invalid\n{\"mcp-requests\":\"50/minute\"}",
			expectNil:         false,
			expectedGitHubAPI: "100/hour",
			expectedMCP:       "50/minute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := compiler.MergeRateLimits(tt.topConfig, tt.importedJSON)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectNil {
				if config != nil {
					t.Errorf("Expected nil config, got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("Expected non-nil config")
			}

			if config.GitHubAPI != tt.expectedGitHubAPI {
				t.Errorf("GitHubAPI = %q, want %q", config.GitHubAPI, tt.expectedGitHubAPI)
			}
			if config.MCPRequests != tt.expectedMCP {
				t.Errorf("MCPRequests = %q, want %q", config.MCPRequests, tt.expectedMCP)
			}
			if config.NetworkRequests != tt.expectedNetwork {
				t.Errorf("NetworkRequests = %q, want %q", config.NetworkRequests, tt.expectedNetwork)
			}
			if config.FileRead != tt.expectedFileRead {
				t.Errorf("FileRead = %q, want %q", config.FileRead, tt.expectedFileRead)
			}
		})
	}
}
