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

// TestCompileWorkflow_FirewallImagesPinnedForDefaultVersion is a regression test for
// gh-aw#43307: the four gh-aw-firewall images at the current default version
// (constants.DefaultFirewallVersion) must all be digest-pinned in consumer lock files
// even when no local action-cache is present.  This covers the cli-proxy image
// introduced in v0.82 as well as the three legacy images (agent, api-proxy, squid).
func TestCompileWorkflow_FirewallImagesPinnedForDefaultVersion(t *testing.T) {
	// Strip the leading "v" to get the Docker image tag (mirrors getAWFImageTag).
	imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")

	// Enable tools.github.mode=gh-proxy so that the cli-proxy sidecar container is
	// included in the Docker pull list and therefore also pinned in the lock file.
	frontmatter := `---
on: workflow_dispatch
engine: claude
network:
  allowed:
    - defaults
tools:
  github:
    mode: gh-proxy
---

# Test
Test workflow.`

	tmpDir := testutil.TempDir(t, "docker-firewall-pins-default-version-test")
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
		"ghcr.io/github/gh-aw-firewall/agent:" + imageTag:     "sha256:55f06588411008b7148eb64b8dfe28602a0cce3675b36c6b190b54aca138468e",
		"ghcr.io/github/gh-aw-firewall/api-proxy:" + imageTag: "sha256:afb9ff9140b17d38871dfb9dbac5ff8689ea634c2f91c435da2825192d4881c1",
		"ghcr.io/github/gh-aw-firewall/cli-proxy:" + imageTag: "sha256:e23e1604241f579b418e6522d938285b57ada31bc27742a65c90ee2250b1755c",
		"ghcr.io/github/gh-aw-firewall/squid:" + imageTag:     "sha256:3cdcc1e2b4b4fe602ba69fd3e21aac7ac512d5c1fce24df4ce69dc4f98164b59",
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
		imageTag + `,`,
		`agent=sha256:55f06588411008b7148eb64b8dfe28602a0cce3675b36c6b190b54aca138468e`,
		`api-proxy=sha256:afb9ff9140b17d38871dfb9dbac5ff8689ea634c2f91c435da2825192d4881c1`,
		`cli-proxy=sha256:e23e1604241f579b418e6522d938285b57ada31bc27742a65c90ee2250b1755c`,
		`squid=sha256:3cdcc1e2b4b4fe602ba69fd3e21aac7ac512d5c1fce24df4ce69dc4f98164b59`,
	} {
		if !strings.Contains(yamlStr, imageTagPart) {
			t.Errorf("Expected AWF config JSON to include %s", imageTagPart)
		}
	}
}
