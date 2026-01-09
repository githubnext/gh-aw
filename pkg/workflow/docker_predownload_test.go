package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"

	"github.com/githubnext/gh-aw/pkg/testutil"
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
				"ghcr.io/github/github-mcp-server:v0.27.0",
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
				"ghcr.io/github/github-mcp-server:v0.27.0",
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
strict: false
engine: claude
mcp-servers:
  custom-tool:
    container: myorg/custom-mcp:v1.0.0
    allowed: ["*"]
---

# Test
Test workflow with custom MCP container.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.27.0",
				"myorg/custom-mcp:v1.0.0",
			},
			expectStep: true,
		},
		{
			name: "Safe outputs MCP server includes node:lts-alpine",
			frontmatter: `---
on: issues
engine: claude
safe-outputs:
  create-issue:
    title-prefix: "Test"
---

# Test
Test workflow with safe outputs.`,
			expectedImages: []string{
				"ghcr.io/github/github-mcp-server:v0.27.0",
				"node:lts-alpine",
			},
			expectStep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := testutil.TempDir(t, "docker-predownload-test")

			// Write test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test-version")
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
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
				// Verify the script call is present
				if !strings.Contains(string(yaml), "bash /opt/gh-aw/actions/download_docker_images.sh") {
					t.Error("Expected to find 'bash /opt/gh-aw/actions/download_docker_images.sh' script call in generated YAML")
				}
				for _, expectedImage := range tt.expectedImages {
					// Check that the image is being passed as an argument to the script
					if !strings.Contains(string(yaml), expectedImage) {
						t.Errorf("Expected to find image '%s' in generated YAML", expectedImage)
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
	tmpDir := testutil.TempDir(t, "docker-predownload-ordering-test")

	// Write test workflow file
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test-version")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
