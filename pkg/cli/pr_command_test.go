//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/parser"
)

func TestCreatePRForRepoSkipsRepositoryLookup(t *testing.T) {
	originalRunGH := createPRRunGHContextWithHost
	t.Cleanup(func() { createPRRunGHContextWithHost = originalRunGH })

	var calls [][]string
	createPRRunGHContextWithHost = func(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			t.Fatal("known repository slug must skip gh repo view")
		}
		return []byte("https://github.com/owner/repo/pull/42\n"), nil
	}

	prNumber, prURL, err := createPRForRepo(context.Background(), "feature", "Title", "Body", "owner/repo", false)
	if err != nil {
		t.Fatalf("createPRForRepo() error = %v", err)
	}
	if prNumber != 42 || prURL != "https://github.com/owner/repo/pull/42" {
		t.Fatalf("createPRForRepo() = (%d, %q), want (42, PR URL)", prNumber, prURL)
	}
	if len(calls) != 1 {
		t.Fatalf("gh call count = %d, want 1", len(calls))
	}
	args := strings.Join(calls[0], " ")
	if !strings.Contains(args, "pr create --repo owner/repo") {
		t.Fatalf("gh args = %q, want explicit repository", args)
	}
}

func TestCreatePatchFromPRWritesOnlyDiff(t *testing.T) {
	originalRunGH := prRunGH
	t.Cleanup(func() { prRunGH = originalRunGH })

	diff := []byte("diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n")
	prRunGH = func(_ string, _ ...string) ([]byte, error) {
		return diff, nil
	}

	patchFile, err := createPatchFromPR("owner", "repo", &PRInfo{
		Number:      42,
		Title:       "title",
		Body:        "message\n---\ndiff --git a/injected b/injected",
		HeadSHA:     "sha",
		AuthorLogin: "author",
	}, false)
	if err != nil {
		t.Fatalf("createPatchFromPR() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(patchFile)) })

	got, err := os.ReadFile(patchFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(diff) {
		t.Fatalf("patch contents = %q, want raw diff %q", got, diff)
	}
}

func TestParsePRURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantPR    int
		wantErr   bool
	}{
		{
			name:      "valid GitHub PR URL",
			url:       "https://github.com/trial/repo/pull/234",
			wantOwner: "trial",
			wantRepo:  "repo",
			wantPR:    234,
			wantErr:   false,
		},
		{
			name:      "valid GitHub PR URL with hyphenated repo name",
			url:       "https://github.com/PR-OWNER/PR-REPO/pull/456",
			wantOwner: "PR-OWNER",
			wantRepo:  "PR-REPO",
			wantPR:    456,
			wantErr:   false,
		},
		{
			name:      "valid GitHub PR URL with underscores",
			url:       "https://github.com/test_owner/test_repo/pull/789",
			wantOwner: "test_owner",
			wantRepo:  "test_repo",
			wantPR:    789,
			wantErr:   false,
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:      "non-GitHub URL with valid path structure",
			url:       "https://gitlab.com/owner/repo/pull/123",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantPR:    123,
			wantErr:   false,
		},
		{
			name:    "invalid GitHub URL path - missing pull",
			url:     "https://github.com/owner/repo/123",
			wantErr: true,
		},
		{
			name:    "invalid GitHub URL path - wrong format",
			url:     "https://github.com/owner/repo/pulls/123",
			wantErr: true,
		},
		{
			name:    "invalid PR number",
			url:     "https://github.com/owner/repo/pull/abc",
			wantErr: true,
		},
		{
			name:    "missing owner",
			url:     "https://github.com//repo/pull/123",
			wantErr: true,
		},
		{
			name:    "missing repo",
			url:     "https://github.com/owner//pull/123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, repo, prNumber, err := parser.ParsePRURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePRURL() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePRURL() unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("ParsePRURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParsePRURL() repo = %v, want %v", repo, tt.wantRepo)
			}

			if prNumber != tt.wantPR {
				t.Errorf("ParsePRURL() prNumber = %v, want %v", prNumber, tt.wantPR)
			}
		})
	}
}

func TestPullRequestSupportsTransferAndAutomergeJSON(t *testing.T) {
	t.Parallel()

	var transferPR PRInfo
	if err := json.Unmarshal([]byte(`{"number":123,"title":"Test PR","state":"open","authorLogin":"test-author"}`), &transferPR); err != nil {
		t.Fatalf("failed to decode transfer PR: %v", err)
	}
	if transferPR.Number != 123 || transferPR.Title != "Test PR" || transferPR.State != "open" || transferPR.AuthorLogin != "test-author" {
		t.Fatalf("unexpected transfer PR: %+v", transferPR)
	}

	var automergePR PullRequest
	if err := json.Unmarshal([]byte(`{"number":456,"title":"Automerge PR","isDraft":true,"mergeable":"MERGEABLE","createdAt":"2026-08-20T12:00:00Z","updatedAt":"2026-08-20T13:00:00Z"}`), &automergePR); err != nil {
		t.Fatalf("failed to decode automerge PR: %v", err)
	}
	if automergePR.Number != 456 || !automergePR.IsDraft || automergePR.Mergeable != "MERGEABLE" {
		t.Fatalf("unexpected automerge PR: %+v", automergePR)
	}
	if want := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC); !automergePR.CreatedAt.Equal(want) {
		t.Fatalf("CreatedAt = %v, want %v", automergePR.CreatedAt, want)
	}
}

// TestNewPRCommand tests that the PR command is created properly
func TestNewPRCommand(t *testing.T) {
	t.Parallel()
	cmd := NewPRCommand()

	if cmd.Use != "pr" {
		t.Errorf("Expected command use to be 'pr', got %s", cmd.Use)
	}

	if cmd.Short != "Pull request utilities" {
		t.Errorf("Expected command short description to be 'Pull request utilities', got %s", cmd.Short)
	}
	if !strings.Contains(cmd.Long, "provides a tool for transferring pull requests") {
		t.Errorf("Expected command long description to document a single transfer tool, got %s", cmd.Long)
	}

	// Check that transfer subcommand is added
	subcommands := cmd.Commands()
	found := false
	for _, subcmd := range subcommands {
		if subcmd.Use == "transfer <pr-url>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'transfer' subcommand to be added to pr command")
	}
}

// TestNewPRTransferSubcommand tests that the transfer subcommand is created properly
func TestNewPRTransferSubcommand(t *testing.T) {
	t.Parallel()
	cmd := NewPRTransferSubcommand()

	if cmd.Use != "transfer <pr-url>" {
		t.Errorf("Expected command use to be 'transfer <pr-url>', got %s", cmd.Use)
	}

	if cmd.Short != "Transfer a pull request to another repository" {
		t.Errorf("Expected command short description to match, got %s", cmd.Short)
	}

	// Check that --repo flag exists
	repoFlag := cmd.Flags().Lookup("repo")
	if repoFlag == nil {
		t.Error("Expected --repo flag to exist")
	}
}
