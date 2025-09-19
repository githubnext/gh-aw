package workflow

import (
	"testing"
)

func TestParsePublishAssetsConfig(t *testing.T) {
	c := &Compiler{}

	tests := []struct {
		name     string
		input    map[string]any
		expected *PublishAssetsConfig
	}{
		{
			name: "basic publish-assets config",
			input: map[string]any{
				"publish-assets": nil,
			},
			expected: &PublishAssetsConfig{
				BranchName:  "assets/${{ github.workflow }}",
				MaxSizeKB:   10240,
				AllowedExts: getDefaultAllowedExtensions(),
			},
		},
		{
			name: "publish-assets config with custom values",
			input: map[string]any{
				"publish-assets": map[string]any{
					"branch-name":   "my-assets/${{ github.event.repository.name }}",
					"max-size-kb":   5120,
					"allowed-exts":  []any{".jpg", ".png", ".txt"},
					"github-token":  "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &PublishAssetsConfig{
				BranchName:  "my-assets/${{ github.event.repository.name }}",
				MaxSizeKB:   5120,
				AllowedExts: []string{".jpg", ".png", ".txt"},
				GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
			},
		},
		{
			name:     "no publish-assets config",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.parsePublishAssetsConfig(tt.input)

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

func TestGetDefaultAllowedExtensions(t *testing.T) {
	exts := getDefaultAllowedExtensions()

	// Check that we have some reasonable defaults
	if len(exts) == 0 {
		t.Error("Expected default extensions, got empty list")
	}

	// Check for some expected extensions
	expectedExts := []string{".jpg", ".png", ".pdf", ".txt", ".json"}
	for _, expected := range expectedExts {
		found := false
		for _, ext := range exts {
			if ext == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected default extension %s not found in %v", expected, exts)
		}
	}

	// Check that executable extensions are not included
	forbiddenExts := []string{".exe", ".bat", ".sh", ".bin"}
	for _, forbidden := range forbiddenExts {
		for _, ext := range exts {
			if ext == forbidden {
				t.Errorf("Forbidden extension %s found in default extensions", forbidden)
			}
		}
	}
}

func TestHasSafeOutputsEnabledWithPublishAssets(t *testing.T) {
	// Test that PublishAssets is properly detected
	config := &SafeOutputsConfig{
		PublishAssets: &PublishAssetsConfig{},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("Expected PublishAssets to be detected as enabled safe output")
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
