package workflow

import (
	"strings"
	"testing"
)

func TestResolveAWFPath(t *testing.T) {
	tests := []struct {
		name         string
		customPath   string
		expectedPath string
	}{
		{
			name:         "empty path returns default",
			customPath:   "",
			expectedPath: "/usr/local/bin/awf",
		},
		{
			name:         "absolute path returned as-is",
			customPath:   "/usr/local/bin/awf-custom",
			expectedPath: "/usr/local/bin/awf-custom",
		},
		{
			name:         "absolute path with nested directory",
			customPath:   "/opt/tools/bin/awf",
			expectedPath: "/opt/tools/bin/awf",
		},
		{
			name:         "relative path resolved to GITHUB_WORKSPACE",
			customPath:   "bin/awf",
			expectedPath: "${GITHUB_WORKSPACE}/bin/awf",
		},
		{
			name:         "relative path without subdirectory",
			customPath:   "awf",
			expectedPath: "${GITHUB_WORKSPACE}/awf",
		},
		{
			name:         "relative path with nested directories",
			customPath:   "tools/bin/awf",
			expectedPath: "${GITHUB_WORKSPACE}/tools/bin/awf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAWFPath(tt.customPath)
			if result != tt.expectedPath {
				t.Errorf("resolveAWFPath(%q) = %q, want %q", tt.customPath, result, tt.expectedPath)
			}
		})
	}
}

func TestGetAWFBinaryPath(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		expectedPath   string
	}{
		{
			name:           "nil config returns default",
			firewallConfig: nil,
			expectedPath:   "awf",
		},
		{
			name:           "config without path returns default",
			firewallConfig: &FirewallConfig{Enabled: true},
			expectedPath:   "awf",
		},
		{
			name:           "config with empty path returns default",
			firewallConfig: &FirewallConfig{Enabled: true, Path: ""},
			expectedPath:   "awf",
		},
		{
			name:           "config with absolute path",
			firewallConfig: &FirewallConfig{Enabled: true, Path: "/custom/path/awf"},
			expectedPath:   "/custom/path/awf",
		},
		{
			name:           "config with relative path",
			firewallConfig: &FirewallConfig{Enabled: true, Path: "bin/awf"},
			expectedPath:   "${GITHUB_WORKSPACE}/bin/awf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAWFBinaryPath(tt.firewallConfig)
			if result != tt.expectedPath {
				t.Errorf("getAWFBinaryPath() = %q, want %q", result, tt.expectedPath)
			}
		})
	}
}

func TestGenerateAWFPathValidationStep(t *testing.T) {
	tests := []struct {
		name           string
		customPath     string
		expectedChecks []string
	}{
		{
			name:       "absolute path validation",
			customPath: "/usr/local/bin/awf-custom",
			expectedChecks: []string{
				"Validate custom AWF binary",
				"/usr/local/bin/awf-custom",
				"if [ ! -f",
				"if [ ! -x",
				"--version",
			},
		},
		{
			name:       "relative path validation",
			customPath: "bin/awf",
			expectedChecks: []string{
				"Validate custom AWF binary",
				"${GITHUB_WORKSPACE}/bin/awf",
				"if [ ! -f",
				"if [ ! -x",
				"--version",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := generateAWFPathValidationStep(tt.customPath)
			stepContent := strings.Join(step, "\n")

			for _, check := range tt.expectedChecks {
				if !strings.Contains(stepContent, check) {
					t.Errorf("generateAWFPathValidationStep(%q) missing expected content: %q\nGot:\n%s", tt.customPath, check, stepContent)
				}
			}
		})
	}
}

func TestFirewallConfigPathExtraction(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name         string
		firewall     any
		expectedPath string
	}{
		{
			name: "path from object config",
			firewall: map[string]any{
				"path": "/custom/awf",
			},
			expectedPath: "/custom/awf",
		},
		{
			name: "path with other config options",
			firewall: map[string]any{
				"path":      "bin/awf",
				"log-level": "debug",
			},
			expectedPath: "bin/awf",
		},
		{
			name: "no path in config",
			firewall: map[string]any{
				"version": "v1.0.0",
			},
			expectedPath: "",
		},
		{
			name:         "boolean config has no path",
			firewall:     true,
			expectedPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractFirewallConfig(tt.firewall)
			if config == nil {
				if tt.expectedPath != "" {
					t.Errorf("extractFirewallConfig() returned nil, expected path %q", tt.expectedPath)
				}
				return
			}

			if config.Path != tt.expectedPath {
				t.Errorf("extractFirewallConfig().Path = %q, want %q", config.Path, tt.expectedPath)
			}
		})
	}
}

func TestVersionIgnoredWhenPathSet(t *testing.T) {
	// When both path and version are specified, path takes precedence
	// and version is effectively ignored (no download happens)
	config := &FirewallConfig{
		Enabled: true,
		Path:    "/custom/awf",
		Version: "v1.0.0", // Should be ignored
	}

	path := getAWFBinaryPath(config)
	if path != "/custom/awf" {
		t.Errorf("getAWFBinaryPath() should use custom path, got %q", path)
	}
}
