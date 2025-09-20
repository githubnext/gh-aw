package workflow

import (
	"testing"
)

func TestParseUploadAssetConfig(t *testing.T) {
	c := &Compiler{}

	tests := []struct {
		name     string
		input    map[string]any
		expected *UploadAssetsConfig
	}{
		{
			name: "upload-asset config with custom values",
			input: map[string]any{
				"upload-assets": map[string]any{
					"branch":       "my-assets/${{ github.event.repository.name }}",
					"max-size":     5120,
					"allowed-exts": []any{".jpg", ".png", ".txt"},
					"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &UploadAssetsConfig{
				BranchName:  "my-assets/${{ github.event.repository.name }}",
				MaxSizeKB:   5120,
				AllowedExts: []string{".jpg", ".png", ".txt"},
				GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
			},
		},
		{
			name: "upload-asset config with defaults",
			input: map[string]any{
				"upload-assets": map[string]any{},
			},
			expected: &UploadAssetsConfig{
				BranchName:  "assets/${{ github.workflow }}",
				MaxSizeKB:   10240,
				AllowedExts: []string{".jpg", ".jpeg", ".png", ".webp", ".mp4", ".webm", ".txt", ".md"},
				GitHubToken: "",
			},
		},
		{
			name:     "no upload-asset config",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.parseUploadAssetConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			if result.BranchName != tt.expected.BranchName {
				t.Errorf("BranchName: expected %s, got %s", tt.expected.BranchName, result.BranchName)
			}

			if result.MaxSizeKB != tt.expected.MaxSizeKB {
				t.Errorf("MaxSizeKB: expected %d, got %d", tt.expected.MaxSizeKB, result.MaxSizeKB)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken: expected %s, got %s", tt.expected.GitHubToken, result.GitHubToken)
			}

			if len(result.AllowedExts) != len(tt.expected.AllowedExts) {
				t.Errorf("AllowedExts length: expected %d, got %d", len(tt.expected.AllowedExts), len(result.AllowedExts))
			}
		})
	}
}

func TestHasSafeOutputsEnabledWithUploadAsset(t *testing.T) {
	// Test that UploadAsset is properly detected
	config := &SafeOutputsConfig{
		UploadAssets: &UploadAssetsConfig{},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("Expected UploadAsset to be detected as enabled safe output")
	}

	// Test with nil config
	if HasSafeOutputsEnabled(nil) {
		t.Error("Expected nil config to return false")
	}

	// Test with empty config
	emptyConfig := &SafeOutputsConfig{}
	if HasSafeOutputsEnabled(emptyConfig) {
		t.Error("Expected empty config to return false")
	}
}
