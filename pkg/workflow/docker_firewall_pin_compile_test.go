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

func TestCompileWorkflow_FirewallImagesPinnedForAWF02711(t *testing.T) {
	frontmatter := `---
on: workflow_dispatch
engine: claude
sandbox:
  agent:
    id: awf
    version: v0.27.11
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
		"ghcr.io/github/gh-aw-firewall/agent:0.27.11":     "sha256:979723c628182da7729333f2208bb249fd25ddee579645cf9a3892d681a929c7",
		"ghcr.io/github/gh-aw-firewall/api-proxy:0.27.11": "sha256:807e4831999b44513b0a66e5859d478dc4da7ae74ab1918cec967d513f95bf9d",
		"ghcr.io/github/gh-aw-firewall/squid:0.27.11":     "sha256:ff27ea0525ad953a6adee28a5fbe9d2e22be47dbec755c15767af4ea3f91df7d",
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
		`0.27.11,`,
		`agent=sha256:979723c628182da7729333f2208bb249fd25ddee579645cf9a3892d681a929c7`,
		`api-proxy=sha256:807e4831999b44513b0a66e5859d478dc4da7ae74ab1918cec967d513f95bf9d`,
		`squid=sha256:ff27ea0525ad953a6adee28a5fbe9d2e22be47dbec755c15767af4ea3f91df7d`,
	} {
		if !strings.Contains(yamlStr, imageTagPart) {
			t.Errorf("Expected AWF config JSON to include %s", imageTagPart)
		}
	}
}
