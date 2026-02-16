//go:build !integration

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyPAT(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected PATType
	}{
		{
			name:     "fine-grained PAT",
			token:    "github_pat_abc123xyz",
			expected: PATTypeFineGrained,
		},
		{
			name:     "classic PAT",
			token:    "ghp_abc123xyz",
			expected: PATTypeClassic,
		},
		{
			name:     "OAuth token",
			token:    "gho_abc123xyz",
			expected: PATTypeOAuth,
		},
		{
			name:     "unknown token - random string",
			token:    "random_token_123",
			expected: PATTypeUnknown,
		},
		{
			name:     "unknown token - empty",
			token:    "",
			expected: PATTypeUnknown,
		},
		{
			name:     "partial prefix - github_pa",
			token:    "github_pa_abc123",
			expected: PATTypeUnknown,
		},
		{
			name:     "partial prefix - gh_",
			token:    "gh_abc123",
			expected: PATTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyPAT(tt.token)
			assert.Equal(t, tt.expected, result, "ClassifyPAT should return correct type")
		})
	}
}

func TestPATType_IsFineGrained(t *testing.T) {
	assert.True(t, PATTypeFineGrained.IsFineGrained(), "fine-grained should return true")
	assert.False(t, PATTypeClassic.IsFineGrained(), "classic should return false")
	assert.False(t, PATTypeOAuth.IsFineGrained(), "OAuth should return false")
	assert.False(t, PATTypeUnknown.IsFineGrained(), "unknown should return false")
}

func TestPATType_IsValid(t *testing.T) {
	assert.True(t, PATTypeFineGrained.IsValid(), "fine-grained should be valid")
	assert.True(t, PATTypeClassic.IsValid(), "classic should be valid")
	assert.True(t, PATTypeOAuth.IsValid(), "OAuth should be valid")
	assert.False(t, PATTypeUnknown.IsValid(), "unknown should not be valid")
}

func TestIsFineGrainedPAT(t *testing.T) {
	assert.True(t, IsFineGrainedPAT("github_pat_abc123"), "should identify fine-grained PAT")
	assert.False(t, IsFineGrainedPAT("ghp_abc123"), "should not identify classic PAT as fine-grained")
	assert.False(t, IsFineGrainedPAT("gho_abc123"), "should not identify OAuth token as fine-grained")
	assert.False(t, IsFineGrainedPAT("random"), "should not identify unknown token as fine-grained")
}

func TestIsClassicPAT(t *testing.T) {
	assert.True(t, IsClassicPAT("ghp_abc123"), "should identify classic PAT")
	assert.False(t, IsClassicPAT("github_pat_abc123"), "should not identify fine-grained PAT as classic")
	assert.False(t, IsClassicPAT("gho_abc123"), "should not identify OAuth token as classic")
	assert.False(t, IsClassicPAT("random"), "should not identify unknown token as classic")
}

func TestIsOAuthToken(t *testing.T) {
	assert.True(t, IsOAuthToken("gho_abc123"), "should identify OAuth token")
	assert.False(t, IsOAuthToken("github_pat_abc123"), "should not identify fine-grained PAT as OAuth")
	assert.False(t, IsOAuthToken("ghp_abc123"), "should not identify classic PAT as OAuth")
	assert.False(t, IsOAuthToken("random"), "should not identify unknown token as OAuth")
}

func TestValidateCopilotPAT(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid fine-grained PAT",
			token:       "github_pat_abc123xyz",
			expectError: false,
		},
		{
			name:        "classic PAT should fail",
			token:       "ghp_abc123xyz",
			expectError: true,
			errorMsg:    "classic personal access tokens",
		},
		{
			name:        "OAuth token should fail",
			token:       "gho_abc123xyz",
			expectError: true,
			errorMsg:    "OAuth tokens",
		},
		{
			name:        "unknown token should fail",
			token:       "random_token",
			expectError: true,
			errorMsg:    "unrecognized token format",
		},
		{
			name:        "empty token should fail",
			token:       "",
			expectError: true,
			errorMsg:    "unrecognized token format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCopilotPAT(tt.token)
			if tt.expectError {
				require.Error(t, err, "should return error for invalid token")
				assert.Contains(t, err.Error(), tt.errorMsg, "error message should contain expected text")
			} else {
				assert.NoError(t, err, "should not return error for valid token")
			}
		})
	}
}

func TestGetPATTypeDescription(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "fine-grained PAT",
			token:    "github_pat_abc123",
			expected: "fine-grained personal access token",
		},
		{
			name:     "classic PAT",
			token:    "ghp_abc123",
			expected: "classic personal access token",
		},
		{
			name:     "OAuth token",
			token:    "gho_abc123",
			expected: "OAuth token",
		},
		{
			name:     "unknown token",
			token:    "random",
			expected: "unknown token type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPATTypeDescription(tt.token)
			assert.Equal(t, tt.expected, result, "should return correct description")
		})
	}
}
