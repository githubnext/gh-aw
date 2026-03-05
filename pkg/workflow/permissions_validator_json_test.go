//go:build !integration

package workflow

import (
	"testing"
)

func TestToolsetPermissionsLoadedFromJSON(t *testing.T) {
	// Test that toolsetPermissionsMap is populated from JSON
	if len(toolsetPermissionsMap) == 0 {
		t.Fatal("toolsetPermissionsMap is empty - JSON loading failed")
	}

	// Test a few known toolsets
	expectedToolsets := []string{"context", "repos", "issues", "pull_requests", "actions"}
	for _, toolset := range expectedToolsets {
		if _, exists := toolsetPermissionsMap[toolset]; !exists {
			t.Errorf("Expected toolset %s not found in toolsetPermissionsMap", toolset)
		}
	}

	// Test specific permission mappings
	repoPerms, exists := toolsetPermissionsMap["repos"]
	if !exists {
		t.Fatal("repos toolset not found")
	}
	if len(repoPerms.ReadPermissions) == 0 {
		t.Error("repos toolset should have read permissions")
	}
	if repoPerms.ReadPermissions[0] != PermissionContents {
		t.Errorf("Expected repos read permission to be 'contents', got %v", repoPerms.ReadPermissions[0])
	}

	// Test tools list is populated
	if len(repoPerms.Tools) == 0 {
		t.Error("repos toolset should have tools listed")
	}

	// Test context toolset has no permissions but has tools
	contextPerms, exists := toolsetPermissionsMap["context"]
	if !exists {
		t.Fatal("context toolset not found")
	}
	if len(contextPerms.ReadPermissions) != 0 || len(contextPerms.WritePermissions) != 0 {
		t.Error("context toolset should have no required permissions")
	}
	if len(contextPerms.Tools) == 0 {
		t.Error("context toolset should have tools listed")
	}
}
