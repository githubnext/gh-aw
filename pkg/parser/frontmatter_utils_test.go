//go:build !integration

package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUnderWorkflowsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "file under .github/workflows",
			filePath: "/some/path/.github/workflows/test.md",
			expected: true,
		},
		{
			name:     "file under .github/workflows subdirectory",
			filePath: "/some/path/.github/workflows/shared/helper.md",
			expected: false, // Files in subdirectories are not top-level workflow files
		},
		{
			name:     "file outside .github/workflows",
			filePath: "/some/path/docs/instructions.md",
			expected: false,
		},
		{
			name:     "file in .github but not workflows",
			filePath: "/some/path/.github/ISSUE_TEMPLATE/bug.md",
			expected: false,
		},
		{
			name:     "relative path under workflows",
			filePath: ".github/workflows/test.md",
			expected: true,
		},
		{
			name:     "relative path outside workflows",
			filePath: "docs/readme.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnderWorkflowsDirectory(tt.filePath)
			assert.Equal(t, tt.expected, result, "isUnderWorkflowsDirectory(%q)", tt.filePath)
		})
	}
}

func TestIsCustomAgentFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "file under .github/agents with .md extension",
			filePath: "/some/path/.github/agents/test-agent.md",
			expected: true,
		},
		{
			name:     "file under .github/agents with .agent.md extension",
			filePath: "/some/path/.github/agents/feature-flag-remover.agent.md",
			expected: true,
		},
		{
			name:     "file under .github/agents subdirectory",
			filePath: "/some/path/.github/agents/subdir/helper.md",
			expected: true, // Still an agent file even in subdirectory
		},
		{
			name:     "file outside .github/agents",
			filePath: "/some/path/docs/instructions.md",
			expected: false,
		},
		{
			name:     "file in .github/workflows",
			filePath: "/some/path/.github/workflows/test.md",
			expected: false,
		},
		{
			name:     "file in .github but not agents",
			filePath: "/some/path/.github/ISSUE_TEMPLATE/bug.md",
			expected: false,
		},
		{
			name:     "relative path under agents",
			filePath: ".github/agents/test-agent.md",
			expected: true,
		},
		{
			name:     "file under agents but not markdown",
			filePath: ".github/agents/config.json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCustomAgentFile(tt.filePath)
			assert.Equal(t, tt.expected, result, "isCustomAgentFile(%q)", tt.filePath)
		})
	}
}

func TestResolveIncludePath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "test_resolve")
	require.NoError(t, err, "should create temp dir")
	defer os.RemoveAll(tempDir)

	// Create regular test file in temp dir
	regularFile := filepath.Join(tempDir, "regular.md")
	err = os.WriteFile(regularFile, []byte("test"), 0644)
	require.NoError(t, err, "should write regular file")

	// Create a repo-like structure: <repoRoot>/.github/workflows/, .github/agents/ and .agents/
	repoRoot := filepath.Join(tempDir, "repo")
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
	agentsDir := filepath.Join(repoRoot, ".github", "agents")
	dotAgentsDir := filepath.Join(repoRoot, ".agents")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "should create workflows dir")
	err = os.MkdirAll(agentsDir, 0755)
	require.NoError(t, err, "should create agents dir")
	err = os.MkdirAll(dotAgentsDir, 0755)
	require.NoError(t, err, "should create .agents dir")

	workflowFile := filepath.Join(workflowsDir, "workflow.md")
	err = os.WriteFile(workflowFile, []byte("test"), 0644)
	require.NoError(t, err, "should write workflow file")

	agentFile := filepath.Join(agentsDir, "planner.md")
	err = os.WriteFile(agentFile, []byte("test"), 0644)
	require.NoError(t, err, "should write agent file")

	dotAgentFile := filepath.Join(dotAgentsDir, "agent.md")
	err = os.WriteFile(dotAgentFile, []byte("test"), 0644)
	require.NoError(t, err, "should write .agents file")

	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
		wantErr  bool
	}{
		{
			name:     "regular relative path",
			filePath: "regular.md",
			baseDir:  tempDir,
			expected: regularFile,
		},
		{
			name:     "regular file not found",
			filePath: "nonexistent.md",
			baseDir:  tempDir,
			wantErr:  true,
		},
		{
			name:     "absolute path outside base dir is rejected for security",
			filePath: "/etc/passwd",
			baseDir:  tempDir,
			wantErr:  true,
		},
		{
			name:     "dotgithub-prefixed path resolves from repo root",
			filePath: ".github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     "slash-dotgithub-prefixed path resolves from repo root",
			filePath: "/.github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     "slash-dotagents-prefixed path resolves from repo root",
			filePath: "/.agents/agent.md",
			baseDir:  workflowsDir,
			expected: dotAgentFile,
		},
		{
			name:     "slash-prefixed path outside .github or .agents is rejected",
			filePath: "/agents/agent.md",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "relative path in workflows dir still works unchanged",
			filePath: "workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},
		{
			name:     "dotgithub-prefixed path that escapes repo root is rejected",
			filePath: ".github/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "slash-prefixed path that escapes repo root is rejected",
			filePath: "/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveIncludePath(tt.filePath, tt.baseDir, nil)

			if tt.wantErr {
				assert.Error(t, err, "ResolveIncludePath(%q, %q) should return error", tt.filePath, tt.baseDir)
				return
			}

			require.NoError(t, err, "ResolveIncludePath(%q, %q) should not error", tt.filePath, tt.baseDir)
			assert.Equal(t, tt.expected, result, "ResolveIncludePath(%q, %q) result", tt.filePath, tt.baseDir)
		})
	}
}

// TestResolveIncludePath_DotGithubRepo tests import path resolution when the repository
// itself is named ".github" (e.g. an org's `org/.github` repository).  In that case the
// on-disk layout is <parent>/.github/.github/workflows/ and the traversal logic must
// correctly treat the inner ".github" directory as the special folder and
// <parent>/.github/ as the repository root.
func TestResolveIncludePath_DotGithubRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_resolve_dotgithub_repo")
	require.NoError(t, err, "should create temp dir")
	defer os.RemoveAll(tempDir)

	// Simulate a repo whose name is ".github": the repo root is <tempDir>/.github
	// and the GitHub Actions folder lives at <tempDir>/.github/.github/workflows/.
	dotGithubRepoRoot := filepath.Join(tempDir, ".github")
	workflowsDir := filepath.Join(dotGithubRepoRoot, ".github", "workflows")
	agentsDir := filepath.Join(dotGithubRepoRoot, ".github", "agents")
	dotAgentsDir := filepath.Join(dotGithubRepoRoot, ".agents")

	for _, dir := range []string{workflowsDir, agentsDir, dotAgentsDir} {
		require.NoError(t, os.MkdirAll(dir, 0755), "should create dir %s", dir)
	}

	workflowFile := filepath.Join(workflowsDir, "workflow.md")
	agentFile := filepath.Join(agentsDir, "planner.md")
	dotAgentFile := filepath.Join(dotAgentsDir, "agent.md")

	require.NoError(t, os.WriteFile(workflowFile, []byte("workflow"), 0644), "should write workflow file")
	require.NoError(t, os.WriteFile(agentFile, []byte("planner"), 0644), "should write agent file")
	require.NoError(t, os.WriteFile(dotAgentFile, []byte("dot-agents"), 0644), "should write .agents file")

	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
		wantErr  bool
	}{
		{
			name:     "relative path still resolves within workflows dir",
			filePath: "workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},
		{
			name:     "dotgithub-prefixed path resolves from repo root inside .github repo",
			filePath: ".github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     "slash-dotgithub-prefixed path resolves from repo root inside .github repo",
			filePath: "/.github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     "slash-dotagents-prefixed path resolves from repo root inside .github repo",
			filePath: "/.agents/agent.md",
			baseDir:  workflowsDir,
			expected: dotAgentFile,
		},
		{
			name:     "slash-prefixed path outside .github or .agents is rejected inside .github repo",
			filePath: "/agents/agent.md",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "dotgithub-prefixed traversal is rejected inside .github repo",
			filePath: ".github/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "slash-prefixed traversal is rejected inside .github repo",
			filePath: "/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveIncludePath(tt.filePath, tt.baseDir, nil)

			if tt.wantErr {
				assert.Error(t, err, "ResolveIncludePath(%q, %q) should return error", tt.filePath, tt.baseDir)
				return
			}

			require.NoError(t, err, "ResolveIncludePath(%q, %q) should not error", tt.filePath, tt.baseDir)
			assert.Equal(t, tt.expected, result, "ResolveIncludePath(%q, %q) result", tt.filePath, tt.baseDir)
		})
	}
}

// TestResolveIncludePath_AllPathStyles exercises every path style that
// ResolveIncludePath must handle:
//
//   - Explicit current-dir-relative  ("./file.md")
//   - Standard relative              ("file.md", "subdir/file.md")
//   - .github/-prefixed repo-root    (".github/agents/planner.md")
//   - /.github/-prefixed repo-root   ("/.github/agents/planner.md")
//   - /.agents/-prefixed repo-root   ("/.agents/agent.md")
//   - Multi-level nested paths       (".github/agents/sub/nested.md", "/.agents/sub/nested.md")
//   - Intra-.github traversal        (".github/agents/../workflows/workflow.md")
//   - Traversal that escapes scope   (".github/../../../etc/passwd")
//   - / prefix outside .github/.agents (rejected)
//   - baseDir without .github parent (plain relative fallback)
//   - Windows-style backslash paths  (normalized via filepath.ToSlash on Windows)
func TestResolveIncludePath_AllPathStyles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_all_path_styles")
	require.NoError(t, err, "should create temp dir")
	defer os.RemoveAll(tempDir)

	// Build a full repo layout:
	//   <tempDir>/repo/
	//     .github/
	//       workflows/
	//         workflow.md
	//         sub/
	//           nested.md
	//       agents/
	//         planner.md
	//         sub/
	//           nested.md
	//     .agents/
	//       agent.md
	//       sub/
	//         nested.md
	repoRoot := filepath.Join(tempDir, "repo")
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
	workflowsSubDir := filepath.Join(workflowsDir, "sub")
	agentsDir := filepath.Join(repoRoot, ".github", "agents")
	agentsSubDir := filepath.Join(agentsDir, "sub")
	dotAgentsDir := filepath.Join(repoRoot, ".agents")
	dotAgentsSubDir := filepath.Join(dotAgentsDir, "sub")

	for _, dir := range []string{workflowsSubDir, agentsSubDir, dotAgentsSubDir} {
		require.NoError(t, os.MkdirAll(dir, 0755), "should create dir %s", dir)
	}

	workflowFile := filepath.Join(workflowsDir, "workflow.md")
	workflowNestedFile := filepath.Join(workflowsSubDir, "nested.md")
	agentFile := filepath.Join(agentsDir, "planner.md")
	agentNestedFile := filepath.Join(agentsSubDir, "nested.md")
	dotAgentFile := filepath.Join(dotAgentsDir, "agent.md")
	dotAgentNestedFile := filepath.Join(dotAgentsSubDir, "nested.md")

	for _, f := range []string{workflowFile, workflowNestedFile, agentFile, agentNestedFile, dotAgentFile, dotAgentNestedFile} {
		require.NoError(t, os.WriteFile(f, []byte("test"), 0644), "should write %s", f)
	}

	// A directory outside any .github tree for the "no ancestor" tests.
	noGithubDir := filepath.Join(tempDir, "standalone")
	require.NoError(t, os.MkdirAll(noGithubDir, 0755), "should create standalone dir")
	standaloneFile := filepath.Join(noGithubDir, "helper.md")
	require.NoError(t, os.WriteFile(standaloneFile, []byte("test"), 0644), "should write standalone file")

	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
		wantErr  bool
	}{
		// ── Standard relative paths ─────────────────────────────────────────────
		{
			name:     "bare filename relative to baseDir",
			filePath: "workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},
		{
			name:     "explicit dot-slash prefix",
			filePath: "./workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},
		{
			name:     "relative subdir path",
			filePath: "sub/nested.md",
			baseDir:  workflowsDir,
			expected: workflowNestedFile,
		},

		// ── .github/-prefixed repo-root paths ───────────────────────────────────
		{
			name:     ".github/agents/planner.md resolves from repo root",
			filePath: ".github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     ".github/agents/sub/nested.md resolves from repo root",
			filePath: ".github/agents/sub/nested.md",
			baseDir:  workflowsDir,
			expected: agentNestedFile,
		},
		{
			name:     ".github/workflows/workflow.md accessible via .github prefix",
			filePath: ".github/workflows/workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},
		{
			name:     "intra-.github traversal stays within scope",
			filePath: ".github/agents/../workflows/workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},

		// ── /.github/-prefixed repo-root paths ──────────────────────────────────
		{
			name:     "/.github/agents/planner.md resolves from repo root",
			filePath: "/.github/agents/planner.md",
			baseDir:  workflowsDir,
			expected: agentFile,
		},
		{
			name:     "/.github/agents/sub/nested.md resolves from repo root",
			filePath: "/.github/agents/sub/nested.md",
			baseDir:  workflowsDir,
			expected: agentNestedFile,
		},
		{
			name:     "/.github/workflows/workflow.md accessible via slash prefix",
			filePath: "/.github/workflows/workflow.md",
			baseDir:  workflowsDir,
			expected: workflowFile,
		},

		// ── /.agents/-prefixed repo-root paths ──────────────────────────────────
		{
			name:     "/.agents/agent.md resolves from repo root",
			filePath: "/.agents/agent.md",
			baseDir:  workflowsDir,
			expected: dotAgentFile,
		},
		{
			name:     "/.agents/sub/nested.md resolves from repo root",
			filePath: "/.agents/sub/nested.md",
			baseDir:  workflowsDir,
			expected: dotAgentNestedFile,
		},

		// ── Security: traversal attempts ────────────────────────────────────────
		{
			name:     ".github prefix that escapes repo root is rejected",
			filePath: ".github/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "/.github prefix that escapes repo root is rejected",
			filePath: "/.github/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "/.agents prefix that escapes scope is rejected",
			filePath: "/.agents/../../../etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "slash prefix to disallowed top-level directory is rejected",
			filePath: "/src/main.go",
			baseDir:  workflowsDir,
			wantErr:  true,
		},
		{
			name:     "slash prefix to /etc/passwd is rejected",
			filePath: "/etc/passwd",
			baseDir:  workflowsDir,
			wantErr:  true,
		},

		// ── baseDir without a .github ancestor (plain relative fallback) ─────────
		{
			name:     "relative path from non-.github baseDir resolves locally",
			filePath: "helper.md",
			baseDir:  noGithubDir,
			expected: standaloneFile,
		},
		{
			name:     "missing file from non-.github baseDir returns error",
			filePath: "missing.md",
			baseDir:  noGithubDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveIncludePath(tt.filePath, tt.baseDir, nil)

			if tt.wantErr {
				assert.Error(t, err, "ResolveIncludePath(%q, %q) should return error", tt.filePath, tt.baseDir)
				return
			}

			require.NoError(t, err, "ResolveIncludePath(%q, %q) should not error", tt.filePath, tt.baseDir)
			assert.Equal(t, tt.expected, result, "ResolveIncludePath(%q, %q) result", tt.filePath, tt.baseDir)
		})
	}
}

func TestIsWorkflowSpec(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid workflowspec",
			path: "owner/repo/path/to/file.md",
			want: true,
		},
		{
			name: "workflowspec with ref",
			path: "owner/repo/workflows/file.md@main",
			want: true,
		},
		{
			name: "workflowspec with section",
			path: "owner/repo/workflows/file.md#section",
			want: true,
		},
		{
			name: "workflowspec with ref and section",
			path: "owner/repo/workflows/file.md@sha123#section",
			want: true,
		},
		{
			name: "local path with .github",
			path: ".github/workflows/file.md",
			want: false,
		},
		{
			name: "relative local path",
			path: "../shared/file.md",
			want: false,
		},
		{
			name: "absolute path",
			path: "/tmp/gh-aw/gh-aw/file.md",
			want: false,
		},
		{
			name: "too few parts",
			path: "owner/repo",
			want: false,
		},
		{
			name: "local path starting with dot",
			path: "./file.md",
			want: false,
		},
		{
			name: "shared path with 2 parts",
			path: "shared/file.md",
			want: false,
		},
		{
			name: "shared path with 3 parts (mcp subdirectory)",
			path: "shared/mcp/gh-aw.md",
			want: false,
		},
		{
			name: "shared path with ref",
			path: "shared/mcp/tavily.md@main",
			want: false,
		},
		{
			name: "malformed workflowspec with empty repo segment",
			path: "owner//path/file.md",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWorkflowSpec(tt.path)
			assert.Equal(t, tt.want, got, "isWorkflowSpec(%q)", tt.path)
		})
	}
}

func TestIsRepositoryImport(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{
			name:       "simple owner/repo",
			importPath: "owner/repo",
			want:       true,
		},
		{
			name:       "owner/repo with ref",
			importPath: "owner/repo@main",
			want:       true,
		},
		{
			name:       "owner/repo with SHA ref",
			importPath: "owner/repo@abc123def456",
			want:       true,
		},
		{
			name:       "owner/repo with section",
			importPath: "owner/repo#section",
			want:       true,
		},
		{
			name:       "owner/repo with ref and section",
			importPath: "owner/repo@main#section",
			want:       true,
		},
		{
			name:       "owner/repo with hyphen",
			importPath: "my-org/my-repo",
			want:       true,
		},
		{
			name:       "owner/repo with underscore",
			importPath: "my_org/my_repo",
			want:       true,
		},
		{
			name:       "workflowspec with three parts is not repository import",
			importPath: "owner/repo/path/to/file.md",
			want:       false,
		},
		{
			name:       "local relative path is not repository import",
			importPath: "relative/path.md",
			want:       false, // repo part contains a file extension
		},
		{
			name:       "local dotfile path is not repository import",
			importPath: ".github/workflows/file.md",
			want:       false,
		},
		{
			name:       "absolute path is not repository import",
			importPath: "/owner/repo",
			want:       false,
		},
		{
			name:       "shared path is not repository import",
			importPath: "shared/mcp",
			want:       false, // reserved path prefix: "shared/" is treated as a local shared directory
		},
		{
			name:       "repo with file extension is not repository import",
			importPath: "owner/repo.md",
			want:       false,
		},
		{
			name:       "single part path is not repository import",
			importPath: "just-a-name",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRepositoryImport(tt.importPath)
			assert.Equal(t, tt.want, got, "isRepositoryImport(%q)", tt.importPath)
		})
	}
}

// processImportsFromFrontmatter is a test helper that wraps ProcessImportsFromFrontmatterWithSource
// returning only the merged tools and engines (mirrors the removed production helper).
func processImportsFromFrontmatter(frontmatter map[string]any, baseDir string) (string, []string, error) {
	result, err := ProcessImportsFromFrontmatterWithSource(frontmatter, baseDir, nil, "", "")
	if err != nil {
		return "", nil, err
	}
	return result.MergedTools, result.MergedEngines, nil
}

func TestProcessImportsFromFrontmatter(t *testing.T) {
	// Create temp directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	// Create a test include file
	includeFile := filepath.Join(tempDir, "include.md")
	includeContent := `---
tools:
  bash:
    allowed:
      - ls
      - cat
---
# Include Content
This is an included file.`
	err := os.WriteFile(includeFile, []byte(includeContent), 0644)
	require.NoError(t, err, "should write include file")

	tests := []struct {
		name          string
		frontmatter   map[string]any
		wantToolsJSON bool
		wantEngines   bool
		wantErr       bool
	}{
		{
			name: "no imports field",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "empty imports array",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": []string{},
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "valid imports",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": []string{"include.md"},
			},
			wantToolsJSON: true,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "invalid imports type",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": "not-an-array",
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools, engines, err := processImportsFromFrontmatter(tt.frontmatter, tempDir)

			if tt.wantErr {
				assert.Error(t, err, "ProcessImportsFromFrontmatter() should return error")
				return
			}

			require.NoError(t, err, "ProcessImportsFromFrontmatter() should not error")

			if tt.wantToolsJSON {
				assert.NotEmpty(t, tools, "ProcessImportsFromFrontmatter() should return tools JSON")
				// Verify it's valid JSON
				var toolsMap map[string]any
				err := json.Unmarshal([]byte(tools), &toolsMap)
				require.NoError(t, err, "ProcessImportsFromFrontmatter() tools should be valid JSON")
			} else {
				assert.Empty(t, tools, "ProcessImportsFromFrontmatter() should return no tools")
			}

			if tt.wantEngines {
				assert.NotEmpty(t, engines, "ProcessImportsFromFrontmatter() should return engines")
			} else {
				assert.Empty(t, engines, "ProcessImportsFromFrontmatter() should return no engines")
			}
		})
	}
}

// TestProcessIncludedFileWithNameAndDescription verifies that name and description fields
// do not generate warnings when processing included files outside .github/workflows/
