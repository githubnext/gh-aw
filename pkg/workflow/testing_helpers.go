package workflow

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

// CompareGoldenFile compares the generated output with a golden file.
// If the -update-golden flag is set, it updates the golden file with the new output.
// Otherwise, it compares the output with the existing golden file.
func CompareGoldenFile(t *testing.T, got []byte, goldenPath string) {
	t.Helper()

	if *updateGolden {
		// Ensure the directory exists
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create golden file directory: %v", err)
		}

		// Write the golden file
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("failed to write golden file %s: %v", goldenPath, err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read the golden file
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v\nRun with -update-golden to create it", goldenPath, err)
	}

	// Compare the output
	gotStr := string(got)
	wantStr := string(want)

	if gotStr != wantStr {
		t.Errorf("Output does not match golden file %s\n\nTo update golden files, run:\n  go test -update-golden\n\nGot:\n%s\n\nWant:\n%s",
			goldenPath, gotStr, wantStr)
	}
}
