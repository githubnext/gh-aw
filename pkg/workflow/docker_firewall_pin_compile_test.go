//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	expectedPins := map[string]string{
		"ghcr.io/github/gh-aw-firewall/agent:0.27.0":     "sha256:3816d1692e6d96887b27f1e4f1d64b8d7edb43ed9d7506b8f203913cbb81c248",
		"ghcr.io/github/gh-aw-firewall/api-proxy:0.27.0": "sha256:f28d2bd3197fb6ef9ec40ef345bbf2bb33e50151a8e72e89abb618fc3d0066eb",
		"ghcr.io/github/gh-aw-firewall/squid:0.27.0":     "sha256:d6a01d4cf3d928e6a7fc42e34afef228e753dce87646edc91d8a5cd0b612d9a6",
	}

	for image, digest := range expectedPins {
		pinnedImage := image + "@" + digest
		if !strings.Contains(yamlStr, `"image":"`+image+`","digest":"`+digest+`","pinned_image":"`+pinnedImage+`"`) {
			t.Errorf("Expected manifest header to include pinned metadata for %s", image)
		}
		if !strings.Contains(yamlStr, "#   - "+pinnedImage) {
			t.Errorf("Expected pinned container comment for %s", image)
		}
		if !strings.Contains(yamlStr, pinnedImage) {
			t.Errorf("Expected pinned download reference for %s", image)
		}
	}

	for _, imageTagPart := range []string{
		`imageTag`,
		`0.27.0,`,
		`agent=sha256:3816d1692e6d96887b27f1e4f1d64b8d7edb43ed9d7506b8f203913cbb81c248`,
		`api-proxy=sha256:f28d2bd3197fb6ef9ec40ef345bbf2bb33e50151a8e72e89abb618fc3d0066eb`,
		`squid=sha256:d6a01d4cf3d928e6a7fc42e34afef228e753dce87646edc91d8a5cd0b612d9a6`,
	} {
		if !strings.Contains(yamlStr, imageTagPart) {
			t.Errorf("Expected AWF config JSON to include %s", imageTagPart)
		}
	}
}
