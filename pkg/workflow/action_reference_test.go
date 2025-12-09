package workflow

import (
	"os"
	"testing"
)

func TestConvertToRemoteActionRef(t *testing.T) {
	// Save original environment
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
	}()

	t.Run("local path with ./ prefix and tag", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/tags/v1.2.3")

		ref := convertToRemoteActionRef("./actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@v1.2.3"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("local path without ./ prefix and tag", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/tags/v1.0.0")

		ref := convertToRemoteActionRef("actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@v1.0.0"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("nested action path with tag", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/tags/v2.0.0")

		ref := convertToRemoteActionRef("./actions/nested/action")
		expected := "githubnext/gh-aw/actions/nested/action@v2.0.0"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("no tag returns empty", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/heads/main")

		ref := convertToRemoteActionRef("./actions/create-issue")
		if ref != "" {
			t.Errorf("Expected empty string without tag, got %q", ref)
		}
	})
}

func TestResolveActionReference(t *testing.T) {
	// Save original environment
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
	}()

	tests := []struct {
		name          string
		actionMode    ActionMode
		localPath     string
		githubRef     string
		expectedRef   string
		shouldBeEmpty bool
		description   string
	}{
		{
			name:        "dev mode",
			actionMode:  ActionModeDev,
			localPath:   "./actions/create-issue",
			expectedRef: "./actions/create-issue",
			description: "Dev mode should return local path",
		},
		{
			name:        "release mode with tag",
			actionMode:  ActionModeRelease,
			localPath:   "./actions/create-issue",
			githubRef:   "refs/tags/v1.0.0",
			expectedRef: "githubnext/gh-aw/actions/create-issue@v1.0.0",
			description: "Release mode should return tag-based reference",
		},
		{
			name:          "release mode without tag",
			actionMode:    ActionModeRelease,
			localPath:     "./actions/create-issue",
			githubRef:     "refs/heads/main",
			shouldBeEmpty: true,
			description:   "Release mode without tag should return empty",
		},
		{
			name:          "inline mode",
			actionMode:    ActionModeInline,
			localPath:     "./actions/create-issue",
			shouldBeEmpty: true,
			description:   "Inline mode should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.githubRef != "" {
				os.Setenv("GITHUB_REF", tt.githubRef)
			} else {
				os.Unsetenv("GITHUB_REF")
			}

			compiler := NewCompiler(false, "", "1.0.0")
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
