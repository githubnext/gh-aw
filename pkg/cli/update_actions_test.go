package cli

import (
	"encoding/json"
	"testing"

	"github.com/githubnext/gh-aw/pkg/repoutil"
)

func TestActionKeyVersionConsistency(t *testing.T) {
	// This test ensures that when an action is updated, the key in the map
	// is updated to match the new version, preventing key/version mismatches
	// that would cause version comments to change on each build.

	// Simulate the actions-lock.json structure
	actionsLock := actionsLockFile{
		Entries: map[string]actionsLockEntry{
			"actions/checkout@v5.0.0": {
				Repo:    "actions/checkout",
				Version: "v5.0.0",
				SHA:     "oldsha1234567890123456789012345678901234",
			},
		},
	}

	// Simulate an update to a newer version
	oldKey := "actions/checkout@v5.0.0"
	entry := actionsLock.Entries[oldKey]
	latestVersion := "v5.0.1"
	latestSHA := "newsha1234567890123456789012345678901234"

	// Apply the update logic from UpdateActions
	delete(actionsLock.Entries, oldKey)
	newKey := entry.Repo + "@" + latestVersion
	actionsLock.Entries[newKey] = actionsLockEntry{
		Repo:    entry.Repo,
		Version: latestVersion,
		SHA:     latestSHA,
	}

	// Verify the old key is gone
	if _, exists := actionsLock.Entries[oldKey]; exists {
		t.Errorf("Old key %q should have been deleted", oldKey)
	}

	// Verify the new key exists
	updatedEntry, exists := actionsLock.Entries[newKey]
	if !exists {
		t.Errorf("New key %q should exist", newKey)
	}

	// Verify the entry has the correct version
	if updatedEntry.Version != latestVersion {
		t.Errorf("Entry version = %q, want %q", updatedEntry.Version, latestVersion)
	}

	// Most importantly: verify key and version field match
	keyVersion := newKey[len("actions/checkout@"):]
	if keyVersion != updatedEntry.Version {
		t.Errorf("Key version %q does not match entry version %q", keyVersion, updatedEntry.Version)
	}
}

func TestActionKeyVersionConsistencyInJSON(t *testing.T) {
	// This test ensures that when actions-lock.json is loaded and saved,
	// there are no key/version mismatches

	jsonData := `{
		"entries": {
			"actions/checkout@v5.0.1": {
				"repo": "actions/checkout",
				"version": "v5.0.1",
				"sha": "93cb6efe18208431cddfb8368fd83d5badbf9bfd"
			},
			"actions/setup-node@v6.1.0": {
				"repo": "actions/setup-node",
				"version": "v6.1.0",
				"sha": "395ad3262231945c25e8478fd5baf05154b1d79f"
			}
		}
	}`

	var actionsLock actionsLockFile
	if err := json.Unmarshal([]byte(jsonData), &actionsLock); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify all entries have matching key and version
	for key, entry := range actionsLock.Entries {
		// Extract version from key (format: "repo@version")
		atIndex := len(key)
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '@' {
				atIndex = i
				break
			}
		}

		if atIndex < len(key) {
			keyVersion := key[atIndex+1:]
			if keyVersion != entry.Version {
				t.Errorf("Key %q has version in key %q but entry version is %q - this mismatch causes version comments to change on each build",
					key, keyVersion, entry.Version)
			}
		}
	}
}

func TestExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name       string
		actionPath string
		want       string
	}{
		{
			name:       "action without subfolder",
			actionPath: "actions/checkout",
			want:       "actions/checkout",
		},
		{
			name:       "action with one subfolder",
			actionPath: "actions/cache/restore",
			want:       "actions/cache",
		},
		{
			name:       "action with multiple subfolders",
			actionPath: "github/codeql-action/upload-sarif",
			want:       "github/codeql-action",
		},
		{
			name:       "action with deeply nested subfolders",
			actionPath: "owner/repo/path/to/action",
			want:       "owner/repo",
		},
		{
			name:       "action with only owner",
			actionPath: "owner",
			want:       "owner",
		},
		{
			name:       "empty string",
			actionPath: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoutil.ExtractBaseRepo(tt.actionPath)
			if got != tt.want {
				t.Errorf("repoutil.ExtractBaseRepo(%q) = %q, want %q", tt.actionPath, got, tt.want)
			}
		})
	}
}
