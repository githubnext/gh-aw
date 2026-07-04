//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestCompileWorkflow_FirewallImagesPinnedForAWF0270(t *testing.T) {
	frontmatter := `---
on: workflow_dispatch
engine: claude
sandbox:
  agent:
    id: awf
    version: v0.27.0
network:
  allowed:
    - defaults
tools:
  web-fetch:
---

# Test
Test workflow.`

	tmpDir := testutil.TempDir(t, "docker-firewall-pins-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(testFile)
	yaml, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(yaml)
	requireEmbeddedPin := func(image string) ContainerPin {
		t.Helper()
		pin, ok := getEmbeddedContainerPin(image)
		if !ok {
			t.Fatalf("Expected embedded pin for %s", image)
		}
		return pin
	}

	expectedPins := []struct {
		name  string
		image string
	}{
		{name: "agent", image: constants.DefaultFirewallRegistry + "/agent:0.27.0"},
		{name: "api-proxy", image: constants.DefaultFirewallRegistry + "/api-proxy:0.27.0"},
		{name: "squid", image: constants.DefaultFirewallRegistry + "/squid:0.27.0"},
	}

	for _, expectedPin := range expectedPins {
		pin := requireEmbeddedPin(expectedPin.image)

		if !strings.Contains(yamlStr, `"image":"`+pin.Image+`","digest":"`+pin.Digest+`","pinned_image":"`+pin.PinnedImage+`"`) {
			t.Errorf("Expected manifest header to include pinned metadata for %s", expectedPin.image)
		}
		if !strings.Contains(yamlStr, "#   - "+pin.PinnedImage) {
			t.Errorf("Expected pinned container comment for %s", expectedPin.image)
		}
		if !strings.Contains(yamlStr, pin.PinnedImage) {
			t.Errorf("Expected pinned download reference for %s", expectedPin.image)
		}
	}

	imageTagParts := []string{
		`imageTag`,
		`0.27.0,`,
	}
	for _, expectedPin := range expectedPins {
		pin := requireEmbeddedPin(expectedPin.image)
		imageTagParts = append(imageTagParts, expectedPin.name+"="+pin.Digest)
	}

	for _, imageTagPart := range imageTagParts {
		if !strings.Contains(yamlStr, imageTagPart) {
			t.Errorf("Expected AWF config JSON to include %s", imageTagPart)
		}
	}
}
