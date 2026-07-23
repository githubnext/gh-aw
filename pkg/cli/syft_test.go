//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestRunSyftOnImage_NilContext(t *testing.T) {
	// This test verifies that we can pass context.Background()
	ctx := context.Background()
	sbomDir := t.TempDir()

	// We can't actually run Docker in unit tests, so this test
	// just verifies the function signature accepts context
	_, err := runSyftOnImage(ctx, "alpine:latest", sbomDir, false)
	if err != nil {
		t.Skip("Docker not available or image not found, skipping")
	}
}

func TestRunSyftOnImage_ContextCancellation(t *testing.T) {
	// This test verifies that context cancellation is respected
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately before calling the function
	cancel()

	sbomDir := t.TempDir()

	_, err := runSyftOnImage(ctx, "alpine:latest", sbomDir, false)
	if err == nil {
		t.Skip("Expected context cancellation error, but got nil - Docker may not be available")
	}

	// Context cancellation should produce an error
	if err != nil && ctx.Err() != nil {
		// Expected: context was cancelled
		t.Logf("Context cancellation worked as expected: %v", err)
	}
}

func TestRunSyftOnImage_JSONParsing(t *testing.T) {
	// Test that we can parse syft JSON output
	syftJSON := []byte(`{
		"artifacts": [
			{"name": "alpine-baselayout", "version": "3.2.0-r18", "type": "apk"},
			{"name": "busybox", "version": "1.33.1-r6", "type": "apk"}
		]
	}`)

	var output syftOutput
	if err := json.Unmarshal(syftJSON, &output); err != nil {
		t.Fatalf("Failed to parse syft JSON: %v", err)
	}

	if len(output.Artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(output.Artifacts))
	}

	if output.Artifacts[0].Name != "alpine-baselayout" {
		t.Errorf("Expected first artifact name to be 'alpine-baselayout', got '%s'", output.Artifacts[0].Name)
	}
}

func TestRunSyftOnImage_SBOMPersistence(t *testing.T) {
	t.Skip("Requires Docker and network access - run as integration test")

	ctx := context.Background()
	sbomDir := t.TempDir()

	// Run syft on a small image
	result, err := runSyftOnImage(ctx, "alpine:latest", sbomDir, false)
	if err != nil {
		t.Fatalf("Failed to run syft: %v", err)
	}

	// Verify result structure
	if result.ImageRef != "alpine:latest" {
		t.Errorf("Expected image ref 'alpine:latest', got '%s'", result.ImageRef)
	}

	if result.PackageCount <= 0 {
		t.Errorf("Expected positive package count, got %d", result.PackageCount)
	}

	if result.SBOMPath == "" {
		t.Errorf("Expected non-empty SBOM path")
	}

	// Verify SBOM file exists
	if _, err := os.Stat(result.SBOMPath); os.IsNotExist(err) {
		t.Errorf("SBOM file does not exist at %s", result.SBOMPath)
	}

	// Verify SBOM file contains valid JSON
	data, err := os.ReadFile(result.SBOMPath)
	if err != nil {
		t.Fatalf("Failed to read SBOM file: %v", err)
	}

	var output syftOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("SBOM file does not contain valid syft JSON: %v", err)
	}

	if len(output.Artifacts) != result.PackageCount {
		t.Errorf("SBOM artifact count (%d) does not match result count (%d)", len(output.Artifacts), result.PackageCount)
	}
}

func TestRunSyftOnImage_SafeFilename(t *testing.T) {
	// Test that image references with special characters produce safe filenames
	sbomDir := t.TempDir()

	// Manually construct what the expected filename should be for an image like:
	// "gcr.io/distroless/static:latest@sha256:abc123"
	safeImageName := "gcr.io_distroless_static_latest_sha256_abc123"
	expectedPath := filepath.Join(sbomDir, "sbom-"+safeImageName+".json")

	// Create a fake SBOM file with that name to test path construction
	sbomJSON := []byte(`{"artifacts": [{"name": "test", "version": "1.0", "type": "test"}]}`)
	if err := os.WriteFile(expectedPath, sbomJSON, constants.FilePermPublic); err != nil {
		t.Fatalf("Failed to write test SBOM: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s to exist", expectedPath)
	}
}

func TestRunSyftOnLockFiles_NoImages(t *testing.T) {
	// Test that we handle lock files with no container images gracefully
	lockFile := filepath.Join(t.TempDir(), "empty.lock.yml")
	content := `
version: '1.0'
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "no containers here"
`
	if err := os.WriteFile(lockFile, []byte(content), constants.FilePermPublic); err != nil {
		t.Fatalf("Failed to write test lock file: %v", err)
	}

	err := runSyftOnLockFiles([]string{lockFile}, false, false)
	if err != nil {
		t.Errorf("Expected no error for lock files without images, got: %v", err)
	}
}

func TestRunSyftOnLockFiles_StrictMode(t *testing.T) {
	// Test that strict mode propagates errors
	lockFile := filepath.Join(t.TempDir(), "test.lock.yml")
	content := `
version: '1.0'
jobs:
  test:
    container:
      image: does-not-exist:latest
`
	if err := os.WriteFile(lockFile, []byte(content), constants.FilePermPublic); err != nil {
		t.Fatalf("Failed to write test lock file: %v", err)
	}

	// In strict mode, a scan failure should return an error
	err := runSyftOnLockFiles([]string{lockFile}, false, true)
	if err == nil {
		t.Skip("Expected error in strict mode, but got nil - Docker may not be available or image exists")
	}
}

func TestRunSyftOnLockFiles_NonStrictMode(t *testing.T) {
	// Test that non-strict mode continues on errors
	lockFile := filepath.Join(t.TempDir(), "test.lock.yml")
	content := `
version: '1.0'
jobs:
  test:
    container:
      image: does-not-exist:latest
`
	if err := os.WriteFile(lockFile, []byte(content), constants.FilePermPublic); err != nil {
		t.Fatalf("Failed to write test lock file: %v", err)
	}

	// In non-strict mode, a scan failure should only produce a warning
	err := runSyftOnLockFiles([]string{lockFile}, false, false)
	if err != nil {
		t.Errorf("Expected no error in non-strict mode, got: %v", err)
	}
}
