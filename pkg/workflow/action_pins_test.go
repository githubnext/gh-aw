package workflow

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestDefaultActionVersionsExist verifies that all default action versions are defined
func TestDefaultActionVersionsExist(t *testing.T) {
	if len(defaultActionVersions) == 0 {
		t.Fatal("No default action versions defined")
	}

	// Verify each entry has required fields
	for repo, version := range defaultActionVersions {
		// Verify the repo is not empty
		if repo == "" {
			t.Errorf("Default action version has empty repo field")
			continue
		}

		// Verify the version is not empty
		if version == "" {
			t.Errorf("Missing version for %s", repo)
		}

		// Verify version starts with 'v'
		if !strings.HasPrefix(version, "v") {
			t.Errorf("Version for %s should start with 'v': %s", repo, version)
		}
	}
}

// TestGetActionPinReturnsValidSHA tests that GetActionPin returns valid SHA references
// This test skips actual resolution if gh CLI is not available
func TestGetActionPinReturnsValidSHA(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("Skipping test: gh CLI not available")
	}

	// Test a few known actions
	testCases := []struct {
		repo    string
		version string
	}{
		{"actions/checkout", "v5"},
		{"actions/setup-node", "v6"},
		{"actions/cache", "v4"},
	}

	// Create WorkflowData with resolver for testing
	data := &WorkflowData{
		ActionCache:    NewActionCache("."),
		ActionResolver: NewActionResolver(NewActionCache(".")),
	}

	for _, tc := range testCases {
		t.Run(tc.repo, func(t *testing.T) {
			result := GetActionPin(tc.repo, data)

			// If resolution failed, we might get empty string
			if result == "" {
				t.Logf("Warning: GetActionPin(%s) returned empty string (resolution may have failed)", tc.repo)
				return
			}

			// Check that the result contains a SHA (40-char hex after @ and before #)
			// Format is: repo@sha # version
			parts := strings.Split(result, "@")
			if len(parts) != 2 {
				t.Errorf("GetActionPin(%s) = %s, expected format repo@sha # version", tc.repo, result)
				return
			}

			// Extract SHA (before the comment marker " # ")
			shaAndComment := parts[1]
			commentIdx := strings.Index(shaAndComment, " # ")
			if commentIdx == -1 {
				t.Errorf("GetActionPin(%s) = %s, expected comment with version tag", tc.repo, result)
				return
			}

			sha := shaAndComment[:commentIdx]

			// All action pins should have valid SHAs
			if !isValidSHA(sha) {
				t.Errorf("GetActionPin(%s) = %s, expected SHA to be 40-char hex", tc.repo, result)
			}
		})
	}
}

// TestGetActionPinFallback tests that GetActionPin returns empty string for unknown actions
func TestGetActionPinFallback(t *testing.T) {
	data := &WorkflowData{}
	result := GetActionPin("unknown/action", data)
	expected := ""
	if result != expected {
		t.Errorf("GetActionPin(unknown/action) = %s, want %s (empty string)", result, expected)
	}
}

// isValidSHA checks if a string is a valid 40-character hexadecimal SHA
func isValidSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	matched, _ := regexp.MatchString("^[0-9a-f]{40}$", s)
	return matched
}

// TestExtractActionRepo tests the extractActionRepo function
func TestExtractActionRepo(t *testing.T) {
	tests := []struct {
		name     string
		uses     string
		expected string
	}{
		{
			name:     "action with version tag",
			uses:     "actions/checkout@v4",
			expected: "actions/checkout",
		},
		{
			name:     "action with SHA",
			uses:     "actions/setup-node@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			expected: "actions/setup-node",
		},
		{
			name:     "action with subpath and version",
			uses:     "github/codeql-action/upload-sarif@v3",
			expected: "github/codeql-action/upload-sarif",
		},
		{
			name:     "action without version",
			uses:     "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "action with branch ref",
			uses:     "actions/setup-python@main",
			expected: "actions/setup-python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractActionRepo(tt.uses)
			if result != tt.expected {
				t.Errorf("extractActionRepo(%q) = %q, want %q", tt.uses, result, tt.expected)
			}
		})
	}
}

// TestExtractActionVersion tests the extractActionVersion function
func TestExtractActionVersion(t *testing.T) {
	tests := []struct {
		name     string
		uses     string
		expected string
	}{
		{
			name:     "action with version tag",
			uses:     "actions/checkout@v4",
			expected: "v4",
		},
		{
			name:     "action with SHA",
			uses:     "actions/setup-node@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			expected: "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
		},
		{
			name:     "action with subpath and version",
			uses:     "github/codeql-action/upload-sarif@v3",
			expected: "v3",
		},
		{
			name:     "action without version",
			uses:     "actions/checkout",
			expected: "",
		},
		{
			name:     "action with branch ref",
			uses:     "actions/setup-python@main",
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractActionVersion(tt.uses)
			if result != tt.expected {
				t.Errorf("extractActionVersion(%q) = %q, want %q", tt.uses, result, tt.expected)
			}
		})
	}
}

// TestApplyActionPinToStep tests the ApplyActionPinToStep function
func TestApplyActionPinToStep(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("Skipping test: gh CLI not available")
	}

	tests := []struct {
		name         string
		stepMap      map[string]any
		expectPinned bool
	}{
		{
			name: "step with pinned action (checkout)",
			stepMap: map[string]any{
				"name": "Checkout code",
				"uses": "actions/checkout@v5",
			},
			expectPinned: true,
		},
		{
			name: "step with unpinned action",
			stepMap: map[string]any{
				"name": "Custom action",
				"uses": "my-org/my-action@v1",
			},
			expectPinned: false,
		},
		{
			name: "step without uses field",
			stepMap: map[string]any{
				"name": "Run command",
				"run":  "echo hello",
			},
			expectPinned: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a WorkflowData with resolver for testing
			data := &WorkflowData{
				ActionCache:    NewActionCache("."),
				ActionResolver: NewActionResolver(NewActionCache(".")),
			}
			result := ApplyActionPinToStep(tt.stepMap, data)

			// Check if uses field exists in result
			if uses, hasUses := result["uses"]; hasUses {
				usesStr, ok := uses.(string)
				if !ok {
					t.Errorf("ApplyActionPinToStep returned non-string uses field")
					return
				}

				// If the action was expected to be pinned, verify format
				if tt.expectPinned && usesStr != "" {
					// Should contain @ and # for pinned format
					if !strings.Contains(usesStr, "@") || !strings.Contains(usesStr, " # ") {
						t.Errorf("ApplyActionPinToStep uses = %q, expected pinned format 'repo@sha # version'", usesStr)
					}
				}

				// Verify other fields are preserved
				if len(result) != len(tt.stepMap) {
					t.Errorf("ApplyActionPinToStep changed number of fields: got %d, want %d", len(result), len(tt.stepMap))
				}
				for k := range tt.stepMap {
					if _, exists := result[k]; !exists {
						t.Errorf("ApplyActionPinToStep lost field %q", k)
					}
				}
			}
		})
	}
}

// TestGetActionPinByRepo tests the GetActionPinByRepo function
func TestGetActionPinByRepo(t *testing.T) {
	tests := []struct {
		repo         string
		expectExists bool
		expectRepo   string
		expectVer    string
	}{
		{
			repo:         "actions/checkout",
			expectExists: true,
			expectRepo:   "actions/checkout",
			expectVer:    "v5",
		},
		{
			repo:         "actions/setup-node",
			expectExists: true,
			expectRepo:   "actions/setup-node",
			expectVer:    "v6",
		},
		{
			repo:         "unknown/action",
			expectExists: false,
		},
		{
			repo:         "",
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			pin, exists := GetActionPinByRepo(tt.repo, nil)

			if exists != tt.expectExists {
				t.Errorf("GetActionPinByRepo(%s) exists = %v, want %v", tt.repo, exists, tt.expectExists)
			}

			if tt.expectExists {
				if pin.Repo != tt.expectRepo {
					t.Errorf("GetActionPinByRepo(%s) repo = %s, want %s", tt.repo, pin.Repo, tt.expectRepo)
				}
				if pin.Version != tt.expectVer {
					t.Errorf("GetActionPinByRepo(%s) version = %s, want %s", tt.repo, pin.Version, tt.expectVer)
				}
				// SHA may be empty if resolution fails, so we don't check it
			}
		})
	}
}

// TestGetDefaultVersion tests the getDefaultVersion function
func TestGetDefaultVersion(t *testing.T) {
	tests := []struct {
		repo     string
		expected string
	}{
		{"actions/checkout", "v5"},
		{"actions/setup-node", "v6"},
		{"actions/cache", "v4"},
		{"unknown/action", ""},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			result := getDefaultVersion(tt.repo)
			if result != tt.expected {
				t.Errorf("getDefaultVersion(%s) = %s, want %s", tt.repo, result, tt.expected)
			}
		})
	}
}
