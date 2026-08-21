//go:build !integration

package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/parser"
)

func TestParsePRURL(t *testing.T) {
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
