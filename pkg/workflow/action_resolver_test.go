//go:build !integration

package workflow

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "simple repo",
			repo:     "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "repo with subpath",
			repo:     "github/codeql-action/upload-sarif",
			expected: "github/codeql-action",
		},
		{
			name:     "repo with multiple subpaths",
			repo:     "owner/repo/sub/path",
			expected: "owner/repo",
		},
		{
			name:     "single part repo",
			repo:     "myrepo",
			expected: "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.ExtractBaseRepo(tt.repo)
			if result != tt.expected {
				t.Errorf("gitutil.ExtractBaseRepo(%q) = %q, want %q", tt.repo, result, tt.expected)
			}
		})
	}
}

func TestActionResolverCache(t *testing.T) {
	// Create a cache and resolver
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	// Manually add an entry to the cache
	cache.Set("actions/checkout", "v5", "test-sha-123")

	// Resolve should return cached value without making API call
	sha, err := resolver.ResolveSHA("actions/checkout", "v5")
	if err != nil {
		t.Errorf("Expected no error for cached entry, got: %v", err)
	}
	if sha != "test-sha-123" {
		t.Errorf("Expected SHA 'test-sha-123', got '%s'", sha)
	}
}

func TestActionResolverFailedResolutionCache(t *testing.T) {
	// Create a cache and resolver
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	// Attempt to resolve a non-existent action
	// This will fail since we don't have a valid GitHub API connection in tests
	repo := "nonexistent/action"
	version := "v999.999.999"

	// First attempt should try to resolve
	_, err1 := resolver.ResolveSHA(repo, version)
	if err1 == nil {
		t.Error("Expected error for non-existent action on first attempt")
	}

	// Verify the failed resolution was tracked
	cacheKey := formatActionCacheKey(repo, version)
	if !resolver.failedResolutions[cacheKey] {
		t.Errorf("Expected failed resolution to be tracked for %s", cacheKey)
	}

	// Second attempt should be skipped and return error immediately
	_, err2 := resolver.ResolveSHA(repo, version)
	if err2 == nil {
		t.Error("Expected error for non-existent action on second attempt")
	}

	// Verify the error message indicates it was skipped
	expectedErrMsg := "previously failed to resolve"
	if !strings.Contains(err2.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err2)
	}
}

// Note: Testing the actual GitHub API resolution requires network access
// and is tested in integration tests or with network-dependent test tags

// TestParseTagRefTSV verifies that ParseTagRefTSV correctly parses the tab-separated
// output produced by the GitHub API jq expression `[.object.sha, .object.type] | @tsv`.
// This is the core parsing step used when resolving action tags to SHAs; it must
// distinguish lightweight tags (type "commit") from annotated tags (type "tag") so
// that annotated tags can be peeled to their underlying commit SHA.
func TestParseTagRefTSV(t *testing.T) {
	const (
		commitSHA    = "ea222e359276c0702a5f5203547ff9d88d0ddd76"
		tagObjectSHA = "2fe53acc038ba01c3bbdc767d4b25df31ca5bdfc"
	)

	tests := []struct {
		name        string
		input       string
		wantSHA     string
		wantType    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "lightweight tag returns commit type",
			input:    commitSHA + "\tcommit\n",
			wantSHA:  commitSHA,
			wantType: "commit",
		},
		{
			name:     "annotated tag returns tag type",
			input:    tagObjectSHA + "\ttag\n",
			wantSHA:  tagObjectSHA,
			wantType: "tag",
		},
		{
			name:     "input without trailing newline",
			input:    commitSHA + "\tcommit",
			wantSHA:  commitSHA,
			wantType: "commit",
		},
		{
			name:        "empty input is rejected",
			input:       "",
			wantErr:     true,
			errContains: "unexpected format",
		},
		{
			name:        "missing tab separator is rejected",
			input:       commitSHA,
			wantErr:     true,
			errContains: "unexpected format",
		},
		{
			name:        "empty type field is rejected",
			input:       commitSHA + "\t",
			wantErr:     true,
			errContains: "unexpected format",
		},
		{
			name:        "short SHA is rejected",
			input:       "abc123\tcommit",
			wantErr:     true,
			errContains: "invalid SHA format",
		},
		{
			name:        "non-hex SHA is rejected",
			input:       "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz\tcommit",
			wantErr:     true,
			errContains: "invalid SHA format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sha, objType, err := ParseTagRefTSV(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTagRefTSV(%q): expected error, got nil", tt.input)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseTagRefTSV(%q): error = %q, want it to contain %q", tt.input, err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseTagRefTSV(%q): unexpected error: %v", tt.input, err)
				return
			}
			if sha != tt.wantSHA {
				t.Errorf("ParseTagRefTSV(%q): sha = %q, want %q", tt.input, sha, tt.wantSHA)
			}
			if objType != tt.wantType {
				t.Errorf("ParseTagRefTSV(%q): type = %q, want %q", tt.input, objType, tt.wantType)
			}
		})
	}
}

func TestFetchGitObjectPublic(t *testing.T) {
	commitSHA := "aabbccddeeff00112233445566778899aabbccdd"
	tagSHA := "1234567890abcdef1234567890abcdef12345678"

	tests := []struct {
		name       string
		body       string
		statusCode int
		wantSHA    string
		wantType   string
		wantErr    bool
	}{
		{
			name:       "commit object",
			statusCode: 200,
			body:       `{"object":{"sha":"` + commitSHA + `","type":"commit"}}`,
			wantSHA:    commitSHA,
			wantType:   "commit",
		},
		{
			name:       "annotated tag object",
			statusCode: 200,
			body:       `{"object":{"sha":"` + tagSHA + `","type":"tag"}}`,
			wantSHA:    tagSHA,
			wantType:   "tag",
		},
		{
			name:       "http error",
			statusCode: 404,
			body:       `{"message":"Not Found"}`,
			wantErr:    true,
		},
		{
			name:       "malformed json",
			statusCode: 200,
			body:       `not json`,
			wantErr:    true,
		},
		{
			name:       "missing sha field",
			statusCode: 200,
			body:       `{"object":{"type":"commit"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			sha, objType, err := fetchGitObjectPublic(srv.URL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("fetchGitObjectPublic: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchGitObjectPublic: unexpected error: %v", err)
			}
			if sha != tt.wantSHA {
				t.Errorf("sha = %q, want %q", sha, tt.wantSHA)
			}
			if objType != tt.wantType {
				t.Errorf("type = %q, want %q", objType, tt.wantType)
			}
		})
	}
}

func TestQueryLatestReleaseTagPublic(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantTag    string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: 200,
			body:       `{"tag_name":"v1.2.3","name":"Release 1.2.3"}`,
			wantTag:    "v1.2.3",
		},
		{
			name:       "not found",
			statusCode: 404,
			body:       `{"message":"Not Found"}`,
			wantErr:    true,
		},
		{
			name:       "empty tag_name",
			statusCode: 200,
			body:       `{"tag_name":""}`,
			wantErr:    true,
		},
		{
			name:       "malformed json",
			statusCode: 200,
			body:       `bad json`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			tag, err := queryLatestReleaseTagPublicFromURL(srv.URL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("queryLatestReleaseTagPublic: expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("queryLatestReleaseTagPublic: unexpected error: %v", err)
			}
			if tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", tag, tt.wantTag)
			}
		})
	}
}
