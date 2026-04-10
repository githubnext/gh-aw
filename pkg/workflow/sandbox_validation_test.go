//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestSandboxTypeEnumValidation tests that sandbox type enum values are correctly validated
func TestSandboxTypeEnumValidation(t *testing.T) {
	tests := []struct {
		name        string
		sandboxType SandboxType
		expectValid bool
	}{
		// Valid enum values
		{
			name:        "valid type: awf",
			sandboxType: SandboxTypeAWF,
			expectValid: true,
		},
		{
			name:        "valid type: default (backward compat)",
			sandboxType: SandboxTypeDefault,
			expectValid: true,
		},
		// Invalid enum values
		{
			name:        "invalid type: AWF (uppercase)",
			sandboxType: "AWF",
			expectValid: false,
		},
		{
			name:        "invalid type: Default (mixed case)",
			sandboxType: "Default",
			expectValid: false,
		},
		{
			name:        "invalid type: empty string",
			sandboxType: "",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupportedSandboxType(tt.sandboxType)
			if result != tt.expectValid {
				t.Errorf("isSupportedSandboxType(%q) = %v, want %v", tt.sandboxType, result, tt.expectValid)
			}
		})
	}
}

// TestSandboxTypeCaseSensitivity tests that sandbox types are case-sensitive
func TestSandboxTypeCaseSensitivity(t *testing.T) {
	caseSensitiveTests := []struct {
		name        string
		sandboxType SandboxType
		shouldMatch bool
	}{
		{name: "lowercase awf matches", sandboxType: "awf", shouldMatch: true},
		{name: "uppercase AWF does not match", sandboxType: "AWF", shouldMatch: false},
		{name: "mixed case Awf does not match", sandboxType: "Awf", shouldMatch: false},
		{name: "lowercase default matches", sandboxType: "default", shouldMatch: true},
		{name: "uppercase DEFAULT does not match", sandboxType: "DEFAULT", shouldMatch: false},
	}

	for _, tt := range caseSensitiveTests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupportedSandboxType(tt.sandboxType)
			if result != tt.shouldMatch {
				t.Errorf("isSupportedSandboxType(%q) = %v, want %v", tt.sandboxType, result, tt.shouldMatch)
			}
		})
	}
}

// TestValidateSandboxConfig_MCPGatewayOTLPCompatibility verifies that validateSandboxConfig
// emits a compile-time error when OTLP headers are configured but the pinned gateway version
// is older than MCPGatewayStringHeadersMinVersion (v0.2.17).
//
// Gateway versions before v0.2.17 require opentelemetry.headers to be an object, but the
// compiler always emits it as a string. Pinning an old version with OTLP headers configured
// would cause a schema validation failure at gateway startup.
func TestValidateSandboxConfig_MCPGatewayOTLPCompatibility(t *testing.T) {
	// workflowDataWithOTLPHeaders returns a WorkflowData with OTEL_EXPORTER_OTLP_HEADERS
	// injected into the env block, simulating a workflow that has OTLP headers configured.
	workflowDataWithOTLPHeaders := func(mcpVersion string) *WorkflowData {
		wd := &WorkflowData{
			// Simulate injectOTLPConfig having run and injected the headers env var.
			Env: "  OTEL_EXPORTER_OTLP_ENDPOINT: https://example.com\n  OTEL_EXPORTER_OTLP_HEADERS: ${{ secrets.OTEL_HEADERS }}",
		}
		if mcpVersion != "" {
			wd.SandboxConfig = &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{Version: mcpVersion},
			}
		}
		return wd
	}

	tests := []struct {
		name         string
		workflowData *WorkflowData
		wantErr      bool
		errContains  string
	}{
		{
			name:         "no OTLP headers - no error even with old gateway version",
			workflowData: &WorkflowData{SandboxConfig: &SandboxConfig{MCP: &MCPGatewayRuntimeConfig{Version: "v0.2.16"}}},
			wantErr:      false,
		},
		{
			name:         "OTLP headers with no pinned version - uses default, no error",
			workflowData: workflowDataWithOTLPHeaders(""),
			wantErr:      false,
		},
		{
			name:         "OTLP headers with v0.2.17 - supported, no error",
			workflowData: workflowDataWithOTLPHeaders("v0.2.17"),
			wantErr:      false,
		},
		{
			name:         "OTLP headers with v0.2.18 - supported, no error",
			workflowData: workflowDataWithOTLPHeaders("v0.2.18"),
			wantErr:      false,
		},
		{
			name:         "OTLP headers with latest - always supported, no error",
			workflowData: workflowDataWithOTLPHeaders("latest"),
			wantErr:      false,
		},
		{
			name:         "OTLP headers with v0.2.16 - not supported, error",
			workflowData: workflowDataWithOTLPHeaders("v0.2.16"),
			wantErr:      true,
			errContains:  "does not support string OTLP headers",
		},
		{
			name:         "OTLP headers with v0.2.0 - not supported, error",
			workflowData: workflowDataWithOTLPHeaders("v0.2.0"),
			wantErr:      true,
			errContains:  "does not support string OTLP headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSandboxConfig(tt.workflowData)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}
