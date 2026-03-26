//go:build !integration

// Tests for argument injection prevention: package and image names starting with '-'
// must be rejected before being passed to exec.Command calls (npm, pip, uv, docker).

package workflow

import (
	"strings"
	"testing"
)

// TestRejectHyphenPrefixPackages tests the shared helper that guards against
// argument injection in all package validation paths.
func TestRejectHyphenPrefixPackages(t *testing.T) {
	tests := []struct {
		name        string
		packages    []string
		kind        string
		expectError bool
		errContains string
	}{
		{
			name:        "empty list is accepted",
			packages:    []string{},
			kind:        "npx",
			expectError: false,
		},
		{
			name:        "valid package names are accepted",
			packages:    []string{"express", "@scope/pkg", "requests==2.28.0"},
			kind:        "pip",
			expectError: false,
		},
		{
			name:        "single hyphen prefix is rejected",
			packages:    []string{"-exploit"},
			kind:        "npx",
			expectError: true,
			errContains: "must not start with '-'",
		},
		{
			name:        "double hyphen prefix is rejected",
			packages:    []string{"--privileged"},
			kind:        "uv",
			expectError: true,
			errContains: "must not start with '-'",
		},
		{
			name:        "mixed list with one invalid is rejected",
			packages:    []string{"valid-pkg", "-exploit"},
			kind:        "pip",
			expectError: true,
			errContains: "-exploit",
		},
		{
			name:        "package with version specifier starting with hyphen is rejected",
			packages:    []string{"-exploit==1.0"},
			kind:        "uv",
			expectError: true,
			errContains: "-exploit==1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rejectHyphenPrefixPackages(tt.packages, tt.kind)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for packages %v but got none", tt.packages)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for packages %v: %v", tt.packages, err)
				}
			}
		})
	}
}

// TestValidateDockerImage_RejectsHyphenPrefix tests that Docker image names starting
// with '-' are rejected without invoking docker. This is the primary injection path
// because image names are taken directly from frontmatter (not filtered by extractors).
func TestValidateDockerImage_RejectsHyphenPrefix(t *testing.T) {
	tests := []struct {
		name        string
		image       string
		errContains string
	}{
		{
			name:        "single hyphen prefix",
			image:       "-exploit",
			errContains: "must not start with '-'",
		},
		{
			name:        "double hyphen prefix",
			image:       "--privileged",
			errContains: "must not start with '-'",
		},
		{
			name:        "looks like a docker flag",
			image:       "-rm",
			errContains: "must not start with '-'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDockerImage(tt.image, false)
			if err == nil {
				t.Errorf("expected error for image %q but got none", tt.image)
				return
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
			}
		})
	}
}

// TestExtractNpxFromCommands_HyphenPrefixFiltered shows that the npx package
// extractor already filters out names starting with '-' (treating them as flags),
// so they never reach the validation guard.
func TestExtractNpxFromCommands_HyphenPrefixFiltered(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		wantLen  int
	}{
		{
			name:     "lone hyphen-prefix arg yields no packages",
			commands: "npx -exploit",
			wantLen:  0,
		},
		{
			name:     "double-hyphen arg yields no packages",
			commands: "npx --exploit",
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractNpxFromCommands(tt.commands)
			if len(packages) != tt.wantLen {
				t.Errorf("expected %d packages, got %d: %v", tt.wantLen, len(packages), packages)
			}
		})
	}
}

// TestExtractPipFromCommands_HyphenPrefixFiltered shows that the pip package
// extractor already filters out names starting with '-', so they never reach
// the validation guard.
func TestExtractPipFromCommands_HyphenPrefixFiltered(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		wantLen  int
	}{
		{
			name:     "hyphen-prefix pip package yields no packages",
			commands: "pip install -exploit",
			wantLen:  0,
		},
		{
			name:     "double-hyphen pip package yields no packages",
			commands: "pip install --exploit",
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractPipFromCommands(tt.commands)
			if len(packages) != tt.wantLen {
				t.Errorf("expected %d packages, got %d: %v", tt.wantLen, len(packages), packages)
			}
		})
	}
}

// TestCollectPackagesFromWorkflow_HyphenPrefixInArgs verifies that the MCP tool
// args extractor also filters out names starting with '-', providing an additional
// layer of defense for the structured args format.
func TestCollectPackagesFromWorkflow_HyphenPrefixInArgs(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"test-tool": map[string]any{
				"command": "npx",
				"args":    []any{"-exploit", "safe-package"},
			},
		},
	}

	packages := collectPackagesFromWorkflow(workflowData, extractNpxFromCommands, "npx")

	for _, pkg := range packages {
		if strings.HasPrefix(pkg, "-") {
			t.Errorf("package starting with '-' should not be collected: %s", pkg)
		}
	}
}
