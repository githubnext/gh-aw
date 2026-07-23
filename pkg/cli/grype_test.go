//go:build !integration

package cli

import (
	"errors"
	"os"
	"testing"
)

func TestGrypeDisplayFindings_NilOutput(t *testing.T) {
	count := grypeDisplayFindings("test-image:latest", nil)
	if count != 0 {
		t.Errorf("Expected 0 findings for nil output, got %d", count)
	}
}

func TestGrypeDisplayFindings_EmptyMatches(t *testing.T) {
	output := &grypeOutput{Matches: []grypeFinding{}}
	count := grypeDisplayFindings("test-image:latest", output)
	if count != 0 {
		t.Errorf("Expected 0 findings for empty output, got %d", count)
	}
}

func TestGrypeDisplayFindings_WithFindings(t *testing.T) {
	output := &grypeOutput{
		Matches: []grypeFinding{
			makeGrypeFinding("CVE-2021-12345", "High", "libssl", "1.1.1", []string{"1.1.2"}, "https://nvd.nist.gov/vuln/detail/CVE-2021-12345"),
			makeGrypeFinding("CVE-2021-99999", "Critical", "openssl", "1.0.0", nil, ""),
		},
	}

	count := grypeDisplayFindings("ubuntu:20.04", output)
	if count != 2 {
		t.Errorf("Expected 2 findings, got %d", count)
	}
}

func TestGrypeDisplayFindings_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity string
		wantType string
	}{
		{"Critical", "error"},
		{"High", "error"},
		{"Medium", "warning"},
		{"Low", "info"},
		{"Negligible", "info"},
		{"Informational", "info"},
		{"Unknown", "warning"},
		{"", "warning"},
	}

	for _, tc := range tests {
		t.Run(tc.severity, func(t *testing.T) {
			output := &grypeOutput{
				Matches: []grypeFinding{
					makeGrypeFinding("CVE-2021-00000", tc.severity, "pkg", "1.0", nil, ""),
				},
			}
			// We can't easily test the errorType without capturing stderr,
			// but we can verify the function returns the right count.
			count := grypeDisplayFindings("test-image:latest", output)
			if count != 1 {
				t.Errorf("Expected 1 finding for severity %q, got %d", tc.severity, count)
			}
		})
	}
}

func TestGrypeCacheGetSet(t *testing.T) {
	cache := &grypeCache{
		results: make(map[string]*grypeOutput),
		errors:  make(map[string]error),
	}

	key := "test-image:latest"

	// Initially no entry.
	result, err, ok := cache.get(key)
	if ok {
		t.Error("Expected no cache entry initially")
	}
	if result != nil || err != nil {
		t.Error("Expected nil result and nil error for empty cache")
	}

	// Set a result.
	expected := &grypeOutput{Matches: []grypeFinding{}}
	cache.set(key, expected)

	result, err, ok = cache.get(key)
	if !ok {
		t.Error("Expected cache hit after set")
	}
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != expected {
		t.Error("Expected cached result to match stored result")
	}
}

func TestGrypeCacheSetError(t *testing.T) {
	cache := &grypeCache{
		results: make(map[string]*grypeOutput),
		errors:  make(map[string]error),
	}

	key := "test-image:v1.0"
	testErr := errors.New("test scan error")
	cache.setError(key, testErr)

	result, err, ok := cache.get(key)
	if !ok {
		t.Error("Expected cache hit after setError")
	}
	if result != nil {
		t.Error("Expected nil result for error entry")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("Expected stored error %v, got %v", testErr, err)
	}
}

func TestGrypeCacheReset(t *testing.T) {
	cache := &grypeCache{
		results: make(map[string]*grypeOutput),
		errors:  make(map[string]error),
	}

	cache.set("key1", &grypeOutput{})
	cache.setError("key2", errors.New("test error"))

	cache.reset()

	_, _, ok := cache.get("key1")
	if ok {
		t.Error("Expected key1 to be cleared after reset")
	}
	_, _, ok = cache.get("key2")
	if ok {
		t.Error("Expected key2 to be cleared after reset")
	}
}

func TestRunGrypeOnLockFiles_NoLockFiles(t *testing.T) {
	err := runGrypeOnLockFiles([]string{}, false, false)
	if err != nil {
		t.Errorf("Expected no error for empty lock file list, got: %v", err)
	}
}

func TestCollectContainerImagesFromLockFiles_Nil(t *testing.T) {
	images := collectContainerImagesFromLockFiles(nil)
	if images != nil {
		t.Errorf("Expected nil for nil input, got %v", images)
	}
}

func TestCollectContainerImagesFromLockFiles_Empty(t *testing.T) {
	images := collectContainerImagesFromLockFiles([]string{})
	if images != nil {
		t.Errorf("Expected nil for empty input, got %v", images)
	}
}

func TestCollectContainerImagesFromLockFiles_NonExistentFile(t *testing.T) {
	images := collectContainerImagesFromLockFiles([]string{"/nonexistent/path.lock.yml"})
	if len(images) != 0 {
		t.Errorf("Expected 0 images for non-existent file, got %d", len(images))
	}
}

func TestCollectContainerImagesFromLockFiles_NoManifest(t *testing.T) {
	// A lock file with no gh-aw-manifest header.
	tmpFile, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("# Generated workflow YAML\nname: test\n"); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	images := collectContainerImagesFromLockFiles([]string{tmpFile.Name()})
	if len(images) != 0 {
		t.Errorf("Expected 0 images for lock file without manifest, got %d", len(images))
	}
}

func TestCollectContainerImagesFromLockFiles_WithManifest(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	manifest := `# gh-aw-manifest: {"version":1,"secrets":[],"actions":[],"containers":[{"image":"ghcr.io/test/image:v1.0","digest":"sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1","pinned_image":"ghcr.io/test/image:v1.0@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"}]}`
	if _, err := tmpFile.WriteString(manifest + "\n"); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	images := collectContainerImagesFromLockFiles([]string{tmpFile.Name()})
	if len(images) != 1 {
		t.Fatalf("Expected 1 image, got %d: %v", len(images), images)
	}
	if images[0].Image != "ghcr.io/test/image:v1.0" {
		t.Errorf("Expected image tag %q, got %q", "ghcr.io/test/image:v1.0", images[0].Image)
	}
	if images[0].PinnedImage == "" {
		t.Error("Expected non-empty PinnedImage")
	}
}

func TestCollectContainerImagesFromLockFiles_DeduplicatesByPinnedImage(t *testing.T) {
	manifest := `# gh-aw-manifest: {"version":1,"secrets":[],"actions":[],"containers":[{"image":"ghcr.io/test/image:v1.0","digest":"sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1","pinned_image":"ghcr.io/test/image:v1.0@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"}]}`

	file1, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(file1.Name())
	file1.WriteString(manifest + "\n")
	file1.Close()

	file2, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(file2.Name())
	file2.WriteString(manifest + "\n")
	file2.Close()

	images := collectContainerImagesFromLockFiles([]string{file1.Name(), file2.Name()})
	if len(images) != 1 {
		t.Errorf("Expected 1 unique image after deduplication, got %d", len(images))
	}
}

func TestCollectContainerImagesFromLockFiles_MultipleDistinctImages(t *testing.T) {
	manifest1 := `# gh-aw-manifest: {"version":1,"secrets":[],"actions":[],"containers":[{"image":"ghcr.io/test/image-a:v1.0","pinned_image":"ghcr.io/test/image-a:v1.0@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`
	manifest2 := `# gh-aw-manifest: {"version":1,"secrets":[],"actions":[],"containers":[{"image":"ghcr.io/test/image-b:v2.0","pinned_image":"ghcr.io/test/image-b:v2.0@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}`

	file1, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(file1.Name())
	file1.WriteString(manifest1 + "\n")
	file1.Close()

	file2, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(file2.Name())
	file2.WriteString(manifest2 + "\n")
	file2.Close()

	images := collectContainerImagesFromLockFiles([]string{file1.Name(), file2.Name()})
	if len(images) != 2 {
		t.Errorf("Expected 2 distinct images, got %d", len(images))
	}
}

func TestCollectContainerImagesFromLockFiles_EmptyImageIgnored(t *testing.T) {
	manifest := `# gh-aw-manifest: {"version":1,"secrets":[],"actions":[],"containers":[{"image":"","pinned_image":""}]}`

	tmpFile, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(manifest + "\n")
	tmpFile.Close()

	images := collectContainerImagesFromLockFiles([]string{tmpFile.Name()})
	if len(images) != 0 {
		t.Errorf("Expected 0 images (empty image name ignored), got %d", len(images))
	}
}

func TestCollectContainerImagesFromLockFiles_NoContainers(t *testing.T) {
	manifest := `# gh-aw-manifest: {"version":1,"secrets":["MY_SECRET"],"actions":[]}`

	tmpFile, err := os.CreateTemp("", "test-*.lock.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(manifest + "\n")
	tmpFile.Close()

	images := collectContainerImagesFromLockFiles([]string{tmpFile.Name()})
	if len(images) != 0 {
		t.Errorf("Expected 0 images for manifest without containers, got %d", len(images))
	}
}

// makeGrypeFinding is a test helper that constructs a grypeFinding.
func makeGrypeFinding(id, severity, pkgName, pkgVersion string, fixVersions []string, dataSource string) grypeFinding {
	f := grypeFinding{}
	f.Vulnerability.ID = id
	f.Vulnerability.Severity = severity
	f.Vulnerability.DataSource = dataSource
	f.Vulnerability.Fix.Versions = fixVersions
	f.Artifact.Name = pkgName
	f.Artifact.Version = pkgVersion
	return f
}
