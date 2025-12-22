package workflow

import (
	"testing"
)

func TestConvertToRemoteActionRef(t *testing.T) {
	t.Run("local path with ./ prefix and version tag", func(t *testing.T) {
		compiler := NewCompiler(false, "", "v1.2.3")
		ref := compiler.convertToRemoteActionRef("./actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@v1.2.3"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("local path without ./ prefix and version tag", func(t *testing.T) {
		compiler := NewCompiler(false, "", "v1.0.0")
		ref := compiler.convertToRemoteActionRef("actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@v1.0.0"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("nested action path with version tag", func(t *testing.T) {
		compiler := NewCompiler(false, "", "v2.0.0")
		ref := compiler.convertToRemoteActionRef("./actions/nested/action")
		expected := "githubnext/gh-aw/actions/nested/action@v2.0.0"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("dev version returns empty", func(t *testing.T) {
		compiler := NewCompiler(false, "", "dev")
		ref := compiler.convertToRemoteActionRef("./actions/create-issue")
		if ref != "" {
			t.Errorf("Expected empty string with 'dev' version, got %q", ref)
		}
	})

	t.Run("empty version returns empty", func(t *testing.T) {
		compiler := NewCompiler(false, "", "")
		ref := compiler.convertToRemoteActionRef("./actions/create-issue")
		if ref != "" {
			t.Errorf("Expected empty string with empty version, got %q", ref)
		}
	})
}

func TestResolveActionReference(t *testing.T) {
	tests := []struct {
		name          string
		actionMode    ActionMode
		localPath     string
		version       string
		expectedRef   string
		shouldBeEmpty bool
		description   string
	}{
		{
			name:        "dev mode",
			actionMode:  ActionModeDev,
			localPath:   "./actions/create-issue",
			version:     "v1.0.0",
			expectedRef: "./actions/create-issue",
			description: "Dev mode should return local path",
		},
		{
			name:        "release mode with version tag",
			actionMode:  ActionModeRelease,
			localPath:   "./actions/create-issue",
			version:     "v1.0.0",
			expectedRef: "githubnext/gh-aw/actions/create-issue@v1.0.0",
			description: "Release mode should return version-based reference",
		},
		{
			name:          "release mode with dev version",
			actionMode:    ActionModeRelease,
			localPath:     "./actions/create-issue",
			version:       "dev",
			shouldBeEmpty: true,
			description:   "Release mode with 'dev' version should return empty",
		},
		// Removed inline mode test case as inline mode no longer exists
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", tt.version)
			compiler.SetActionMode(tt.actionMode)

			data := &WorkflowData{}
			ref := compiler.resolveActionReference(tt.localPath, data)

			if tt.shouldBeEmpty {
				if ref != "" {
					t.Errorf("%s: expected empty string, got %q", tt.description, ref)
				}
			} else {
				if ref != tt.expectedRef {
					t.Errorf("%s: expected %q, got %q", tt.description, tt.expectedRef, ref)
				}
			}
		})
	}
}
