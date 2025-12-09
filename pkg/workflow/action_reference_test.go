package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestConvertToRemoteActionRef(t *testing.T) {
	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origSHA != "" {
			os.Setenv("GITHUB_SHA", origSHA)
		} else {
			os.Unsetenv("GITHUB_SHA")
		}
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
		ResetGitSHACache()
	}()

	mockSHA := "abc123def456abc123def456abc123def456abc1"

	t.Run("local path with ./ prefix", func(t *testing.T) {
		ResetGitSHACache()
		os.Setenv("GITHUB_SHA", mockSHA)
		os.Unsetenv("GITHUB_REF")

		ref := convertToRemoteActionRef("./actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@" + mockSHA
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("local path without ./ prefix", func(t *testing.T) {
		ResetGitSHACache()
		os.Setenv("GITHUB_SHA", mockSHA)
		os.Unsetenv("GITHUB_REF")

		ref := convertToRemoteActionRef("actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@" + mockSHA
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("with tag comment", func(t *testing.T) {
		ResetGitSHACache()
		os.Setenv("GITHUB_SHA", mockSHA)
		os.Setenv("GITHUB_REF", "refs/tags/v1.2.3")

		ref := convertToRemoteActionRef("./actions/create-issue")
		expected := "githubnext/gh-aw/actions/create-issue@" + mockSHA + " # v1.2.3"
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})

	t.Run("nested action path", func(t *testing.T) {
		ResetGitSHACache()
		os.Setenv("GITHUB_SHA", mockSHA)
		os.Unsetenv("GITHUB_REF")

		ref := convertToRemoteActionRef("./actions/nested/action")
		expected := "githubnext/gh-aw/actions/nested/action@" + mockSHA
		if ref != expected {
			t.Errorf("Expected %q, got %q", expected, ref)
		}
	})
}

func TestResolveActionReference(t *testing.T) {
	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origSHA != "" {
			os.Setenv("GITHUB_SHA", origSHA)
		} else {
			os.Unsetenv("GITHUB_SHA")
		}
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
		ResetGitSHACache()
	}()

	mockSHA := "abc123def456abc123def456abc123def456abc1"

	tests := []struct {
		name         string
		actionMode   ActionMode
		localPath    string
		githubSHA    string
		githubRef    string
		expectedRef  string
		shouldBeEmpty bool
		description  string
	}{
		{
			name:        "dev mode",
			actionMode:  ActionModeDev,
			localPath:   "./actions/create-issue",
			expectedRef: "./actions/create-issue",
			description: "Dev mode should return local path",
		},
		{
			name:        "release mode without tag",
			actionMode:  ActionModeRelease,
			localPath:   "./actions/create-issue",
			githubSHA:   mockSHA,
			expectedRef: "githubnext/gh-aw/actions/create-issue@" + mockSHA,
			description: "Release mode should return SHA-pinned remote reference",
		},
		{
			name:        "release mode with tag",
			actionMode:  ActionModeRelease,
			localPath:   "./actions/create-issue",
			githubSHA:   mockSHA,
			githubRef:   "refs/tags/v1.0.0",
			expectedRef: "githubnext/gh-aw/actions/create-issue@" + mockSHA + " # v1.0.0",
			description: "Release mode should include tag comment",
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
			ResetGitSHACache()

			// Set up environment
			if tt.githubSHA != "" {
				os.Setenv("GITHUB_SHA", tt.githubSHA)
			} else {
				os.Unsetenv("GITHUB_SHA")
			}

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

func TestResolveActionReferenceIntegration(t *testing.T) {
	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origSHA != "" {
			os.Setenv("GITHUB_SHA", origSHA)
		} else {
			os.Unsetenv("GITHUB_SHA")
		}
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
		ResetGitSHACache()
	}()

	t.Run("uses actual git SHA when GITHUB_SHA not set", func(t *testing.T) {
		ResetGitSHACache()
		os.Unsetenv("GITHUB_SHA")
		os.Unsetenv("GITHUB_REF")

		compiler := NewCompiler(false, "", "1.0.0")
		compiler.SetActionMode(ActionModeRelease)

		data := &WorkflowData{}
		ref := compiler.resolveActionReference("./actions/test", data)

		if ref == "" {
			t.Error("Expected non-empty reference")
		}

		if !strings.HasPrefix(ref, "githubnext/gh-aw/actions/test@") {
			t.Errorf("Expected reference to start with 'githubnext/gh-aw/actions/test@', got %q", ref)
		}

		// Extract SHA and validate it's 40 hex chars
		parts := strings.Split(ref, "@")
		if len(parts) != 2 {
			t.Errorf("Expected reference with @ separator, got %q", ref)
			return
		}

		sha := parts[1]
		if len(sha) < 40 {
			t.Errorf("Expected SHA to be at least 40 characters, got %d: %q", len(sha), sha)
		}
	})
}
