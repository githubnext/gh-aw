package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestCollectUsedActionPins tests the collectUsedActionPins function
func TestCollectUsedActionPins(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected map[string]string // map of repo to version
	}{
		{
			name: "single action",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
`,
			expected: map[string]string{
				"actions/checkout": "v5",
			},
		},
		{
			name: "multiple different actions",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
      - uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
      - uses: actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830
`,
			expected: map[string]string{
				"actions/checkout":   "v5",
				"actions/setup-node": "v4",
				"actions/cache":      "v4",
			},
		},
		{
			name: "duplicate actions",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
      - uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
`,
			expected: map[string]string{
				"actions/checkout":   "v5",
				"actions/setup-node": "v4",
			},
		},
		{
			name: "no pinned actions",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Hello"
`,
			expected: map[string]string{},
		},
		{
			name: "unpinned third-party action",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: some-org/some-action@v1
`,
			expected: map[string]string{},
		},
		{
			name: "nested action path",
			yaml: `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: github/codeql-action/upload-sarif@562257dc84ee23987d348302b161ee561898ec02
`,
			expected: map[string]string{
				"github/codeql-action/upload-sarif": "v3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectUsedActionPins(tt.yaml)

			// Check that we got the expected number of pins
			if len(result) != len(tt.expected) {
				t.Errorf("collectUsedActionPins() returned %d pins, expected %d", len(result), len(tt.expected))
				t.Logf("Got: %v", result)
				t.Logf("Expected repos: %v", tt.expected)
			}

			// Check each expected pin
			for repo, expectedVersion := range tt.expected {
				pin, exists := result[repo]
				if !exists {
					t.Errorf("Expected pin for %s not found in result", repo)
					continue
				}

				if pin.Version != expectedVersion {
					t.Errorf("Pin for %s has version %s, expected %s", repo, pin.Version, expectedVersion)
				}
			}
		})
	}
}

// TestGeneratePinnedActionsComment tests the generatePinnedActionsComment function
func TestGeneratePinnedActionsComment(t *testing.T) {
	tests := []struct {
		name             string
		usedPins         map[string]ActionPin
		expectedContains []string
		shouldBeEmpty    bool
	}{
		{
			name:          "empty pins",
			usedPins:      map[string]ActionPin{},
			shouldBeEmpty: true,
		},
		{
			name: "single pin",
			usedPins: map[string]ActionPin{
				"actions/checkout": {
					Repo:    "actions/checkout",
					Version: "v5",
					SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
				},
			},
			expectedContains: []string{
				"# Pinned GitHub Actions:",
				"#   - actions/checkout@v5 (08c6903cd8c0fde910a37f88322edcfb5dd907a8)",
				"#     https://github.com/actions/checkout/commit/08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			},
		},
		{
			name: "multiple pins sorted alphabetically",
			usedPins: map[string]ActionPin{
				"actions/setup-node": {
					Repo:    "actions/setup-node",
					Version: "v4",
					SHA:     "49933ea5288caeca8642d1e84afbd3f7d6820020",
				},
				"actions/checkout": {
					Repo:    "actions/checkout",
					Version: "v5",
					SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
				},
				"actions/cache": {
					Repo:    "actions/cache",
					Version: "v4",
					SHA:     "0057852bfaa89a56745cba8c7296529d2fc39830",
				},
			},
			expectedContains: []string{
				"# Pinned GitHub Actions:",
				"#   - actions/cache@v4",
				"#   - actions/checkout@v5",
				"#   - actions/setup-node@v4",
				"#     https://github.com/actions/cache/commit/",
				"#     https://github.com/actions/checkout/commit/",
				"#     https://github.com/actions/setup-node/commit/",
			},
		},
		{
			name: "nested action path",
			usedPins: map[string]ActionPin{
				"github/codeql-action/upload-sarif": {
					Repo:    "github/codeql-action/upload-sarif",
					Version: "v3",
					SHA:     "562257dc84ee23987d348302b161ee561898ec02",
				},
			},
			expectedContains: []string{
				"# Pinned GitHub Actions:",
				"#   - github/codeql-action/upload-sarif@v3",
				"#     https://github.com/github/codeql-action/commit/562257dc84ee23987d348302b161ee561898ec02",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePinnedActionsComment(tt.usedPins)

			if tt.shouldBeEmpty {
				if result != "" {
					t.Errorf("Expected empty comment, got:\n%s", result)
				}
				return
			}

			// Check for expected content
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Comment does not contain expected string:\n  Expected: %s\n  Got:\n%s", expected, result)
				}
			}

			// Verify comment starts with proper formatting
			if !strings.HasPrefix(result, "#\n# Pinned GitHub Actions:\n") {
				t.Errorf("Comment does not start with expected header:\n%s", result)
			}
		})
	}
}

// TestPinnedActionsCommentInGeneratedYAML tests that the pinned actions comment appears in generated YAML
func TestPinnedActionsCommentInGeneratedYAML(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "pinned-actions-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple workflow
	workflow := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
---

# Test Workflow

This is a test workflow.
`

	// Write the workflow to a file
	testFile := tmpDir + "/test.md"
	if err := os.WriteFile(testFile, []byte(workflow), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetNoEmit(true) // Don't write lock files

	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated YAML (it should be in memory since we used SetNoEmit)
	// Actually, SetNoEmit prevents writing, so we need to compile without it to read the file
	compiler2 := NewCompiler(false, "", "test-version")
	if err := compiler2.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFile := tmpDir + "/test.lock.yml"
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(yamlContent)

	// Check that the pinned actions comment appears
	expectedStrings := []string{
		"# Pinned GitHub Actions:",
		"actions/checkout@v5",
		"https://github.com/actions/checkout/commit/",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
			// Print first 2000 chars of YAML for debugging
			if len(yamlStr) > 2000 {
				t.Logf("YAML preview:\n%s\n...(truncated)", yamlStr[:2000])
			} else {
				t.Logf("YAML:\n%s", yamlStr)
			}
		}
	}

	// Verify the comment appears before the "name:" section
	pinnedActionsPos := strings.Index(yamlStr, "# Pinned GitHub Actions:")
	namePos := strings.Index(yamlStr, "name:")

	if pinnedActionsPos == -1 {
		t.Error("Pinned actions comment not found in generated YAML")
	} else if namePos == -1 {
		t.Error("name: field not found in generated YAML")
	} else if pinnedActionsPos > namePos {
		t.Error("Pinned actions comment appears after the name: field, should be before")
	}
}
