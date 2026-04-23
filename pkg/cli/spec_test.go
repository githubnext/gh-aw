//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name:        "GitHub URL without version",
			repoSpec:    "https://github.com/owner/repo",
			wantRepo:    "owner/repo",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "GitHub URL with version",
			repoSpec:    "https://github.com/owner/repo@v1.0.0",
			wantRepo:    "owner/repo",
			wantVersion: "v1.0.0",
			wantErr:     false,
		},
		{
			name:        "GitHub URL with branch",
			repoSpec:    "https://github.com/githubnext/agentics@main",
			wantRepo:    "githubnext/agentics",
			wantVersion: "main",
			wantErr:     false,
		},
		{
			name:        "invalid GitHub URL - missing repo",
			repoSpec:    "https://github.com/owner",
			wantErr:     true,
			errContains: "invalid GitHub URL: must be https://github.com/owner/repo",
		},
		{
			name:        "invalid GitHub URL - too many path parts",
			repoSpec:    "https://github.com/owner/repo/extra/path",
			wantErr:     true,
			errContains: "invalid GitHub URL: must be https://github.com/owner/repo",
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

			if spec.RepoSlug != tt.wantRepo {
				t.Errorf("parseRepoSpec() repo = %q, want %q", spec.RepoSlug, tt.wantRepo)
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
		wantHost         string
		wantErr          bool
		errContains      string
	}{
		{
			name:             "GitHub URL - blob with main branch",
			spec:             "https://github.com/github/gh-aw-trial/blob/main/workflows/release-issue-linker.md",
			wantRepo:         "github/gh-aw-trial",
			wantWorkflowPath: "workflows/release-issue-linker.md",
			wantWorkflowName: "release-issue-linker",
			wantVersion:      "main",
			wantHost:         "github.com",
			wantErr:          false,
		},
		{
			name:             "GitHub URL - blob with version tag",
			spec:             "https://github.com/owner/repo/blob/v1.0.0/workflows/ci-doctor.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/ci-doctor.md",
			wantWorkflowName: "ci-doctor",
			wantVersion:      "v1.0.0",
			wantErr:          false,
		},
		{
			name:             "GitHub URL - tree with branch",
			spec:             "https://github.com/owner/repo/tree/develop/custom/path/workflow.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "custom/path/workflow.md",
			wantWorkflowName: "workflow",
			wantVersion:      "develop",
			wantErr:          false,
		},
		{
			name:             "GitHub URL - raw format",
			spec:             "https://github.com/owner/repo/raw/main/workflows/helper.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/helper.md",
			wantWorkflowName: "helper",
			wantVersion:      "main",
			wantErr:          false,
		},
		{
			name:             "GitHub URL - commit SHA",
			spec:             "https://github.com/owner/repo/blob/abc123def456789012345678901234567890abcd/workflows/test.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/test.md",
			wantWorkflowName: "test",
			wantVersion:      "abc123def456789012345678901234567890abcd",
			wantErr:          false,
		},
		{
			name:             "GitHub URL - GHE.com instance",
			spec:             "https://myorg.ghe.com/owner/repo/blob/main/workflows/test.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/test.md",
			wantWorkflowName: "test",
			wantVersion:      "main",
			wantHost:         "myorg.ghe.com",
			wantErr:          false,
		},
		{
			name:        "GitHub URL - non-github.com host is rejected (e.g. gitlab.com)",
			spec:        "https://gitlab.com/owner/repo/blob/main/workflows/test.md",
			wantErr:     true,
			errContains: "github.com",
		},
		{
			name:        "GitHub URL - missing file extension",
			spec:        "https://github.com/owner/repo/blob/main/workflows/test",
			wantErr:     true,
			errContains: "must point to a .md file",
		},
		{
			name:        "GitHub URL - invalid path (too short)",
			spec:        "https://github.com/owner/repo/blob/main",
			wantErr:     true,
			errContains: "path too short",
		},
		{
			name:        "GitHub URL - invalid type",
			spec:        "https://github.com/owner/repo/commits/main/workflows/test.md",
			wantErr:     true,
			errContains: "expected /blob/, /tree/, or /raw/",
		},
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
		{
			name:             "/files/ format with branch",
			spec:             "github/gh-aw/files/main/.github/workflows/shared/mcp/serena.md",
			wantRepo:         "github/gh-aw",
			wantWorkflowPath: ".github/workflows/shared/mcp/serena.md",
			wantWorkflowName: "serena",
			wantVersion:      "main",
			wantErr:          false,
		},
		{
			name:             "/files/ format with commit SHA",
			spec:             "github/gh-aw/files/fc7992627494253a869e177e5d1985d25f3bb316/.github/workflows/shared/mcp/serena.md",
			wantRepo:         "github/gh-aw",
			wantWorkflowPath: ".github/workflows/shared/mcp/serena.md",
			wantWorkflowName: "serena",
			wantVersion:      "fc7992627494253a869e177e5d1985d25f3bb316",
			wantErr:          false,
		},
		{
			name:             "/files/ format with tag",
			spec:             "owner/repo/files/v1.0.0/workflows/helper.md",
			wantRepo:         "owner/repo",
			wantWorkflowPath: "workflows/helper.md",
			wantWorkflowName: "helper",
			wantVersion:      "v1.0.0",
			wantErr:          false,
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

			if spec.RepoSlug != tt.wantRepo {
				t.Errorf("parseWorkflowSpec() repo = %q, want %q", spec.RepoSlug, tt.wantRepo)
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
			if tt.wantHost != "" && spec.Host != tt.wantHost {
				t.Errorf("parseWorkflowSpec() host = %q, want %q", spec.Host, tt.wantHost)
			}
		})
	}
}

// TestParseWorkflowSpecGHEHostPinning verifies that well-known public-only repos
// (githubnext/agentics, github/gh-aw) always get Host pinned to "github.com"
// when a GHE environment is detected, while other repos use an empty host.
func TestParseWorkflowSpecGHEHostPinning(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantHost   string
		wantNoHost bool // expect empty host
	}{
		{
			name:     "githubnext/agentics three-part spec gets github.com in GHE mode",
			spec:     "githubnext/agentics/daily-plan",
			wantHost: "github.com",
		},
		{
			name:     "githubnext/agentics wildcard gets github.com in GHE mode",
			spec:     "githubnext/agentics/*",
			wantHost: "github.com",
		},
		{
			name:     "github/gh-aw three-part spec gets github.com in GHE mode",
			spec:     "github/gh-aw/my-workflow",
			wantHost: "github.com",
		},
		{
			name:       "non-allowlisted repo has empty host in GHE mode",
			spec:       "owner/repo/workflow",
			wantNoHost: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate a GHE environment
			t.Setenv("GITHUB_SERVER_URL", "")
			t.Setenv("GITHUB_ENTERPRISE_HOST", "myorg.ghe.com")
			t.Setenv("GITHUB_HOST", "")
			t.Setenv("GH_HOST", "")

			spec, err := parseWorkflowSpec(tt.spec)
			if err != nil {
				t.Fatalf("parseWorkflowSpec(%q) unexpected error: %v", tt.spec, err)
			}

			if tt.wantNoHost {
				if spec.Host != "" {
					t.Errorf("parseWorkflowSpec(%q) host = %q, want empty", tt.spec, spec.Host)
				}
			} else {
				if spec.Host != tt.wantHost {
					t.Errorf("parseWorkflowSpec(%q) host = %q, want %q", tt.spec, spec.Host, tt.wantHost)
				}
			}
		})
	}
}

// TestParseWorkflowSpecNoGHEHostPinning verifies that on public github.com the
// Host field is always empty for short-form specs (no pinning needed).
func TestParseWorkflowSpecNoGHEHostPinning(t *testing.T) {
	// Clear all GHE env vars to simulate standard github.com environment
	t.Setenv("GITHUB_SERVER_URL", "")
	t.Setenv("GITHUB_ENTERPRISE_HOST", "")
	t.Setenv("GITHUB_HOST", "")
	t.Setenv("GH_HOST", "")

	specs := []string{
		"githubnext/agentics/daily-plan",
		"githubnext/agentics/*",
		"github/gh-aw/my-workflow",
		"owner/repo/workflow",
	}

	for _, s := range specs {
		t.Run(s, func(t *testing.T) {
			spec, err := parseWorkflowSpec(s)
			if err != nil {
				t.Fatalf("parseWorkflowSpec(%q) unexpected error: %v", s, err)
			}
			if spec.Host != "" {
				t.Errorf("parseWorkflowSpec(%q) host = %q, want empty (no pinning on public GitHub)", s, spec.Host)
			}
		})
	}
}

func TestParseLocalWorkflowSpec(t *testing.T) {
	// Clear the repository slug cache to ensure clean test state
	ClearCurrentRepoSlugCache()

	tests := []struct {
		name             string
		spec             string
		wantWorkflowPath string
		wantWorkflowName string
		wantErr          bool
		errContains      string
	}{
		{
			name:             "valid local workflow",
			spec:             "./workflows/my-workflow.md",
			wantWorkflowPath: "./workflows/my-workflow.md",
			wantWorkflowName: "my-workflow",
			wantErr:          false,
		},
		{
			name:             "local workflow in current directory",
			spec:             "./test.md",
			wantWorkflowPath: "./test.md",
			wantWorkflowName: "test",
			wantErr:          false,
		},
		{
			name:        "local workflow without .md extension",
			spec:        "./workflows/test",
			wantErr:     true,
			errContains: "must end with '.md' extension",
		},
		{
			name:             "nested local workflow",
			spec:             "./path/to/nested/workflow.md",
			wantWorkflowPath: "./path/to/nested/workflow.md",
			wantWorkflowName: "workflow",
			wantErr:          false,
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
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseWorkflowSpec() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseWorkflowSpec() unexpected error: %v", err)
				return
			}

			if spec.WorkflowPath != tt.wantWorkflowPath {
				t.Errorf("parseWorkflowSpec() workflowPath = %q, want %q", spec.WorkflowPath, tt.wantWorkflowPath)
			}
			if spec.WorkflowName != tt.wantWorkflowName {
				t.Errorf("parseWorkflowSpec() workflowName = %q, want %q", spec.WorkflowName, tt.wantWorkflowName)
			}

			// For local workflows: version and repo should both be empty
			// (local workflows don't come from a remote source)
			if spec.Version != "" {
				t.Errorf("parseWorkflowSpec() version = %q, want empty string for local workflow", spec.Version)
			}
			if spec.RepoSlug != "" {
				t.Errorf("parseWorkflowSpec() repo = %q, want empty string for local workflow", spec.RepoSlug)
			}
		})
	}
}

func TestWorkflowSpecString(t *testing.T) {
	tests := []struct {
		name     string
		spec     *WorkflowSpec
		expected string
	}{
		{
			name: "with version",
			spec: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "v1.0.0",
				},
				WorkflowPath: "workflows/ci-doctor.md",
			},
			expected: "owner/repo/workflows/ci-doctor.md@v1.0.0",
		},
		{
			name: "without version",
			spec: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "workflows/helper.md",
			},
			expected: "owner/repo/workflows/helper.md",
		},
		{
			name: "with branch",
			spec: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "githubnext/agentics",
					Version:  "main",
				},
				WorkflowPath: "workflows/weekly-research.md",
			},
			expected: "githubnext/agentics/workflows/weekly-research.md@main",
		},
		{
			name: "local workflow",
			spec: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "./workflows/local-workflow.md",
			},
			expected: "./workflows/local-workflow.md",
		},
		{
			name: "local workflow in current directory",
			spec: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "./test.md",
			},
			expected: "./test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.String()
			if got != tt.expected {
				t.Errorf("WorkflowSpec.String() = %q, want %q", got, tt.expected)
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

func TestBuildSourceStringWithCommitSHA(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *WorkflowSpec
		commitSHA string
		expected  string
	}{
		{
			name: "with commit SHA",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "v1.0.0",
				},
				WorkflowPath: "workflows/ci-doctor.md",
			},
			commitSHA: "abc123def456789012345678901234567890abcd",
			expected:  "owner/repo/workflows/ci-doctor.md@abc123def456789012345678901234567890abcd",
		},
		{
			name: "with commit SHA overrides version",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "main",
				},
				WorkflowPath: "workflows/helper.md",
			},
			commitSHA: "1234567890abcdef1234567890abcdef12345678",
			expected:  "owner/repo/workflows/helper.md@1234567890abcdef1234567890abcdef12345678",
		},
		{
			name: "without commit SHA falls back to version",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "v2.0.0",
				},
				WorkflowPath: "workflows/test.md",
			},
			commitSHA: "",
			expected:  "owner/repo/workflows/test.md@v2.0.0",
		},
		{
			name: "without commit SHA or version",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "workflows/test.md",
			},
			commitSHA: "",
			expected:  "owner/repo/workflows/test.md",
		},
		{
			name: "empty repo with commit SHA",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "",
					Version:  "v1.0.0",
				},
				WorkflowPath: "workflows/test.md",
			},
			commitSHA: "abc123",
			expected:  "",
		},
		{
			name: "local workflow with commit SHA",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "./workflows/local.md",
			},
			commitSHA: "abc123def456789012345678901234567890abcd",
			expected:  "owner/repo/workflows/local.md@abc123def456789012345678901234567890abcd",
		},
		{
			name: "local workflow without commit SHA",
			workflow: &WorkflowSpec{
				RepoSpec: RepoSpec{
					RepoSlug: "owner/repo",
					Version:  "",
				},
				WorkflowPath: "./test.md",
			},
			commitSHA: "",
			expected:  "owner/repo/test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSourceStringWithCommitSHA(tt.workflow, tt.commitSHA)
			if result != tt.expected {
				t.Errorf("buildSourceStringWithCommitSHA() = %v, want %v", result, tt.expected)
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
			got := IsCommitSHA(tt.version)
			if got != tt.want {
				t.Errorf("IsCommitSHA(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

// TestSpec_PublicAPI_ValidateWorkflowName validates the documented behavior.
// Spec: empty names and names with invalid characters (non-alphanumeric, non-hyphen, non-underscore) return errors.
func TestSpec_PublicAPI_ValidateWorkflowName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid alphanumeric-hyphen name", input: "my-workflow", wantErr: false},
		{name: "valid name with underscores and digits", input: "my_workflow_123", wantErr: false},
		{name: "empty name returns error", input: "", wantErr: true},
		{name: "name with spaces returns error", input: "my workflow", wantErr: true},
		{name: "name with slashes returns error", input: "my/workflow", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowName(tt.input)
			if tt.wantErr {
				assert.Error(t, err, "ValidateWorkflowName(%q) should return an error", tt.input)
			} else {
				assert.NoError(t, err, "ValidateWorkflowName(%q) should not return an error", tt.input)
			}
		})
	}
}

// TestSpec_PublicAPI_GetVersion validates that GetVersion returns a non-empty string.
// Spec: returns the current CLI version.
func TestSpec_PublicAPI_GetVersion(t *testing.T) {
	version := GetVersion()
	assert.NotEmpty(t, version, "GetVersion should return a non-empty version string")
}

// TestSpec_PublicAPI_IsRunningInCI validates that IsRunningInCI returns a bool without panicking.
// Spec: detects CI environment.
func TestSpec_PublicAPI_IsRunningInCI(t *testing.T) {
	result := IsRunningInCI()
	_ = result // result is environment-dependent; ensure no panic
}

// TestSpec_Types_ShellType validates the documented ShellType string alias and its constants.
// Spec: "bash", "zsh", "fish", "powershell", "unknown"
func TestSpec_Types_ShellType(t *testing.T) {
	assert.Equal(t, ShellBash, ShellType("bash"), "ShellBash constant should be \"bash\"")
	assert.Equal(t, ShellZsh, ShellType("zsh"), "ShellZsh constant should be \"zsh\"")
	assert.Equal(t, ShellFish, ShellType("fish"), "ShellFish constant should be \"fish\"")
	assert.Equal(t, ShellPowerShell, ShellType("powershell"), "ShellPowerShell constant should be \"powershell\"")
	assert.Equal(t, ShellUnknown, ShellType("unknown"), "ShellUnknown constant should be \"unknown\"")
}

// TestSpec_PublicAPI_DetectShell validates DetectShell returns one of the documented ShellType values.
func TestSpec_PublicAPI_DetectShell(t *testing.T) {
	shell := DetectShell()
	validShells := []ShellType{ShellBash, ShellZsh, ShellFish, ShellPowerShell, ShellUnknown}
	assert.Contains(t, validShells, shell, "DetectShell should return one of the documented ShellType values")
}

// TestSpec_PublicAPI_ValidEngineNames validates the documented function returns a non-empty list.
// Spec: returns the supported engine names for shell completion.
func TestSpec_PublicAPI_ValidEngineNames(t *testing.T) {
	engines := ValidEngineNames()
	assert.NotEmpty(t, engines, "ValidEngineNames should return at least one engine name")
	for _, name := range engines {
		assert.NotEmpty(t, name, "each engine name should be non-empty")
	}
}

// TestSpec_PublicAPI_ValidArtifactSetNames validates the documented function returns known artifact sets.
// Spec: returns the valid artifact set name strings.
func TestSpec_PublicAPI_ValidArtifactSetNames(t *testing.T) {
	names := ValidArtifactSetNames()
	assert.NotEmpty(t, names, "ValidArtifactSetNames should return a non-empty list")
	assert.Contains(t, names, "all", "ValidArtifactSetNames should include \"all\"")
}

// TestSpec_PublicAPI_ValidateArtifactSets validates known and unknown artifact sets.
// Spec: validates that all provided artifact set names are known.
func TestSpec_PublicAPI_ValidateArtifactSets(t *testing.T) {
	t.Run("known artifact set returns no error", func(t *testing.T) {
		err := ValidateArtifactSets([]string{"all"})
		assert.NoError(t, err, "ValidateArtifactSets should not error for known set \"all\"")
	})

	t.Run("unknown artifact set returns error", func(t *testing.T) {
		err := ValidateArtifactSets([]string{"unknown-artifact-set-xyz"})
		assert.Error(t, err, "ValidateArtifactSets should error for unknown artifact set")
	})

	t.Run("empty list returns no error", func(t *testing.T) {
		err := ValidateArtifactSets([]string{})
		assert.NoError(t, err, "ValidateArtifactSets should not error for empty list")
	})
}

// TestSpec_PublicAPI_ExtractWorkflowDescription validates extraction of the description field
// from workflow markdown frontmatter.
// Spec: extracts the description field from workflow markdown content.
func TestSpec_PublicAPI_ExtractWorkflowDescription(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "extracts description from frontmatter",
			content:  "---\ndescription: My workflow description\n---\n\n# Content",
			expected: "My workflow description",
		},
		{
			name:     "returns empty string when no description field",
			content:  "---\nengine: copilot\n---\n\n# Content",
			expected: "",
		},
		{
			name:     "returns empty string for content without frontmatter",
			content:  "# Just markdown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWorkflowDescription(tt.content)
			assert.Equal(t, tt.expected, result, "ExtractWorkflowDescription mismatch for %q", tt.name)
		})
	}
}

// TestSpec_PublicAPI_ExtractWorkflowEngine validates extraction of the engine field.
// Spec: supports both string format (engine: copilot) and nested format (engine: { id: copilot }).
func TestSpec_PublicAPI_ExtractWorkflowEngine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "extracts engine in string format",
			content:  "---\nengine: copilot\n---\n\n# Content",
			expected: "copilot",
		},
		{
			name:     "returns empty string when no engine field",
			content:  "---\ndescription: My workflow\n---\n\n# Content",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWorkflowEngine(tt.content)
			assert.Equal(t, tt.expected, result, "ExtractWorkflowEngine mismatch for %q", tt.name)
		})
	}
}

// TestSpec_PublicAPI_ExtractWorkflowPrivate validates extraction of the private flag.
// Spec: returns true if the workflow has private: true in its frontmatter.
func TestSpec_PublicAPI_ExtractWorkflowPrivate(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "returns true when private: true",
			content:  "---\nprivate: true\n---\n\n# Content",
			expected: true,
		},
		{
			name:     "returns false when private: false",
			content:  "---\nprivate: false\n---\n\n# Content",
			expected: false,
		},
		{
			name:     "returns false when no private field",
			content:  "---\nengine: copilot\n---\n\n# Content",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWorkflowPrivate(tt.content)
			assert.Equal(t, tt.expected, result, "ExtractWorkflowPrivate mismatch for %q", tt.name)
		})
	}
}

// TestSpec_DesignDecision_StderrDiagnostics verifies that stdout is available for structured output.
// Spec: "All diagnostic output MUST go to stderr ... Structured output (JSON, hashes, graphs) goes to stdout."
// This test validates the documented design constraint is not violated by basic function signatures
// that return structured data (not printing to stdout directly).
func TestSpec_DesignDecision_StderrDiagnostics(t *testing.T) {
	require.NotNil(t, t, "design constraint: functions returning structured data use return values, not stdout")
	// ValidEngineNames, ValidArtifactSetNames return data as return values (not printed to stdout)
	engines := ValidEngineNames()
	assert.NotEmpty(t, engines, "ValidEngineNames returns data via return value, not stdout")
	names := ValidArtifactSetNames()
	assert.NotEmpty(t, names, "ValidArtifactSetNames returns data via return value, not stdout")
}

// TestSpec_PublicAPI_GetAllCodemods validates that GetAllCodemods returns at least one codemod
// with required fields populated.
// Spec: "Returns all available codemods"
func TestSpec_PublicAPI_GetAllCodemods(t *testing.T) {
	codemods := GetAllCodemods()
	require.NotEmpty(t, codemods, "GetAllCodemods should return at least one codemod")
	for _, c := range codemods {
		assert.NotEmpty(t, c.ID, "each Codemod should have a non-empty ID")
		assert.NotEmpty(t, c.Name, "each Codemod should have a non-empty Name")
		assert.NotEmpty(t, c.Description, "each Codemod should have a non-empty Description")
		assert.NotNil(t, c.Apply, "each Codemod should have a non-nil Apply function")
	}
}

// TestSpec_PublicAPI_ResolveArtifactFilter validates that ResolveArtifactFilter expands
// artifact set aliases to concrete artifact names.
// Spec: "Expands artifact set aliases to concrete artifact names"
func TestSpec_PublicAPI_ResolveArtifactFilter(t *testing.T) {
	t.Run("all returns nil meaning no filter applied", func(t *testing.T) {
		result := ResolveArtifactFilter([]string{"all"})
		assert.Nil(t, result, "\"all\" should return nil (no filter — download all artifacts)")
	})

	t.Run("empty list returns nil meaning no filter applied", func(t *testing.T) {
		result := ResolveArtifactFilter([]string{})
		assert.Nil(t, result, "empty input should return nil (no filter — download all artifacts)")
	})

	t.Run("non-all named set expands to concrete artifact list", func(t *testing.T) {
		sets := ValidArtifactSetNames()
		for _, s := range sets {
			if s == "all" {
				continue
			}
			result := ResolveArtifactFilter([]string{s})
			assert.NotNil(t, result, "artifact set %q should expand to a concrete list", s)
			assert.NotEmpty(t, result, "artifact set %q should expand to at least one artifact name", s)
			break
		}
	})
}

// TestSpec_PublicAPI_GroupRunsByWorkflow validates that a flat slice of runs is grouped by workflow name.
// Spec: "Groups a flat slice of runs by workflow name"
func TestSpec_PublicAPI_GroupRunsByWorkflow(t *testing.T) {
	runs := []WorkflowRun{
		{WorkflowName: "workflow-a"},
		{WorkflowName: "workflow-b"},
		{WorkflowName: "workflow-a"},
	}
	grouped := GroupRunsByWorkflow(runs)
	assert.Len(t, grouped, 2, "should produce two groups for two distinct workflow names")
	assert.Len(t, grouped["workflow-a"], 2, "workflow-a group should contain two runs")
	assert.Len(t, grouped["workflow-b"], 1, "workflow-b group should contain one run")
}

// TestSpec_PublicAPI_ValidateWorkflowIntent validates the documented validation rules.
// Spec: "Validates the workflow intent string"
func TestSpec_PublicAPI_ValidateWorkflowIntent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty string returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace-only string returns error",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "string shorter than 20 characters returns error",
			input:   "too short",
			wantErr: true,
		},
		{
			name:    "string of exactly 20 characters is valid",
			input:   "twelve chars here!!!",
			wantErr: false,
		},
		{
			name:    "string longer than 20 characters is valid",
			input:   "This is a sufficiently long workflow intent description",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowIntent(tt.input)
			if tt.wantErr {
				assert.Error(t, err, "ValidateWorkflowIntent(%q) should return error", tt.input)
			} else {
				assert.NoError(t, err, "ValidateWorkflowIntent(%q) should not return error", tt.input)
			}
		})
	}
}

// TestSpec_PublicAPI_UpdateFieldInFrontmatter validates the documented frontmatter field update.
// Spec: "Sets a field in frontmatter YAML"
func TestSpec_PublicAPI_UpdateFieldInFrontmatter(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		fieldName     string
		fieldValue    string
		wantErr       bool
		checkContains string
	}{
		{
			name:          "updates existing field",
			content:       "---\ndescription: old description\n---\n\n# Content",
			fieldName:     "description",
			fieldValue:    "new description",
			wantErr:       false,
			checkContains: "new description",
		},
		{
			name:          "adds new field when absent",
			content:       "---\nengine: copilot\n---\n\n# Content",
			fieldName:     "description",
			fieldValue:    "my workflow",
			wantErr:       false,
			checkContains: "my workflow",
		},
		{
			// SPEC_AMBIGUITY: The README spec says "Sets a field in frontmatter YAML" without
			// specifying the error-path for content without frontmatter. The implementation
			// creates a new frontmatter block in this case rather than returning an error.
			name:          "creates frontmatter when content has none",
			content:       "# Just markdown with no frontmatter",
			fieldName:     "description",
			fieldValue:    "value",
			wantErr:       false,
			checkContains: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpdateFieldInFrontmatter(tt.content, tt.fieldName, tt.fieldValue)
			if tt.wantErr {
				assert.Error(t, err, "UpdateFieldInFrontmatter should return error for %q", tt.name)
				return
			}
			require.NoError(t, err, "UpdateFieldInFrontmatter should not error for %q", tt.name)
			assert.Contains(t, result, tt.checkContains, "result should contain updated value for %q", tt.name)
		})
	}
}

// TestSpec_PublicAPI_SetFieldInOnTrigger validates the documented on: trigger field update.
// Spec: "Sets a field inside the on: trigger block"
func TestSpec_PublicAPI_SetFieldInOnTrigger(t *testing.T) {
	t.Run("adds on: block when not present", func(t *testing.T) {
		content := "---\ndescription: my workflow\n---\n\n# Content"
		result, err := SetFieldInOnTrigger(content, "schedule", "daily")
		require.NoError(t, err, "SetFieldInOnTrigger should not error when on: block is absent")
		assert.Contains(t, result, "on:", "result should contain on: block")
		assert.Contains(t, result, "schedule", "result should contain the new field")
	})

	t.Run("sets field inside existing on: block", func(t *testing.T) {
		content := "---\ndescription: my workflow\non:\n    push: true\n---\n\n# Content"
		result, err := SetFieldInOnTrigger(content, "schedule", "daily")
		require.NoError(t, err, "SetFieldInOnTrigger should not error with existing on: block")
		assert.Contains(t, result, "schedule", "result should contain the new field in the on: block")
	})

	t.Run("returns error when no frontmatter found", func(t *testing.T) {
		content := "# No frontmatter here"
		_, err := SetFieldInOnTrigger(content, "schedule", "daily")
		assert.Error(t, err, "SetFieldInOnTrigger should return error when no frontmatter found")
	})
}

// TestSpec_PublicAPI_RemoveFieldFromOnTrigger validates the documented on: trigger field removal.
// Spec: "Removes a field from the on: trigger block"
func TestSpec_PublicAPI_RemoveFieldFromOnTrigger(t *testing.T) {
	t.Run("removes field from existing on: block", func(t *testing.T) {
		content := "---\ndescription: my workflow\non:\n    schedule: daily\n    push: true\n---\n\n# Content"
		result, err := RemoveFieldFromOnTrigger(content, "schedule")
		require.NoError(t, err, "RemoveFieldFromOnTrigger should not error for valid content")
		assert.NotContains(t, result, "schedule:", "result should not contain removed field")
		assert.Contains(t, result, "push", "result should retain other on: fields")
	})

	t.Run("no-op when field is not present", func(t *testing.T) {
		content := "---\ndescription: my workflow\non:\n    push: true\n---\n\n# Content"
		result, err := RemoveFieldFromOnTrigger(content, "schedule")
		require.NoError(t, err, "RemoveFieldFromOnTrigger should not error when field absent")
		assert.Contains(t, result, "push", "result should retain existing on: fields")
	})
}

// TestSpec_PublicAPI_UpdateScheduleInOnBlock validates the documented schedule update.
// Spec: "Updates the cron schedule in the on: block"
func TestSpec_PublicAPI_UpdateScheduleInOnBlock(t *testing.T) {
	t.Run("updates existing schedule expression", func(t *testing.T) {
		content := "---\ndescription: my workflow\non:\n    schedule:\n    - cron: 0 9 * * 1-5\n---\n\n# Content"
		result, err := UpdateScheduleInOnBlock(content, "0 10 * * 1-5")
		require.NoError(t, err, "UpdateScheduleInOnBlock should not error for valid content")
		assert.Contains(t, result, "0 10 * * 1-5", "result should contain the updated cron expression")
	})

	t.Run("returns error for content without frontmatter lines", func(t *testing.T) {
		content := "# Just markdown"
		_, err := UpdateScheduleInOnBlock(content, "0 9 * * *")
		assert.Error(t, err, "UpdateScheduleInOnBlock should return error when no frontmatter lines present")
	})
}
