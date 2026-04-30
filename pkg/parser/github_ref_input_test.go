//go:build !integration

package parser

import (
	"testing"
)

func TestValidateGitHubRefInput(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid formats
		{name: "owner/repo", value: "owner/repo", wantErr: false},
		{name: "owner/repo with hyphen", value: "my-org/my-repo", wantErr: false},
		{name: "owner/repo with ref", value: "owner/repo@v1.0.0", wantErr: false},
		{name: "owner/repo with SHA ref", value: "owner/repo@abc1234567890abcdef1234567890abcdef123456", wantErr: false},
		{name: "owner/repo/path", value: "owner/repo/path/to/skill", wantErr: false},
		{name: "owner/repo/path with ref", value: "owner/repo/path/to/skill@main", wantErr: false},
		{name: "microsoft package", value: "microsoft/apm-sample-package", wantErr: false},
		{name: "github skills path", value: "github/awesome-copilot/skills/review-and-refactor", wantErr: false},
		{name: "underscore in name", value: "my_org/my_repo", wantErr: false},
		{name: "dots in name", value: "my.org/my.repo", wantErr: false},
		// Invalid formats
		{name: "just owner", value: "owner", wantErr: true},
		{name: "empty string", value: "", wantErr: true},
		{name: "full HTTPS URL", value: "https://github.com/owner/repo", wantErr: true},
		{name: "slash only", value: "/", wantErr: true},
		{name: "owner slash empty repo", value: "owner/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubRefInput(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGitHubRefInput(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestParseGitHubRefParts(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantRepo    string
		wantSubpath string
		wantRef     string
	}{
		{
			name:        "owner/repo only",
			value:       "owner/repo",
			wantRepo:    "owner/repo",
			wantSubpath: "",
			wantRef:     "",
		},
		{
			name:        "owner/repo with ref",
			value:       "owner/repo@v1.0.0",
			wantRepo:    "owner/repo",
			wantSubpath: "",
			wantRef:     "v1.0.0",
		},
		{
			name:        "owner/repo/path without ref",
			value:       "github/awesome-copilot/skills/review-and-refactor",
			wantRepo:    "github/awesome-copilot",
			wantSubpath: "skills/review-and-refactor",
			wantRef:     "",
		},
		{
			name:        "owner/repo/path with ref",
			value:       "github/awesome-copilot/skills/review-and-refactor@main",
			wantRepo:    "github/awesome-copilot",
			wantSubpath: "skills/review-and-refactor",
			wantRef:     "main",
		},
		{
			name:        "microsoft package",
			value:       "microsoft/apm-sample-package",
			wantRepo:    "microsoft/apm-sample-package",
			wantSubpath: "",
			wantRef:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, subpath, ref := ParseGitHubRefParts(tt.value)
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if subpath != tt.wantSubpath {
				t.Errorf("subpath = %q, want %q", subpath, tt.wantSubpath)
			}
			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

func TestReconstructGitHubRefValue(t *testing.T) {
	tests := []struct {
		name       string
		pinnedRepo string
		subpath    string
		wantResult string
	}{
		{
			name:       "no subpath",
			pinnedRepo: "owner/repo@abc123 # v1.0.0",
			subpath:    "",
			wantResult: "owner/repo@abc123 # v1.0.0",
		},
		{
			name:       "with subpath, pinned repo",
			pinnedRepo: "github/awesome-copilot@abc123 # v2.0.0",
			subpath:    "skills/review-and-refactor",
			wantResult: "github/awesome-copilot/skills/review-and-refactor@abc123 # v2.0.0",
		},
		{
			name:       "with subpath, unpinned repo",
			pinnedRepo: "owner/repo",
			subpath:    "path/to/skill",
			wantResult: "owner/repo/path/to/skill",
		},
		{
			name:       "no subpath, unpinned repo",
			pinnedRepo: "owner/repo",
			subpath:    "",
			wantResult: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReconstructGitHubRefValue(tt.pinnedRepo, tt.subpath)
			if result != tt.wantResult {
				t.Errorf("ReconstructGitHubRefValue(%q, %q) = %q, want %q", tt.pinnedRepo, tt.subpath, result, tt.wantResult)
			}
		})
	}
}
