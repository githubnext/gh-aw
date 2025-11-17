package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestExtractActionsFromLockFile(t *testing.T) {
	// Create a temporary lock file with test content
	tmpDir := testutil.TempDir(t, "test-*")
	lockFile := filepath.Join(tmpDir, "test.lock.yml")

	lockContent := `
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
      - uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
      - name: Run tests
        run: npm test
      - uses: github/codeql-action/upload-sarif@ab2e54f42aa112ff08704159b88a57517f6f0ebb
`

	if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	// Extract actions
	actions, err := ExtractActionsFromLockFile(lockFile)
	if err != nil {
		t.Fatalf("ExtractActionsFromLockFile failed: %v", err)
	}

	// Verify we extracted the expected actions
	if len(actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(actions))
	}

	// Check that we have the expected repositories
	expectedRepos := map[string]bool{
		"actions/checkout":                  false,
		"actions/setup-node":                false,
		"github/codeql-action/upload-sarif": false,
	}

	for _, action := range actions {
		if _, exists := expectedRepos[action.Repo]; exists {
			expectedRepos[action.Repo] = true
		}
	}

	for repo, found := range expectedRepos {
		if !found {
			t.Errorf("Expected to find action %s, but it was not extracted", repo)
		}
	}

	// Verify SHA format
	for _, action := range actions {
		if len(action.SHA) != 40 {
			t.Errorf("Expected SHA to be 40 characters, got %d for %s", len(action.SHA), action.Repo)
		}
	}
}

func TestExtractActionsFromLockFileNoDuplicates(t *testing.T) {
	// Create a temporary lock file with duplicate actions
	tmpDir := testutil.TempDir(t, "test-*")
	lockFile := filepath.Join(tmpDir, "test.lock.yml")

	lockContent := `
name: Test Workflow
on: push
jobs:
  test1:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
      - uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
  test2:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
      - uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
`

	if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	// Extract actions
	actions, err := ExtractActionsFromLockFile(lockFile)
	if err != nil {
		t.Fatalf("ExtractActionsFromLockFile failed: %v", err)
	}

	// Verify we only have 2 unique actions despite being used twice
	if len(actions) != 2 {
		t.Errorf("Expected 2 unique actions, got %d", len(actions))
	}
}

func TestCheckActionSHAUpdates(t *testing.T) {
	// Create a test action cache
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)

	// Create test actions with known SHAs
	actions := []ActionUsage{
		{
			Repo:    "actions/checkout",
			SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8", // Current SHA
			Version: "v5",
		},
		{
			Repo:    "actions/setup-node",
			SHA:     "oldsha0000000000000000000000000000000000", // Outdated SHA
			Version: "v6",
		},
	}

	// Pre-populate the cache with known values
	// For actions/checkout@v5, use the same SHA (up to date)
	cache.Set("actions/checkout", "v5", "08c6903cd8c0fde910a37f88322edcfb5dd907a8")
	// For actions/setup-node@v6, use a different SHA (needs update)
	cache.Set("actions/setup-node", "v6", "newsha0000000000000000000000000000000000")

	// Create resolver with the cache
	resolver := NewActionResolver(cache)

	// Check for updates
	checks := CheckActionSHAUpdates(actions, resolver)

	// Verify results
	if len(checks) != 2 {
		t.Errorf("Expected 2 check results, got %d", len(checks))
	}

	// First action (actions/checkout) should be up to date
	if checks[0].NeedsUpdate {
		t.Errorf("Expected actions/checkout to be up to date, but it needs update")
	}

	// Second action (actions/setup-node) should need update
	if !checks[1].NeedsUpdate {
		t.Errorf("Expected actions/setup-node to need update, but it's marked as up to date")
	}
}

func TestExtractActionsFromLockFileNoActions(t *testing.T) {
	// Create a temporary lock file with no actions
	tmpDir := testutil.TempDir(t, "test-*")
	lockFile := filepath.Join(tmpDir, "test.lock.yml")

	lockContent := `
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Run tests
        run: npm test
`

	if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	// Extract actions
	actions, err := ExtractActionsFromLockFile(lockFile)
	if err != nil {
		t.Fatalf("ExtractActionsFromLockFile failed: %v", err)
	}

	// Verify we have no actions
	if len(actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(actions))
	}
}

func TestExtractActionsFromLockFileInvalidFile(t *testing.T) {
	// Try to extract from non-existent file
	_, err := ExtractActionsFromLockFile("/nonexistent/file.yml")
	if err == nil {
		t.Error("Expected error when reading non-existent file, got nil")
	}
}
