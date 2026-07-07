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
		"ghcr.io/github/gh-aw-firewall/agent:" + imageTag:     "sha256:b283b1f037d2e068532fe178a06f2944696c3933ba11604979134e7896ac6f8c",
		"ghcr.io/github/gh-aw-firewall/api-proxy:" + imageTag: "sha256:1743dbac9bf4225f3acfdcbce4f77f5a3e61e22b2e929305525f00196693c015",
		"ghcr.io/github/gh-aw-firewall/cli-proxy:" + imageTag: "sha256:6006b1c579ca550e023697b5dfd832ae03361328a4c0f2eb49fb181a6b8d7a4b",
		"ghcr.io/github/gh-aw-firewall/squid:" + imageTag:     "sha256:d4647c3bfaf80889eb1dbd3d4e2063340cbf94c0ca6c5747bbfc8507b12f3485",
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
		`agent=sha256:b283b1f037d2e068532fe178a06f2944696c3933ba11604979134e7896ac6f8c`,
		`api-proxy=sha256:1743dbac9bf4225f3acfdcbce4f77f5a3e61e22b2e929305525f00196693c015`,
		`cli-proxy=sha256:6006b1c579ca550e023697b5dfd832ae03361328a4c0f2eb49fb181a6b8d7a4b`,
		`squid=sha256:d4647c3bfaf80889eb1dbd3d4e2063340cbf94c0ca6c5747bbfc8507b12f3485`,
	} {
		if !strings.Contains(yamlStr, imageTagPart) {
			t.Errorf("Expected AWF config JSON to include %s", imageTagPart)
		}
	}
}

// TestCompileWorkflow_BuildToolsImagePinnedForArcDind is a regression test for
// gh-aw#44040: when runner.topology is arc-dind, the build-tools image must be
// digest-pinned in the compiled lock file the same way the other four gh-aw-firewall
// images (agent, api-proxy, cli-proxy, squid) are.
func TestCompileWorkflow_BuildToolsImagePinnedForArcDind(t *testing.T) {
	// Strip the leading "v" to get the Docker image tag (mirrors getAWFImageTag).
	imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")

	frontmatter := `---
on: workflow_dispatch
engine: claude
runner:
  topology: arc-dind
network:
  allowed:
    - defaults
---

# Test
Test workflow.`

	tmpDir := testutil.TempDir(t, "docker-firewall-pins-arc-dind-test")
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

	buildToolsImage := "ghcr.io/github/gh-aw-firewall/build-tools:" + imageTag
	buildToolsDigest := "sha256:5823f6cec65210cd6e3e6320c165979fb46f78c76600f58f4189d2c89c2be8da"
	pinnedBuildTools := buildToolsImage + "@" + buildToolsDigest

	if !strings.Contains(yamlStr, `"image":"`+buildToolsImage+`","digest":"`+buildToolsDigest+`","pinned_image":"`+pinnedBuildTools+`"`) {
		t.Errorf("Expected manifest header to include pinned metadata for %s", buildToolsImage)
	}
	if !strings.Contains(yamlStr, "#   - "+pinnedBuildTools) {
		t.Errorf("Expected pinned container comment for %s", buildToolsImage)
	}
	if !strings.Contains(yamlStr, pinnedBuildTools) {
		t.Errorf("Expected pinned download reference for %s", buildToolsImage)
	}

	if !strings.Contains(yamlStr, `build-tools=`+buildToolsDigest) {
		t.Errorf("Expected AWF config JSON to include build-tools=%s", buildToolsDigest)
	}
}
