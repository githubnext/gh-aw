package workflow

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestActionPinsExist verifies that all action pinning entries exist
func TestActionPinsExist(t *testing.T) {
	expectedActions := []string{
		"actions/checkout",
		"actions/github-script",
		"actions/upload-artifact",
		"actions/download-artifact",
		"actions/cache",
		"actions/setup-node",
		"actions/setup-python",
		"actions/setup-go",
		"actions/setup-java",
		"actions/setup-dotnet",
		"erlef/setup-beam",
		"haskell-actions/setup",
		"ruby/setup-ruby",
		"astral-sh/setup-uv",
	}

	actionPins := getActionPins()
	for _, action := range expectedActions {
		var pin ActionPin
		found := false
		for _, p := range actionPins {
			if p.Repo == action {
				pin = p
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing action pin for %s", action)
			continue
		}

		// Verify the pin has a valid SHA (40 character hex string)
		if !isValidSHA(pin.SHA) {
			t.Errorf("Invalid SHA for %s: %s (expected 40-character hex string)", action, pin.SHA)
		}

		// Verify the pin has a version
		if pin.Version == "" {
			t.Errorf("Missing version for %s", action)
		}
	}
}

// TestGetActionPinReturnsValidSHA tests that GetActionPin returns valid SHA references
func TestGetActionPinReturnsValidSHA(t *testing.T) {
	tests := []struct {
		repo    string
		wantSHA bool
	}{
		{"actions/checkout", true},
		{"actions/github-script", true},
		{"actions/upload-artifact", true},
		{"actions/download-artifact", true},
		{"actions/cache", true},
		{"actions/setup-node", true},
		{"actions/setup-python", true},
		{"actions/setup-go", true},
		{"actions/setup-java", true},
		{"actions/setup-dotnet", true},
		{"erlef/setup-beam", true},
		{"haskell-actions/setup", true},
		{"ruby/setup-ruby", true},
		{"astral-sh/setup-uv", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			result := GetActionPin(tt.repo)

			// Check that the result contains a SHA (40-char hex after @)
			parts := strings.Split(result, "@")
			if len(parts) != 2 {
				t.Errorf("GetActionPin(%s) = %s, expected format repo@sha", tt.repo, result)
				return
			}

			if tt.wantSHA {
				if !isValidSHA(parts[1]) {
					t.Errorf("GetActionPin(%s) = %s, expected SHA to be 40-char hex", tt.repo, result)
				}
			}
		})
	}
}

// TestGetActionPinFallback tests that GetActionPin returns empty string for unknown actions
func TestGetActionPinFallback(t *testing.T) {
	result := GetActionPin("unknown/action")
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

// TestActionPinSHAsMatchVersionTags verifies that the SHAs in actionPins actually correspond to their version tags
// by querying the GitHub repositories using git ls-remote
func TestActionPinSHAsMatchVersionTags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	actionPins := getActionPins()
	// Test all action pins in parallel for faster execution
	for _, pin := range actionPins {
		pin := pin // Capture for parallel execution
		t.Run(pin.Repo, func(t *testing.T) {
			t.Parallel() // Run subtests in parallel

			// Extract the repository URL from the repo field
			// For actions like "actions/checkout", the URL is https://github.com/actions/checkout.git
			// For actions like "github/codeql-action/upload-sarif", we need the base repo
			repoURL := getGitHubRepoURL(pin.Repo)

			// Use git ls-remote to get the SHA for the version tag
			cmd := exec.Command("git", "ls-remote", repoURL, "refs/tags/"+pin.Version)
			output, err := cmd.Output()
			if err != nil {
				t.Logf("Warning: Could not verify %s@%s - git ls-remote failed: %v", pin.Repo, pin.Version, err)
				t.Logf("This may be expected for actions that don't follow standard tagging or private repos")
				return // Skip verification but don't fail the test
			}

			outputStr := strings.TrimSpace(string(output))
			if outputStr == "" {
				t.Logf("Warning: No tag found for %s@%s", pin.Repo, pin.Version)
				return // Skip verification but don't fail the test
			}

			// Extract SHA from git ls-remote output (format: "SHA\trefs/tags/version")
			parts := strings.Fields(outputStr)
			if len(parts) < 1 {
				t.Errorf("Unexpected git ls-remote output format for %s@%s: %s", pin.Repo, pin.Version, outputStr)
				return
			}

			actualSHA := parts[0]

			// Verify the SHA matches
			if actualSHA != pin.SHA {
				t.Errorf("SHA mismatch for %s@%s:\n  Expected: %s\n  Got:      %s",
					pin.Repo, pin.Version, pin.SHA, actualSHA)
				t.Logf("To fix, update the SHA in action_pins.go to: %s", actualSHA)
			}
		})
	}
}

// getGitHubRepoURL converts a repo path to a GitHub URL
// For "actions/checkout" -> "https://github.com/actions/checkout.git"
// For "github/codeql-action/upload-sarif" -> "https://github.com/github/codeql-action.git"
func getGitHubRepoURL(repo string) string {
	// For actions with subpaths (like codeql-action/upload-sarif), extract the base repo
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		// Take first two parts (owner/repo)
		baseRepo := parts[0] + "/" + parts[1]
		return "https://github.com/" + baseRepo + ".git"
	}
	return "https://github.com/" + repo + ".git"
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
	tests := []struct {
		name         string
		stepMap      map[string]any
		expectPinned bool
		expectedUses string
	}{
		{
			name: "step with pinned action (checkout)",
			stepMap: map[string]any{
				"name": "Checkout code",
				"uses": "actions/checkout@v5",
			},
			expectPinned: true,
			expectedUses: "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
		},
		{
			name: "step with pinned action (setup-node)",
			stepMap: map[string]any{
				"name": "Setup Node",
				"uses": "actions/setup-node@v4",
				"with": map[string]any{
					"node-version": "20",
				},
			},
			expectPinned: true,
			expectedUses: "actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020",
		},
		{
			name: "step with unpinned action",
			stepMap: map[string]any{
				"name": "Custom action",
				"uses": "my-org/my-action@v1",
			},
			expectPinned: false,
			expectedUses: "my-org/my-action@v1",
		},
		{
			name: "step without uses field",
			stepMap: map[string]any{
				"name": "Run command",
				"run":  "echo hello",
			},
			expectPinned: false,
			expectedUses: "",
		},
		{
			name: "step with already pinned SHA",
			stepMap: map[string]any{
				"name": "Checkout",
				"uses": "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			},
			expectPinned: true,
			expectedUses: "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal WorkflowData for testing with ActionPinManager
			tmpDir := t.TempDir()
			data := &WorkflowData{
				ActionPinManager: NewActionPinManager(tmpDir),
			}
			// Load builtin pins
			if err := data.ActionPinManager.LoadBuiltinPins(); err != nil {
				t.Fatalf("Failed to load builtin pins: %v", err)
			}
			if err := data.ActionPinManager.MergePins(); err != nil {
				t.Fatalf("Failed to merge pins: %v", err)
			}

			result := ApplyActionPinToStep(tt.stepMap, data)

			// Check if uses field exists in result
			if uses, hasUses := result["uses"]; hasUses {
				usesStr, ok := uses.(string)
				if !ok {
					t.Errorf("ApplyActionPinToStep returned non-string uses field")
					return
				}

				if usesStr != tt.expectedUses {
					t.Errorf("ApplyActionPinToStep uses = %q, want %q", usesStr, tt.expectedUses)
				}

				// Verify other fields are preserved (check length and keys)
				if len(result) != len(tt.stepMap) {
					t.Errorf("ApplyActionPinToStep changed number of fields: got %d, want %d", len(result), len(tt.stepMap))
				}
				for k := range tt.stepMap {
					if _, exists := result[k]; !exists {
						t.Errorf("ApplyActionPinToStep lost field %q", k)
					}
				}
			} else if tt.expectedUses != "" {
				t.Errorf("ApplyActionPinToStep removed uses field when it should be %q", tt.expectedUses)
			}
		})
	}
}

// TestGetActionPinsSorting tests that getActionPins returns sorted action pins
func TestGetActionPinsSorting(t *testing.T) {
	pins := getActionPins()

	// Verify we got all the pins (should be 16)
	if len(pins) != 16 {
		t.Errorf("getActionPins() returned %d pins, expected 16", len(pins))
	}

	// Verify they are sorted by version (descending) then by repository name (ascending)
	for i := 0; i < len(pins)-1; i++ {
		if pins[i].Version < pins[i+1].Version {
			t.Errorf("Pins not sorted correctly by version: %s (v%s) should come before %s (v%s)",
				pins[i].Repo, pins[i].Version, pins[i+1].Repo, pins[i+1].Version)
		} else if pins[i].Version == pins[i+1].Version && pins[i].Repo > pins[i+1].Repo {
			t.Errorf("Pins not sorted correctly by repo name within same version: %s should come before %s",
				pins[i].Repo, pins[i+1].Repo)
		}
	}

	// Verify all pins have the required fields
	for _, pin := range pins {
		if pin.Repo == "" {
			t.Error("Found pin with empty Repo field")
		}
		if pin.Version == "" {
			t.Errorf("Pin %s has empty Version field", pin.Repo)
		}
		if !isValidSHA(pin.SHA) {
			t.Errorf("Pin %s has invalid SHA: %s", pin.Repo, pin.SHA)
		}
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
			expectVer:    "v4",
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
			pin, exists := GetActionPinByRepo(tt.repo)

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
				if !isValidSHA(pin.SHA) {
					t.Errorf("GetActionPinByRepo(%s) has invalid SHA: %s", tt.repo, pin.SHA)
				}
			}
		})
	}
}
