//go:build !integration

package workflow

import (
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
