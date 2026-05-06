//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractOTLPEndpointDomain verifies hostname extraction from OTLP endpoint URLs.
func TestExtractOTLPEndpointDomain(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "empty endpoint returns empty string",
			endpoint: "",
			expected: "",
		},
		{
			name:     "GitHub Actions expression returns empty string",
			endpoint: "${{ secrets.OTLP_ENDPOINT }}",
			expected: "",
		},
		{
			name:     "inline expression returns empty string",
			endpoint: "https://${{ secrets.HOST }}:4317",
			expected: "",
		},
		{
			name:     "HTTPS URL without port",
			endpoint: "https://traces.example.com",
			expected: "traces.example.com",
		},
		{
			name:     "HTTPS URL with port",
			endpoint: "https://traces.example.com:4317",
			expected: "traces.example.com",
		},
		{
			name:     "HTTP URL with path",
			endpoint: "http://otel-collector.internal:4318/v1/traces",
			expected: "otel-collector.internal",
		},
		{
			name:     "gRPC URL",
			endpoint: "grpc://traces.example.com:4317",
			expected: "traces.example.com",
		},
		{
			name:     "subdomain",
			endpoint: "https://otel.collector.corp.example.com:4317",
			expected: "otel.collector.corp.example.com",
		},
		{
			name:     "invalid URL (no scheme) returns empty string",
			endpoint: "traces.example.com:4317",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOTLPEndpointDomain(tt.endpoint)
			assert.Equal(t, tt.expected, got, "extractOTLPEndpointDomain(%q)", tt.endpoint)
		})
	}
}

// TestGetOTLPEndpointEnvValue verifies endpoint value extraction from FrontmatterConfig.
func TestGetOTLPEndpointEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		config   *FrontmatterConfig
		expected string
	}{
		{
			name:     "nil config returns empty string",
			config:   nil,
			expected: "",
		},
		{
			name:     "nil observability returns empty string",
			config:   &FrontmatterConfig{},
			expected: "",
		},
		{
			name: "nil OTLP returns empty string",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{},
			},
			expected: "",
		},
		{
			name: "empty string endpoint returns empty string",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: ""},
				},
			},
			expected: "",
		},
		{
			name: "static URL endpoint (string form)",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://traces.example.com:4317"},
				},
			},
			expected: "https://traces.example.com:4317",
		},
		{
			name: "secret expression endpoint (string form)",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "${{ secrets.OTLP_ENDPOINT }}"},
				},
			},
			expected: "${{ secrets.OTLP_ENDPOINT }}",
		},
		{
			name: "object form returns empty string (only string form handled by this function)",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: map[string]any{"url": "https://traces.example.com:4317"}},
				},
			},
			expected: "",
		},
		{
			name: "nil endpoint returns empty string",
			config: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: nil},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOTLPEndpointEnvValue(tt.config)
			assert.Equal(t, tt.expected, got, "getOTLPEndpointEnvValue")
		})
	}
}

// TestInjectOTLPConfig verifies that injectOTLPConfig correctly modifies WorkflowData.
func TestInjectOTLPConfig(t *testing.T) {
	newCompiler := func() *Compiler { return &Compiler{} }

	t.Run("no-op when OTLP is not configured", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{},
		}
		c.injectOTLPConfig(wd)
		assert.Nil(t, wd.NetworkPermissions, "NetworkPermissions should remain nil")
		assert.Empty(t, wd.Env, "Env should remain empty")
	})

	t.Run("no-op when ParsedFrontmatter is nil", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{}
		c.injectOTLPConfig(wd)
		assert.Nil(t, wd.NetworkPermissions, "NetworkPermissions should remain nil")
		assert.Empty(t, wd.Env, "Env should remain empty")
	})

	t.Run("injects env vars when endpoint is a secret expression", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "${{ secrets.OTLP_ENDPOINT }}"},
				},
			},
		}
		c.injectOTLPConfig(wd)

		// NetworkPermissions.Allowed should NOT be populated (can't resolve expression)
		if wd.NetworkPermissions != nil {
			assert.Empty(t, wd.NetworkPermissions.Allowed, "Allowed should be empty for expression endpoints")
		}

		// Env should contain the OTEL vars
		require.NotEmpty(t, wd.Env, "Env should be set")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: ${{ secrets.OTLP_ENDPOINT }}", "should contain endpoint var")
		assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME: gh-aw", "should contain service name")
	})

	t.Run("adds domain to new NetworkPermissions and injects env vars for static URL", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://traces.example.com:4317"},
				},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotNil(t, wd.NetworkPermissions, "NetworkPermissions should be created")
		assert.Contains(t, wd.NetworkPermissions.Allowed, "traces.example.com", "should contain OTLP domain")

		require.NotEmpty(t, wd.Env, "Env should be set")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com:4317")
		assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME: gh-aw")
		assert.True(t, strings.HasPrefix(wd.Env, "env:"), "Env should start with 'env:'")
	})

	t.Run("appends domain to existing NetworkPermissions.Allowed", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://traces.example.com:4317"},
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.github.com", "pypi.org"},
			},
		}
		c.injectOTLPConfig(wd)

		assert.Contains(t, wd.NetworkPermissions.Allowed, "api.github.com", "existing domains should remain")
		assert.Contains(t, wd.NetworkPermissions.Allowed, "pypi.org", "existing domains should remain")
		assert.Contains(t, wd.NetworkPermissions.Allowed, "traces.example.com", "OTLP domain should be appended")
	})

	t.Run("appends OTEL vars to existing Env block", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://traces.example.com"},
				},
			},
			Env: "env:\n  MY_VAR: hello",
		}
		c.injectOTLPConfig(wd)

		assert.Contains(t, wd.Env, "MY_VAR: hello", "existing env var should remain")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com")
		assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME: gh-aw")
		// Should still be a single env: block
		assert.Equal(t, 1, strings.Count(wd.Env, "env:"), "should have exactly one env: key")
	})

	t.Run("OTEL_SERVICE_NAME is always gh-aw", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://otel.corp.com"},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME: gh-aw", "service name should always be gh-aw")
	})

	t.Run("injects OTEL_EXPORTER_OTLP_HEADERS when headers are configured", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com",
						Headers:  "Authorization=Bearer tok,X-Tenant=acme",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: Authorization=Bearer tok,X-Tenant=acme", "headers var should be injected")
	})

	t.Run("injects OTEL_EXPORTER_OTLP_HEADERS for secret expression", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com",
						Headers:  "${{ secrets.OTLP_HEADERS }}",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: ${{ secrets.OTLP_HEADERS }}", "headers var should support secret expressions")
	})

	t.Run("does not inject OTEL_EXPORTER_OTLP_HEADERS when headers not configured", func(t *testing.T) {
		c := newCompiler()
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{Endpoint: "https://traces.example.com"},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.NotContains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS", "headers var should not appear when unconfigured")
	})
}

// TestObservabilityConfigParsing verifies that the OTLPConfig is correctly parsed
// from raw frontmatter via ParseFrontmatterConfig.
func TestObservabilityConfigParsing(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      map[string]any
		wantOTLPConfig   bool
		expectedEndpoint string
		expectedHeaders  string
	}{
		{
			name:           "no observability section",
			frontmatter:    map[string]any{},
			wantOTLPConfig: false,
		},
		{
			name:           "observability without otlp",
			frontmatter:    map[string]any{"observability": map[string]any{}},
			wantOTLPConfig: false,
		},
		{
			name: "observability with otlp endpoint",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4317",
					},
				},
			},
			wantOTLPConfig:   true,
			expectedEndpoint: "https://traces.example.com:4317",
		},
		{
			name: "observability with otlp secret expression",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "${{ secrets.OTLP_ENDPOINT }}",
					},
				},
			},
			wantOTLPConfig:   true,
			expectedEndpoint: "${{ secrets.OTLP_ENDPOINT }}",
		},
		{
			name: "observability with both otlp endpoint and config",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
					},
				},
			},
			wantOTLPConfig:   true,
			expectedEndpoint: "https://traces.example.com",
		},
		{
			name: "observability with otlp endpoint and headers",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers":  "Authorization=Bearer tok,X-Tenant=acme",
					},
				},
			},
			wantOTLPConfig:   true,
			expectedEndpoint: "https://traces.example.com",
			expectedHeaders:  "Authorization=Bearer tok,X-Tenant=acme",
		},
		{
			name: "observability with otlp headers as secret expression",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers":  "${{ secrets.OTLP_HEADERS }}",
					},
				},
			},
			wantOTLPConfig:   true,
			expectedEndpoint: "https://traces.example.com",
			expectedHeaders:  "${{ secrets.OTLP_HEADERS }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseFrontmatterConfig(tt.frontmatter)
			require.NoError(t, err, "ParseFrontmatterConfig should not fail")
			require.NotNil(t, config, "Config should not be nil")

			if !tt.wantOTLPConfig {
				if config.Observability != nil {
					assert.Nil(t, config.Observability.OTLP, "OTLP should be nil")
				}
				return
			}

			require.NotNil(t, config.Observability, "Observability should not be nil")
			require.NotNil(t, config.Observability.OTLP, "OTLP should not be nil")
			assert.Equal(t, tt.expectedEndpoint, config.Observability.OTLP.Endpoint, "Endpoint should match")
			// Normalize Headers (any) to string for comparison
			normalizedHeaders, _ := normalizeOTLPHeaders(config.Observability.OTLP.Headers)
			assert.Equal(t, tt.expectedHeaders, normalizedHeaders, "Headers should match")
		})
	}
}

// TestInjectOTLPConfig_RawFrontmatterFallback verifies that injectOTLPConfig works
// when ParsedFrontmatter is nil (e.g. complex engine objects cause ParseFrontmatterConfig
// to fail) but the raw frontmatter contains valid OTLP configuration.
func TestInjectOTLPConfig_RawFrontmatterFallback(t *testing.T) {
	c := &Compiler{}

	t.Run("injects OTLP from raw frontmatter when ParsedFrontmatter is nil", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: nil, // simulates ParseFrontmatterConfig failure
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "${{ secrets.GH_AW_OTEL_ENDPOINT }}",
						"headers":  "${{ secrets.GH_AW_OTEL_HEADERS }}",
					},
				},
				// Simulate complex engine object that would cause ParseFrontmatterConfig to fail.
				"engine": map[string]any{"id": "copilot", "max-continuations": 2},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotEmpty(t, wd.Env, "Env should be set even without ParsedFrontmatter")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: ${{ secrets.GH_AW_OTEL_ENDPOINT }}", "endpoint should be injected from raw")
		assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME: gh-aw", "service name should be set")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: ${{ secrets.GH_AW_OTEL_HEADERS }}", "headers should be injected from raw")
	})

	t.Run("no-op when neither raw nor parsed frontmatter has OTLP", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: nil,
			RawFrontmatter:    map[string]any{"name": "my-workflow"},
		}
		c.injectOTLPConfig(wd)
		assert.Empty(t, wd.Env, "Env should remain empty")
		assert.Nil(t, wd.NetworkPermissions, "NetworkPermissions should remain nil")
	})
}

// TestIsOTLPHeadersPresent verifies that isOTLPHeadersPresent correctly detects
// whether OTEL_EXPORTER_OTLP_HEADERS is present in the workflow env block.
func TestIsOTLPHeadersPresent(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
	}{
		{
			name:     "nil WorkflowData returns false",
			data:     nil,
			expected: false,
		},
		{
			name:     "empty Env returns false",
			data:     &WorkflowData{},
			expected: false,
		},
		{
			name: "Env without OTEL_EXPORTER_OTLP_HEADERS returns false",
			data: &WorkflowData{
				Env: "env:\n  OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com\n  OTEL_SERVICE_NAME: gh-aw",
			},
			expected: false,
		},
		{
			name: "Env with OTEL_EXPORTER_OTLP_HEADERS returns true",
			data: &WorkflowData{
				Env: "env:\n  OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com\n  OTEL_SERVICE_NAME: gh-aw\n  OTEL_EXPORTER_OTLP_HEADERS: Authorization=Bearer tok",
			},
			expected: true,
		},
		{
			name: "Env with secret expression headers returns true",
			data: &WorkflowData{
				Env: "env:\n  OTEL_EXPORTER_OTLP_ENDPOINT: ${{ secrets.OTLP_ENDPOINT }}\n  OTEL_SERVICE_NAME: gh-aw\n  OTEL_EXPORTER_OTLP_HEADERS: ${{ secrets.OTLP_HEADERS }}",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOTLPHeadersPresent(tt.data)
			assert.Equal(t, tt.expected, got, "isOTLPHeadersPresent")
		})
	}
}

// TestGenerateOTLPHeadersMaskStep verifies that generateOTLPHeadersMaskStep
// emits a step that delegates to mask_otlp_headers.sh.
func TestGenerateOTLPHeadersMaskStep(t *testing.T) {
	step := generateOTLPHeadersMaskStep()

	assert.Contains(t, step, "- name: Mask OTLP telemetry headers", "should have the masking step name")
	assert.Contains(t, step, "mask_otlp_headers.sh", "should delegate to the mask_otlp_headers.sh script")
	assert.Contains(t, step, "${RUNNER_TEMP}/gh-aw/actions/", "should reference the runtime actions directory")
}

// TestInjectOTLPConfig_HeadersPresenceAfterInjection verifies that
// isOTLPHeadersPresent returns the expected value after injectOTLPConfig runs.
func TestInjectOTLPConfig_HeadersPresenceAfterInjection(t *testing.T) {
	t.Run("isOTLPHeadersPresent returns true after headers are injected", func(t *testing.T) {
		c := &Compiler{}
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com",
						Headers:  "Authorization=Bearer tok",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.True(t, isOTLPHeadersPresent(wd), "isOTLPHeadersPresent should return true after headers are injected")
	})

	t.Run("isOTLPHeadersPresent returns false when no headers are configured", func(t *testing.T) {
		c := &Compiler{}
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.False(t, isOTLPHeadersPresent(wd), "isOTLPHeadersPresent should return false when no headers are configured")
	})
}

// TestInjectOTLPConfig_OTLPEndpointField verifies that injectOTLPConfig sets workflowData.OTLPEndpoint
// so that downstream code (buildMCPGatewayConfig, mcp_setup_generator) can use it as the
// single source of truth for "is OTLP configured?" without re-reading raw frontmatter.
func TestInjectOTLPConfig_OTLPEndpointField(t *testing.T) {
	c := &Compiler{}

	t.Run("sets OTLPEndpoint when endpoint is configured", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4318",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Equal(t, "https://traces.example.com:4318", wd.OTLPEndpoint, "OTLPEndpoint should be set to the resolved endpoint")
	})

	t.Run("does not set OTLPEndpoint when OTLP is not configured", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{"name": "no-otlp"},
		}
		c.injectOTLPConfig(wd)
		assert.Empty(t, wd.OTLPEndpoint, "OTLPEndpoint should remain empty when OTLP is not configured")
	})

	t.Run("sets OTLPEndpoint from imported observability merged into RawFrontmatter", func(t *testing.T) {
		// Simulate what compiler_orchestrator_workflow.go does when importing shared/observability-otlp.md:
		// the imported observability JSON is decoded and injected into RawFrontmatter before injectOTLPConfig runs.
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				// Imported observability merged in (top-level has no observability key)
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "${{ secrets.GH_AW_OTEL_ENDPOINT }}",
						"headers":  "${{ secrets.GH_AW_OTEL_HEADERS }}",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Equal(t, "${{ secrets.GH_AW_OTEL_ENDPOINT }}", wd.OTLPEndpoint, "OTLPEndpoint should be set from imported observability")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT:", "env var should be injected")
	})
}

// TestInjectOTLPConfig_OTLPHeadersField verifies that injectOTLPConfig sets workflowData.OTLPHeaders
// so that buildMCPGatewayConfig can read it directly instead of re-reading raw frontmatter.
func TestInjectOTLPConfig_OTLPHeadersField(t *testing.T) {
	c := &Compiler{}

	t.Run("sets OTLPHeaders when headers are configured (map form)", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers":  map[string]any{"Authorization": "Bearer tok", "X-Tenant": "acme"},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Equal(t, "Authorization=Bearer tok,X-Tenant=acme", wd.OTLPHeaders, "OTLPHeaders should be set from map form")
	})

	t.Run("sets OTLPHeaders when headers are configured (string form)", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers":  "Authorization=Bearer tok",
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Equal(t, "Authorization=Bearer tok", wd.OTLPHeaders, "OTLPHeaders should be set from string form")
	})

	t.Run("OTLPHeaders is empty when no headers are configured", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{"endpoint": "https://traces.example.com"},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Empty(t, wd.OTLPHeaders, "OTLPHeaders should be empty when no headers are configured")
	})
}

// TestNormalizeOTLPHeaders verifies the normalizeOTLPHeaders helper function.
func TestNormalizeOTLPHeaders(t *testing.T) {
	tests := []struct {
		name               string
		input              any
		expectedHeaders    string
		expectedDeprecated bool
	}{
		{
			name:               "nil returns empty non-deprecated",
			input:              nil,
			expectedHeaders:    "",
			expectedDeprecated: false,
		},
		{
			name:               "empty string returns empty non-deprecated",
			input:              "",
			expectedHeaders:    "",
			expectedDeprecated: false,
		},
		{
			name:               "non-empty string returns string as deprecated",
			input:              "Authorization=Bearer tok",
			expectedHeaders:    "Authorization=Bearer tok",
			expectedDeprecated: true,
		},
		{
			name:               "secret expression string is deprecated",
			input:              "${{ secrets.OTLP_HEADERS }}",
			expectedHeaders:    "${{ secrets.OTLP_HEADERS }}",
			expectedDeprecated: true,
		},
		{
			name:               "empty map returns empty non-deprecated",
			input:              map[string]any{},
			expectedHeaders:    "",
			expectedDeprecated: false,
		},
		{
			name:            "single-entry map",
			input:           map[string]any{"Authorization": "Bearer tok"},
			expectedHeaders: "Authorization=Bearer tok",
		},
		{
			name: "multi-entry map sorts keys deterministically",
			input: map[string]any{
				"X-Tenant":      "acme",
				"Authorization": "Bearer tok",
			},
			expectedHeaders: "Authorization=Bearer tok,X-Tenant=acme",
		},
		{
			name: "map with secret expression value",
			input: map[string]any{
				"Authorization": "${{ secrets.TOKEN }}",
				"X-Tenant":      "acme",
			},
			expectedHeaders: "Authorization=${{ secrets.TOKEN }},X-Tenant=acme",
		},
		{
			name:               "unsupported type returns empty non-deprecated",
			input:              42,
			expectedHeaders:    "",
			expectedDeprecated: false,
		},
		{
			name: "non-string map values are skipped",
			input: map[string]any{
				"Authorization": "Bearer tok",
				"bad-value":     123, // non-string: skipped
			},
			expectedHeaders:    "Authorization=Bearer tok",
			expectedDeprecated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHeaders, gotDeprecated := normalizeOTLPHeaders(tt.input)
			assert.Equal(t, tt.expectedHeaders, gotHeaders, "headers should match")
			assert.Equal(t, tt.expectedDeprecated, gotDeprecated, "deprecated flag should match")
		})
	}
}

// TestInjectOTLPConfig_MapHeaders verifies that the map form for headers is supported.
func TestInjectOTLPConfig_MapHeaders(t *testing.T) {
	t.Run("injects OTEL_EXPORTER_OTLP_HEADERS from map form", func(t *testing.T) {
		c := &Compiler{}
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers": map[string]any{
							"Authorization": "Bearer ${{ secrets.TOKEN }}",
							"X-Tenant":      "acme",
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: Authorization=Bearer ${{ secrets.TOKEN }},X-Tenant=acme",
			"headers should be serialised as sorted key=value pairs")
	})

	t.Run("map form with single header", func(t *testing.T) {
		c := &Compiler{}
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com",
						"headers": map[string]any{
							"api-key": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: api-key=${{ secrets.API_KEY }}")
	})

	t.Run("map form via ParsedFrontmatter fallback", func(t *testing.T) {
		c := &Compiler{}
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com",
						Headers: map[string]any{
							"Authorization": "Bearer tok",
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: Authorization=Bearer tok",
			"map headers should work via ParsedFrontmatter fallback")
	})
}

// correctly parsed by ParseFrontmatterConfig.
func TestObservabilityConfigParsing_MapHeaders(t *testing.T) {
	t.Run("map headers parsed as any type", func(t *testing.T) {
		frontmatter := map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com",
					"headers": map[string]any{
						"Authorization": "Bearer tok",
						"X-Tenant":      "acme",
					},
				},
			},
		}
		config, err := ParseFrontmatterConfig(frontmatter)
		require.NoError(t, err, "ParseFrontmatterConfig should not fail")
		require.NotNil(t, config.Observability)
		require.NotNil(t, config.Observability.OTLP)
		assert.Equal(t, "https://traces.example.com", config.Observability.OTLP.Endpoint)

		// The Headers field should hold the map as-is
		headersMap, ok := config.Observability.OTLP.Headers.(map[string]any)
		require.True(t, ok, "Headers should be a map[string]any when map form is used")
		assert.Equal(t, "Bearer tok", headersMap["Authorization"])
		assert.Equal(t, "acme", headersMap["X-Tenant"])
	})

	t.Run("string headers parsed as any string", func(t *testing.T) {
		frontmatter := map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com",
					"headers":  "Authorization=Bearer tok",
				},
			},
		}
		config, err := ParseFrontmatterConfig(frontmatter)
		require.NoError(t, err, "ParseFrontmatterConfig should not fail")
		require.NotNil(t, config.Observability)
		require.NotNil(t, config.Observability.OTLP)
		headersStr, ok := config.Observability.OTLP.Headers.(string)
		require.True(t, ok, "Headers should be a string when string form is used")
		assert.Equal(t, "Authorization=Bearer tok", headersStr)
	})
}

// TestCollectAllOTLPEndpoints verifies that endpoint entries are correctly parsed from
// the polymorphic `endpoint` field (string, object, or array).
func TestCollectAllOTLPEndpoints(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantEntries []otlpEndpointEntry
		wantDep     bool
	}{
		{
			name:        "empty frontmatter returns empty slice",
			frontmatter: map[string]any{},
			wantEntries: nil,
		},
		{
			name: "string form: single URL",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4317",
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "string form: single URL with top-level headers (deprecated string form)",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4317",
						"headers":  "Authorization=Bearer tok",
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://traces.example.com:4317", Headers: "Authorization=Bearer tok"},
			},
			wantDep: true,
		},
		{
			name: "string form: single URL with top-level headers (map form)",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4317",
						"headers":  map[string]any{"Authorization": "Bearer tok"},
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://traces.example.com:4317", Headers: "Authorization=Bearer tok"},
			},
		},
		{
			name: "object form: single endpoint with per-endpoint headers",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": map[string]any{
							"url":     "https://traces.example.com:4317",
							"headers": map[string]any{"X-API-Key": "key1"},
						},
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://traces.example.com:4317", Headers: "X-API-Key=key1"},
			},
		},
		{
			name: "object form: single endpoint without headers",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": map[string]any{
							"url": "https://traces.example.com:4317",
						},
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "array form: multiple endpoints",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "https://primary.example.com:4317"},
							map[string]any{"url": "https://secondary.example.com:4317", "headers": map[string]any{"X-API-Key": "key2"}},
						},
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://primary.example.com:4317"},
				{URL: "https://secondary.example.com:4317", Headers: "X-API-Key=key2"},
			},
		},
		{
			name: "array form: entries with empty URL are skipped",
			frontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": ""},
							map[string]any{"url": "https://valid.example.com:4317"},
						},
					},
				},
			},
			wantEntries: []otlpEndpointEntry{
				{URL: "https://valid.example.com:4317"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotDep := collectAllOTLPEndpoints(tt.frontmatter)
			assert.Equal(t, tt.wantEntries, got, "endpoint entries")
			assert.Equal(t, tt.wantDep, gotDep, "deprecated flag")
		})
	}
}

// TestEncodeOTLPEndpoints verifies JSON serialisation of endpoint entries.
func TestEncodeOTLPEndpoints(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		assert.Empty(t, encodeOTLPEndpoints(nil))
		assert.Empty(t, encodeOTLPEndpoints([]otlpEndpointEntry{}))
	})

	t.Run("single entry without headers", func(t *testing.T) {
		encoded := encodeOTLPEndpoints([]otlpEndpointEntry{{URL: "https://traces.example.com:4317"}})
		assert.JSONEq(t, `[{"url":"https://traces.example.com:4317"}]`, encoded)
	})

	t.Run("single entry with headers", func(t *testing.T) {
		encoded := encodeOTLPEndpoints([]otlpEndpointEntry{{URL: "https://traces.example.com:4317", Headers: "Authorization=Bearer tok"}})
		assert.JSONEq(t, `[{"url":"https://traces.example.com:4317","headers":"Authorization=Bearer tok"}]`, encoded)
	})

	t.Run("multiple entries", func(t *testing.T) {
		encoded := encodeOTLPEndpoints([]otlpEndpointEntry{
			{URL: "https://primary.example.com:4317", Headers: "Authorization=Bearer tok1"},
			{URL: "https://secondary.example.com:4317", Headers: "Authorization=Bearer tok2"},
		})
		assert.JSONEq(t, `[{"url":"https://primary.example.com:4317","headers":"Authorization=Bearer tok1"},{"url":"https://secondary.example.com:4317","headers":"Authorization=Bearer tok2"}]`, encoded)
	})
}

// TestAllOTLPHeaders verifies that allOTLPHeaders concatenates headers from all entries.
func TestAllOTLPHeaders(t *testing.T) {
	t.Run("empty entries returns empty string", func(t *testing.T) {
		assert.Empty(t, allOTLPHeaders(nil))
	})

	t.Run("entries without headers returns empty string", func(t *testing.T) {
		entries := []otlpEndpointEntry{{URL: "https://a.example.com"}, {URL: "https://b.example.com"}}
		assert.Empty(t, allOTLPHeaders(entries))
	})

	t.Run("single entry with headers", func(t *testing.T) {
		entries := []otlpEndpointEntry{{URL: "https://a.example.com", Headers: "Authorization=Bearer tok"}}
		assert.Equal(t, "Authorization=Bearer tok", allOTLPHeaders(entries))
	})

	t.Run("multiple entries with headers are comma-joined", func(t *testing.T) {
		entries := []otlpEndpointEntry{
			{URL: "https://a.example.com", Headers: "Authorization=Bearer tok1"},
			{URL: "https://b.example.com", Headers: "X-API-Key=key2"},
		}
		assert.Equal(t, "Authorization=Bearer tok1,X-API-Key=key2", allOTLPHeaders(entries))
	})

	t.Run("entries without headers are skipped", func(t *testing.T) {
		entries := []otlpEndpointEntry{
			{URL: "https://a.example.com", Headers: "Authorization=Bearer tok1"},
			{URL: "https://b.example.com"},
			{URL: "https://c.example.com", Headers: "X-API-Key=key3"},
		}
		assert.Equal(t, "Authorization=Bearer tok1,X-API-Key=key3", allOTLPHeaders(entries))
	})
}

// TestInjectOTLPConfig_MultipleEndpoints verifies the multi-endpoint injection path.
func TestInjectOTLPConfig_MultipleEndpoints(t *testing.T) {
	c := &Compiler{}

	t.Run("injects GH_AW_OTLP_ENDPOINTS for array endpoint", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "https://primary.example.com:4317"},
							map[string]any{"url": "https://secondary.example.com:4317"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotEmpty(t, wd.Env, "Env should be set")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: https://primary.example.com:4317", "first endpoint should be set as primary")
		// GH_AW_OTLP_ENDPOINTS must be single-quoted so YAML parsers treat the
		// leading '[' as a string, not a sequence node.
		assert.Contains(t, wd.Env, "GH_AW_OTLP_ENDPOINTS: '[", "multi-endpoint env var should be single-quoted")
		assert.Contains(t, wd.Env, "primary.example.com", "primary endpoint should appear in GH_AW_OTLP_ENDPOINTS")
		assert.Contains(t, wd.Env, "secondary.example.com", "secondary endpoint should appear in GH_AW_OTLP_ENDPOINTS")
	})

	t.Run("escapes single quotes in GH_AW_OTLP_ENDPOINTS", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{
								"url":     "https://primary.example.com:4317",
								"headers": map[string]any{"Authorization": "Bearer O'Reilly"},
							},
							map[string]any{"url": "https://secondary.example.com:4317"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		assert.Contains(t, wd.Env, "GH_AW_OTLP_ENDPOINTS:", "multi-endpoint env var should be injected")
		assert.Contains(
			t,
			wd.Env,
			"GH_AW_OTLP_ENDPOINTS: '[{\"url\":\"https://primary.example.com:4317\",\"headers\":\"Authorization=Bearer O''Reilly\"}",
			"single quotes must be escaped inside GH_AW_OTLP_ENDPOINTS YAML single-quoted scalar",
		)
	})

	t.Run("adds all static endpoint domains to firewall allowlist", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "https://primary.example.com:4317"},
							map[string]any{"url": "https://secondary.example.com:4317"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotNil(t, wd.NetworkPermissions, "NetworkPermissions should be created")
		assert.Contains(t, wd.NetworkPermissions.Allowed, "primary.example.com")
		assert.Contains(t, wd.NetworkPermissions.Allowed, "secondary.example.com")
	})

	t.Run("sets GH_AW_OTLP_ALL_HEADERS when multiple endpoints have headers", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "https://primary.example.com:4317", "headers": map[string]any{"Authorization": "Bearer tok1"}},
							map[string]any{"url": "https://secondary.example.com:4317", "headers": map[string]any{"Authorization": "Bearer tok2"}},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		assert.Contains(t, wd.Env, "GH_AW_OTLP_ALL_HEADERS:", "all-headers env var should be injected for multiple endpoints")
		assert.True(t, isOTLPHeadersPresent(wd), "isOTLPHeadersPresent should detect GH_AW_OTLP_ALL_HEADERS")
	})

	t.Run("does not set GH_AW_OTLP_ALL_HEADERS for single endpoint (string form)", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": "https://traces.example.com:4317",
						"headers":  map[string]any{"Authorization": "Bearer tok"},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		assert.NotContains(t, wd.Env, "GH_AW_OTLP_ALL_HEADERS", "all-headers var should not be set for single endpoint")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS:", "standard headers var should still be set")
	})

	t.Run("does not set GH_AW_OTLP_ALL_HEADERS for single endpoint (object form)", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": map[string]any{
							"url":     "https://traces.example.com:4317",
							"headers": map[string]any{"Authorization": "Bearer tok"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		assert.NotContains(t, wd.Env, "GH_AW_OTLP_ALL_HEADERS", "all-headers var should not be set for single endpoint")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS:", "standard headers var should still be set")
	})

	t.Run("OTLPEndpoints field is set to JSON-encoded array", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "https://primary.example.com:4317"},
							map[string]any{"url": "https://secondary.example.com:4317"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotEmpty(t, wd.OTLPEndpoints, "OTLPEndpoints field should be set")
		assert.Contains(t, wd.OTLPEndpoints, "primary.example.com")
		assert.Contains(t, wd.OTLPEndpoints, "secondary.example.com")
	})

	t.Run("expression-only endpoints do not add to firewall allowlist", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": []any{
							map[string]any{"url": "${{ secrets.OTLP_ENDPOINT1 }}"},
							map[string]any{"url": "${{ secrets.OTLP_ENDPOINT2 }}"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		assert.Nil(t, wd.NetworkPermissions, "expression endpoints should not add to firewall (NetworkPermissions should be nil)")
	})

	t.Run("object form: injects single endpoint with per-endpoint headers", func(t *testing.T) {
		wd := &WorkflowData{
			RawFrontmatter: map[string]any{
				"observability": map[string]any{
					"otlp": map[string]any{
						"endpoint": map[string]any{
							"url":     "https://traces.example.com:4317",
							"headers": map[string]any{"Authorization": "Bearer tok"},
						},
					},
				},
			},
		}
		c.injectOTLPConfig(wd)

		require.NotEmpty(t, wd.Env)
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com:4317")
		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS: Authorization=Bearer tok")
		assert.Contains(t, wd.Env, "GH_AW_OTLP_ENDPOINTS:")
		require.NotNil(t, wd.NetworkPermissions)
		assert.Contains(t, wd.NetworkPermissions.Allowed, "traces.example.com")
	})
}

// TestIsOTLPHeadersPresent_AllHeaders verifies that isOTLPHeadersPresent detects
// GH_AW_OTLP_ALL_HEADERS in addition to OTEL_EXPORTER_OTLP_HEADERS.
func TestIsOTLPHeadersPresent_AllHeaders(t *testing.T) {
	t.Run("detects GH_AW_OTLP_ALL_HEADERS", func(t *testing.T) {
		wd := &WorkflowData{
			Env: "env:\n  OTEL_EXPORTER_OTLP_ENDPOINT: https://traces.example.com\n  GH_AW_OTLP_ALL_HEADERS: Authorization=Bearer tok1,Authorization=Bearer tok2",
		}
		assert.True(t, isOTLPHeadersPresent(wd), "should detect GH_AW_OTLP_ALL_HEADERS")
	})
}

// TestExtractRawOTLPEndpointMaps verifies that all three endpoint forms (string, object, array)
// are extracted as raw maps with original header format preserved.
func TestExtractRawOTLPEndpointMaps(t *testing.T) {
	tests := []struct {
		name string
		obs  map[string]any
		want []map[string]any
	}{
		{
			name: "nil map returns nil",
			obs:  nil,
			want: nil,
		},
		{
			name: "empty map returns nil",
			obs:  map[string]any{},
			want: nil,
		},
		{
			name: "missing otlp key returns nil",
			obs:  map[string]any{"other": "value"},
			want: nil,
		},
		{
			name: "string form without headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
				},
			},
			want: []map[string]any{
				{"url": "https://traces.example.com:4317"},
			},
		},
		{
			name: "string form with top-level headers preserved as map",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
					"headers":  map[string]any{"Authorization": "Bearer tok"},
				},
			},
			want: []map[string]any{
				{"url": "https://traces.example.com:4317", "headers": map[string]any{"Authorization": "Bearer tok"}},
			},
		},
		{
			name: "object form with headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{
						"url":     "https://traces.example.com:4317",
						"headers": map[string]any{"X-API-Key": "key1"},
					},
				},
			},
			want: []map[string]any{
				{"url": "https://traces.example.com:4317", "headers": map[string]any{"X-API-Key": "key1"}},
			},
		},
		{
			name: "array form with multiple endpoints",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": "https://primary.example.com:4317"},
						map[string]any{"url": "https://secondary.example.com:4317", "headers": map[string]any{"X-Key": "v"}},
					},
				},
			},
			want: []map[string]any{
				{"url": "https://primary.example.com:4317"},
				{"url": "https://secondary.example.com:4317", "headers": map[string]any{"X-Key": "v"}},
			},
		},
		{
			name: "array form skips entries with empty URL",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": ""},
						map[string]any{"url": "https://valid.example.com:4317"},
					},
				},
			},
			want: []map[string]any{
				{"url": "https://valid.example.com:4317"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRawOTLPEndpointMaps(tt.obs)
			assert.Equal(t, tt.want, got, "extractRawOTLPEndpointMaps")
		})
	}
}
