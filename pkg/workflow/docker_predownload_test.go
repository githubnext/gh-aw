package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerImagePredownload(t *testing.T) {
	tests := []struct {
		name           string
		frontmatter    string
		expectedImages []string
		expectStep     bool
	}{
		{
			name: "GitHub tool generates image download step",
			frontmatter: `---
on: issues
engine: claude
tools:
  github:
---

# Test
Test workflow.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.19.0",
			},
			expectStep: true,
		},
		{
			name: "GitHub tool with custom version",
			frontmatter: `---
on: issues
engine: claude
tools:
  github:
    version: v0.17.0
---

# Test
Test workflow.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.17.0",
			},
			expectStep: true,
		},
		{
			name: "Codex with only edit tool still gets GitHub MCP by default",
			frontmatter: `---
on: issues
engine: codex
tools:
  edit:
---

# Test
Test workflow.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.19.0",
			},
			expectStep: true,
		},
		{
			name: "GitHub remote mode does not generate docker image download",
			frontmatter: `---
on: issues
engine: claude
tools:
  github:
    mode: remote
---

# Test
Test workflow.`,
			expectedImages: nil,
			expectStep:     false,
		},
		{
			name: "Custom MCP server with container",
			frontmatter: `---
on: issues
engine: claude
mcp-servers:
  custom-tool:
    container: myorg/custom-mcp:v1.0.0
    allowed: ["*"]
---

# Test
Test workflow with custom MCP container.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.19.0",
				"myorg/custom-mcp:v1.0.0",
			},
			expectStep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "docker-predownload-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Write test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test-version")
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			yaml, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			// Check if the "Downloading container images" step exists
			hasStep := strings.Contains(string(yaml), "Downloading container images")
			if hasStep != tt.expectStep {
				t.Errorf("Expected step existence: %v, got: %v", tt.expectStep, hasStep)
			}

			// If we expect a step, verify the images are present
			if tt.expectStep {
				for _, expectedImage := range tt.expectedImages {
					if !strings.Contains(string(yaml), "docker pull "+expectedImage) {
						t.Errorf("Expected to find 'docker pull %s' in generated YAML", expectedImage)
					}
				}
			}
		})
	}
}

func TestDockerImagePredownloadOrdering(t *testing.T) {
	// Test that the "Downloading container images" step comes before "Setup MCPs"
	frontmatter := `---
on: issues
engine: claude
tools:
  github:
---

# Test
Test workflow.`

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "docker-predownload-ordering-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test workflow file
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test-version")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	yaml, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(yaml)

	// Find the positions of both steps
	downloadPos := strings.Index(yamlStr, "Downloading container images")
	setupPos := strings.Index(yamlStr, "Setup MCPs")

	if downloadPos == -1 {
		t.Fatal("Expected 'Downloading container images' step not found")
	}

	if setupPos == -1 {
		t.Fatal("Expected 'Setup MCPs' step not found")
	}

	// Verify the download step comes before setup step
	if downloadPos > setupPos {
		t.Errorf("Expected 'Downloading container images' to come before 'Setup MCPs', but found it after")
	}
}
