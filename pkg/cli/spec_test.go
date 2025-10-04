package cli

import (
	"testing"
)

func TestParseRepoSpec(t *testing.T) {
	tests := []struct {
		name        string
		repoSpec    string
		wantRepo    string
		wantVersion string
		wantErr     bool
		errContains string
	}{
		{
			name:        "repo with version tag",
			repoSpec:    "owner/repo@v1.0.0",
			wantRepo:    "owner/repo",
			wantVersion: "v1.0.0",
			wantErr:     false,
		},
		{
			name:        "repo with branch",
			repoSpec:    "owner/repo@main",
			wantRepo:    "owner/repo",
			wantVersion: "main",
			wantErr:     false,
		},
		{
			name:        "repo without version",
			repoSpec:    "owner/repo",
			wantRepo:    "owner/repo",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "repo with commit SHA",
			repoSpec:    "owner/repo@abc123def456",
			wantRepo:    "owner/repo",
			wantVersion: "abc123def456",
			wantErr:     false,
		},
		{
			name:        "invalid format - missing owner",
			repoSpec:    "repo@v1.0.0",
			wantErr:     true,
			errContains: "must be in format 'org/repo'",
		},
		{
			name:        "invalid format - missing repo",
			repoSpec:    "owner/@v1.0.0",
			wantErr:     true,
			errContains: "must be in format 'org/repo'",
		},
		{
			name:        "invalid format - no slash",
			repoSpec:    "ownerrepo@v1.0.0",
			wantErr:     true,
			errContains: "must be in format 'org/repo'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseRepoSpec(tt.repoSpec)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRepoSpec() expected error, got nil")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("parseRepoSpec() unexpected error: %v", err)
				return
			}

			if spec.Repo != tt.wantRepo {
				t.Errorf("parseRepoSpec() repo = %q, want %q", spec.Repo, tt.wantRepo)
			}
			if spec.Version != tt.wantVersion {
				t.Errorf("parseRepoSpec() version = %q, want %q", spec.Version, tt.wantVersion)
			}
		})
	}
}

func TestParseWorkflowSpec(t *testing.T) {
	tests := []struct {
		name             string
		spec             string
		wantRepo         string
		wantWorkflowPath string
		wantWorkflowName string
		wantVersion      string
		wantErr          bool
		errContains      string
	}{
		{
			name:             "three-part spec with version",
			spec:             "owner/repo/workflow@v1.0.0",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/workflow.md",
			wantWorkflowName: "workflow",
			wantVersion:      "v1.0.0",
			wantErr:          false,
		},
		{
			name:             "three-part spec without version",
			spec:             "owner/repo/workflow",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/workflow.md",
			wantWorkflowName: "workflow",
			wantVersion:      "",
			wantErr:          false,
		},
		{
			name:             "four-part spec with workflows prefix",
			spec:             "owner/repo/workflows/ci-doctor.md@v1.0.0",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/ci-doctor.md",
			wantWorkflowName: "ci-doctor",
			wantVersion:      "v1.0.0",
			wantErr:          false,
		},
		{
			name:             "nested path with version",
			spec:             "owner/repo/path/to/workflow.md@main",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "path/to/workflow.md",
			wantWorkflowName: "workflow",
			wantVersion:      "main",
			wantErr:          false,
		},
		{
			name:        "invalid - too few parts",
			spec:        "owner/repo@v1.0.0",
			wantErr:     true,
			errContains: "must be in format",
		},
		{
			name:        "invalid - four parts without .md extension",
			spec:        "owner/repo/workflows/workflow@v1.0.0",
			wantErr:     true,
			errContains: "must end with '.md' extension",
		},
		{
			name:        "invalid - empty owner",
			spec:        "/repo/workflow@v1.0.0",
			wantErr:     true,
			errContains: "owner and repo cannot be empty",
		},
		{
			name:        "invalid - empty repo",
			spec:        "owner//workflow@v1.0.0",
			wantErr:     true,
			errContains: "owner and repo cannot be empty",
		},
		{
			name:        "invalid - bad GitHub identifier",
			spec:        "owner-/repo/workflow@v1.0.0",
			wantErr:     true,
			errContains: "does not look like a valid GitHub repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseWorkflowSpec(tt.spec)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseWorkflowSpec() expected error, got nil")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("parseWorkflowSpec() unexpected error: %v", err)
				return
			}

			if spec.Repo != tt.wantRepo {
				t.Errorf("parseWorkflowSpec() repo = %q, want %q", spec.Repo, tt.wantRepo)
			}
			if spec.WorkflowPath != tt.wantWorkflowPath {
				t.Errorf("parseWorkflowSpec() workflowPath = %q, want %q", spec.WorkflowPath, tt.wantWorkflowPath)
			}
			if spec.WorkflowName != tt.wantWorkflowName {
				t.Errorf("parseWorkflowSpec() workflowName = %q, want %q", spec.WorkflowName, tt.wantWorkflowName)
			}
			if spec.Version != tt.wantVersion {
				t.Errorf("parseWorkflowSpec() version = %q, want %q", spec.Version, tt.wantVersion)
			}
		})
	}
}

func TestParseSourceSpec(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantRepo    string
		wantPath    string
		wantRef     string
		wantErr     bool
		errContains string
	}{
		{
			name:     "full spec with tag",
			source:   "githubnext/agentics/workflows/ci-doctor.md@v1.0.0",
			wantRepo: "githubnext/agentics",
			wantPath: "workflows/ci-doctor.md",
			wantRef:  "v1.0.0",
			wantErr:  false,
		},
		{
			name:     "full spec with branch",
			source:   "githubnext/agentics/workflows/ci-doctor.md@main",
			wantRepo: "githubnext/agentics",
			wantPath: "workflows/ci-doctor.md",
			wantRef:  "main",
			wantErr:  false,
		},
		{
			name:     "spec without ref",
			source:   "githubnext/agentics/workflows/ci-doctor.md",
			wantRepo: "githubnext/agentics",
			wantPath: "workflows/ci-doctor.md",
			wantRef:  "",
			wantErr:  false,
		},
		{
			name:     "nested path",
			source:   "owner/repo/path/to/workflow.md@v2.0.0",
			wantRepo: "owner/repo",
			wantPath: "path/to/workflow.md",
			wantRef:  "v2.0.0",
			wantErr:  false,
		},
		{
			name:        "invalid format - too few parts",
			source:      "owner/repo@v1.0.0",
			wantErr:     true,
			errContains: "invalid source format",
		},
		{
			name:        "invalid format - missing owner",
			source:      "@v1.0.0",
			wantErr:     true,
			errContains: "invalid source format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseSourceSpec(tt.source)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSourceSpec() expected error containing %q, got nil", tt.errContains)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("parseSourceSpec() unexpected error: %v", err)
				return
			}

			if spec.Repo != tt.wantRepo {
				t.Errorf("parseSourceSpec() repo = %q, want %q", spec.Repo, tt.wantRepo)
			}
			if spec.Path != tt.wantPath {
				t.Errorf("parseSourceSpec() path = %q, want %q", spec.Path, tt.wantPath)
			}
			if spec.Ref != tt.wantRef {
				t.Errorf("parseSourceSpec() ref = %q, want %q", spec.Ref, tt.wantRef)
			}
		})
	}
}

func TestBuildSourceString(t *testing.T) {
	tests := []struct {
		name     string
		workflow *WorkflowSpec
		expected string
	}{
		{
			name: "workflow with version",
			workflow: &WorkflowSpec{
				Repo:         "owner/repo",
				WorkflowPath: "workflows/ci-doctor.md",
				Version:      "v1.0.0",
			},
			expected: "owner/repo/workflows/ci-doctor.md@v1.0.0",
		},
		{
			name: "workflow with branch",
			workflow: &WorkflowSpec{
				Repo:         "owner/repo",
				WorkflowPath: "workflows/helper.md",
				Version:      "main",
			},
			expected: "owner/repo/workflows/helper.md@main",
		},
		{
			name: "workflow without version",
			workflow: &WorkflowSpec{
				Repo:         "owner/repo",
				WorkflowPath: "workflows/test.md",
				Version:      "",
			},
			expected: "owner/repo/workflows/test.md",
		},
		{
			name: "workflow with nested path",
			workflow: &WorkflowSpec{
				Repo:         "owner/repo",
				WorkflowPath: "path/to/workflow.md",
				Version:      "v2.0.0",
			},
			expected: "owner/repo/path/to/workflow.md@v2.0.0",
		},
		{
			name: "empty repo",
			workflow: &WorkflowSpec{
				Repo:         "",
				WorkflowPath: "workflows/test.md",
				Version:      "v1.0.0",
			},
			expected: "",
		},
		{
			name: "empty workflow path",
			workflow: &WorkflowSpec{
				Repo:         "owner/repo",
				WorkflowPath: "",
				Version:      "v1.0.0",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSourceString(tt.workflow)
			if result != tt.expected {
				t.Errorf("buildSourceString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidGitHubIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       bool
	}{
		{"valid alphanumeric", "myrepo123", true},
		{"valid with hyphen", "my-repo", true},
		{"valid with underscore", "my_repo", true},
		{"valid mixed", "My-Repo_123", true},
		{"invalid - starts with hyphen", "-myrepo", false},
		{"invalid - ends with hyphen", "myrepo-", false},
		{"invalid - empty string", "", false},
		{"invalid - special chars", "my@repo", false},
		{"invalid - space", "my repo", false},
		{"invalid - dot", "my.repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGitHubIdentifier(tt.identifier)
			if got != tt.want {
				t.Errorf("isValidGitHubIdentifier(%q) = %v, want %v", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestIsCommitSHA(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"valid SHA", "abc123def456789012345678901234567890abcd", true},
		{"valid SHA lowercase", "abcdef1234567890123456789012345678901234", true},
		{"valid SHA uppercase", "ABCDEF1234567890123456789012345678901234", true},
		{"valid SHA mixed case", "AbCdEf1234567890123456789012345678901234", true},
		{"invalid - too short", "abc123def456", false},
		{"invalid - too long", "abc123def456789012345678901234567890abcdef", false},
		{"invalid - contains non-hex", "abc123def456789012345678901234567890abcg", false},
		{"invalid - empty", "", false},
		{"invalid - branch name", "main", false},
		{"invalid - version tag", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommitSHA(tt.version)
			if got != tt.want {
				t.Errorf("isCommitSHA(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
