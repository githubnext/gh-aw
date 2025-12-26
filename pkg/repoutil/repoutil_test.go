package repoutil

import "testing"

func TestSplitRepoSlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "valid slug",
			slug:          "githubnext/gh-aw",
			expectedOwner: "githubnext",
			expectedRepo:  "gh-aw",
			expectError:   false,
		},
		{
			name:          "another valid slug",
			slug:          "octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			expectError:   false,
		},
		{
			name:        "invalid slug - no separator",
			slug:        "githubnext",
			expectError: true,
		},
		{
			name:        "invalid slug - multiple separators",
			slug:        "githubnext/gh-aw/extra",
			expectError: true,
		},
		{
			name:        "invalid slug - empty",
			slug:        "",
			expectError: true,
		},
		{
			name:        "invalid slug - only separator",
			slug:        "/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.expectError {
				if err == nil {
					t.Errorf("SplitRepoSlug(%q) expected error, got nil", tt.slug)
				}
			} else {
				if err != nil {
					t.Errorf("SplitRepoSlug(%q) unexpected error: %v", tt.slug, err)
				}
				if owner != tt.expectedOwner {
					t.Errorf("SplitRepoSlug(%q) owner = %q; want %q", tt.slug, owner, tt.expectedOwner)
				}
				if repo != tt.expectedRepo {
					t.Errorf("SplitRepoSlug(%q) repo = %q; want %q", tt.slug, repo, tt.expectedRepo)
				}
			}
		})
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "SSH format with .git",
			url:           "git@github.com:githubnext/gh-aw.git",
			expectedOwner: "githubnext",
			expectedRepo:  "gh-aw",
			expectError:   false,
		},
		{
			name:          "SSH format without .git",
			url:           "git@github.com:octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			expectError:   false,
		},
		{
			name:          "HTTPS format with .git",
			url:           "https://github.com/githubnext/gh-aw.git",
			expectedOwner: "githubnext",
			expectedRepo:  "gh-aw",
			expectError:   false,
		},
		{
			name:          "HTTPS format without .git",
			url:           "https://github.com/octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			expectError:   false,
		},
		{
			name:        "non-GitHub URL",
			url:         "https://gitlab.com/user/repo.git",
			expectError: true,
		},
		{
			name:        "invalid URL",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitHubURL(tt.url)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseGitHubURL(%q) expected error, got nil", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("ParseGitHubURL(%q) unexpected error: %v", tt.url, err)
				}
				if owner != tt.expectedOwner {
					t.Errorf("ParseGitHubURL(%q) owner = %q; want %q", tt.url, owner, tt.expectedOwner)
				}
				if repo != tt.expectedRepo {
					t.Errorf("ParseGitHubURL(%q) repo = %q; want %q", tt.url, repo, tt.expectedRepo)
				}
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		expected string
	}{
		{
			name:     "normal slug",
			slug:     "githubnext/gh-aw",
			expected: "githubnext-gh-aw",
		},
		{
			name:     "empty slug",
			slug:     "",
			expected: "clone-mode",
		},
		{
			name:     "slug with multiple slashes",
			slug:     "owner/repo/extra",
			expected: "owner-repo-extra",
		},
		{
			name:     "slug with hyphen",
			slug:     "owner/my-repo",
			expected: "owner-my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForFilename(tt.slug)
			if result != tt.expected {
				t.Errorf("SanitizeForFilename(%q) = %q; want %q", tt.slug, result, tt.expected)
			}
		})
	}
}

func BenchmarkSplitRepoSlug(b *testing.B) {
	slug := "githubnext/gh-aw"
	for i := 0; i < b.N; i++ {
		_, _, _ = SplitRepoSlug(slug)
	}
}

func BenchmarkParseGitHubURL(b *testing.B) {
	url := "https://github.com/githubnext/gh-aw.git"
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseGitHubURL(url)
	}
}

func BenchmarkSanitizeForFilename(b *testing.B) {
	slug := "githubnext/gh-aw"
	for i := 0; i < b.N; i++ {
		_ = SanitizeForFilename(slug)
	}
}
