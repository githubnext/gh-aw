//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestGetSandboxDisableJustification tests the full justification validation logic,
// including all the rejection cases required by the acceptance criteria:
//   - boolean true fails (no longer a legacy shorthand)
//   - expressions fail
//   - too-short strings fail
//   - whitespace-padded strings fail
//   - a 20+ character literal reason passes
func TestGetSandboxDisableJustification(t *testing.T) {
	makeData := func(value any) *WorkflowData {
		return &WorkflowData{
			Features: map[string]any{
				"dangerously-disable-sandbox-agent": value,
			},
		}
	}

	t.Run("boolean true is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData(true))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "string", "should explain that a string is required")
	})

	t.Run("boolean false is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData(false))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "string", "should explain that a string is required")
	})

	t.Run("empty string is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData(""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "20", "should mention minimum length")
	})

	t.Run("short string is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData("too short"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "20", "should mention minimum length")
	})

	t.Run("whitespace-padded short string is rejected", func(t *testing.T) {
		// 22 spaces - long enough on paper but collapses to empty after TrimSpace
		_, err := getSandboxDisableJustification(makeData("                      "))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "20", "should mention minimum length")
	})

	t.Run("whitespace-padded string where trimmed is below minimum is rejected", func(t *testing.T) {
		// "short" padded with whitespace to 25 total chars still fails (trimmed is 5)
		_, err := getSandboxDisableJustification(makeData("          short          "))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "20", "should mention minimum length")
	})

	t.Run("GitHub Actions expression is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData("${{ inputs.reason }}"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expressions")
	})

	t.Run("longer expression with surrounding text is rejected", func(t *testing.T) {
		_, err := getSandboxDisableJustification(makeData("reason: ${{ inputs.reason }} end"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expressions")
	})

	t.Run("20+ character literal reason passes", func(t *testing.T) {
		justification, err := getSandboxDisableJustification(makeData("controlled environment with no internet access"))
		require.NoError(t, err)
		assert.Equal(t, "controlled environment with no internet access", justification)
	})

	t.Run("justification is trimmed before return", func(t *testing.T) {
		justification, err := getSandboxDisableJustification(makeData("  controlled environment with no internet access  "))
		require.NoError(t, err)
		assert.Equal(t, "controlled environment with no internet access", justification)
	})

	t.Run("feature missing returns error", func(t *testing.T) {
		_, err := getSandboxDisableJustification(&WorkflowData{Features: map[string]any{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("nil features returns error", func(t *testing.T) {
		_, err := getSandboxDisableJustification(&WorkflowData{})
		require.Error(t, err)
	})

	t.Run("nil workflow data returns error", func(t *testing.T) {
		_, err := getSandboxDisableJustification(nil)
		require.Error(t, err)
	})
}

// TestValidateSandboxConfigTrustBoundaryMessage tests that the compiler diagnostic
// says the sandbox removal is a trust boundary change, not just a validator check.
func TestValidateSandboxConfigTrustBoundaryMessage(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{Disabled: true},
		},
		// No Features — validation must fail
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "trust boundary", "diagnostic must say the sandbox removal removes a trust boundary")
	assert.Contains(t, errMsg, "dangerously-disable-sandbox-agent", "diagnostic must name the required feature flag")
}

// TestValidateSandboxConfigStoresJustification tests that a valid justification is
// stored in AgentSandboxConfig.DisableReason for downstream diagnostics and audit.
func TestValidateSandboxConfigStoresJustification(t *testing.T) {
	const reason = "controlled environment with no internet access"
	workflowData := &WorkflowData{
		Features: map[string]any{
			"dangerously-disable-sandbox-agent": reason,
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{Disabled: true},
		},
	}

	err := validateSandboxConfig(workflowData)
	require.NoError(t, err, "valid justification should pass validation")
	assert.Equal(t, reason, workflowData.SandboxConfig.Agent.DisableReason,
		"justification must be stored on AgentSandboxConfig for audit/logging")
}
