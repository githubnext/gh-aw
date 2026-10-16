//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstallScriptURLCodemod(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	// Verify codemod metadata
	assert.Equal(t, "install-script-url-migration", codemod.ID, "Codemod ID should match")
	assert.Equal(t, "Migrate install script URL from githubnext/gh-aw to github/gh-aw", codemod.Name, "Codemod name should match")
	assert.NotEmpty(t, codemod.Description, "Codemod should have a description")
	assert.Equal(t, "0.9.0", codemod.IntroducedIn, "Codemod version should match")
	require.NotNil(t, codemod.Apply, "Codemod should have an Apply function")
}

func TestInstallScriptURLCodemod_RawGitHubUserContent(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - name: Install gh-aw
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"jobs": map[string]any{
			"setup": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{
						"name": "Install gh-aw",
						"run":  "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash",
					},
				},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh", "Result should contain updated URL")
	assert.NotContains(t, result, "githubnext/gh-aw", "Result should not contain old URL")
}

func TestInstallScriptURLCodemod_RefsHeadsMain(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install gh-aw extension
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh", "Result should contain updated URL")
	assert.NotContains(t, result, "githubnext/gh-aw", "Result should not contain old URL")
}

func TestInstallScriptURLCodemod_ShortForm(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install extension
        run: gh extension install githubnext/gh-aw
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "github/gh-aw", "Result should contain updated repo")
	assert.NotContains(t, result, "githubnext/gh-aw", "Result should not contain old repo")
}

func TestInstallScriptURLCodemod_AlreadyMigrated(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install gh-aw
        run: curl -fsSL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Codemod should not report changes when already migrated")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestInstallScriptURLCodemod_NoInstallScript(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Run tests
        run: npm test
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Codemod should not report changes when no install script found")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestInstallScriptURLCodemod_MultipleOccurrences(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install gh-aw
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
      - name: Install extension
        run: gh extension install githubnext/gh-aw
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh", "Result should contain updated URL")
	assert.Contains(t, result, "gh extension install github/gh-aw", "Result should contain updated repo")
	assert.NotContains(t, result, "githubnext/gh-aw", "Result should not contain old references")
}

func TestInstallScriptURLCodemod_PreservesMarkdown(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install gh-aw
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
---

# Test Workflow

This workflow installs gh-aw from githubnext.

## Steps
- Download install script
- Run install script

` + "```bash" + `
curl -fsSL https://example.com/script.sh | bash
` + "```"

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "# Test Workflow", "Result should preserve markdown")
	assert.Contains(t, result, "## Steps", "Result should preserve markdown sections")
	assert.Contains(t, result, "```bash", "Result should preserve code blocks")
	assert.Contains(t, result, "This workflow installs gh-aw from githubnext", "Result should preserve markdown text")
}

func TestInstallScriptURLCodemod_PreservesIndentation(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - name: Install gh-aw
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	// Check that indentation is preserved
	assert.Contains(t, result, "      - name: Install gh-aw", "Result should preserve indentation")
	assert.Contains(t, result, "        run: curl", "Result should preserve indentation")
}

func TestInstallScriptURLCodemod_DifferentBranches(t *testing.T) {
	t.Parallel()
	codemod := getInstallScriptURLCodemod()

	content := `---
on: workflow_dispatch
jobs:
  setup:
    steps:
      - name: Install from develop
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/develop/install-gh-aw.sh | bash
      - name: Install from specific tag
        run: curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/v1.0.0/install-gh-aw.sh | bash
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "https://raw.githubusercontent.com/github/gh-aw/develop/install-gh-aw.sh", "Result should update develop branch")
	assert.Contains(t, result, "https://raw.githubusercontent.com/github/gh-aw/v1.0.0/install-gh-aw.sh", "Result should update tag reference")
	assert.NotContains(t, result, "githubnext/gh-aw", "Result should not contain old references")
}

func TestInstallScriptURLCodemod_RepositoryBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "cao source", input: "source: githubnext/gh-aw-cao@main"},
		{name: "cao condition", input: "if: ${{ inputs.target_repo == 'githubnext/gh-aw-cao' }}"},
		{name: "starter repository", input: "env: {STARTER_REPO: githubnext/gh-aw-cao-bug-bash}"},
		{name: "raw URL for other repository", input: "run: curl https://raw.githubusercontent.com/githubnext/gh-aw-cao/main/install-gh-aw.sh"},
		{name: "underscore suffix", input: "source: githubnext/gh-aw_test@main"},
		{name: "dot suffix", input: "source: githubnext/gh-aw.test@main"},
		{name: "trailing dot repository", input: "source: githubnext/gh-aw.@main"},
		{name: "git suffix repository", input: "source: githubnext/gh-aw.git-test@main"},
		{name: "git dot suffix repository", input: "source: githubnext/gh-aw.git.test@main"},
		{name: "alphanumeric suffix", input: "source: githubnext/gh-aw2@main"},
		{name: "owner prefix", input: "run: gh extension install notgithubnext/gh-aw"},
		{name: "owner hyphen prefix", input: "run: gh extension install not-githubnext/gh-aw"},
		{name: "owner dot prefix", input: "run: gh extension install not.githubnext/gh-aw"},
		{name: "source ref", input: "source: githubnext/gh-aw@main", want: "source: github/gh-aw@main"},
		{name: "source path", input: "source: githubnext/gh-aw/workflows/test.md@main", want: "source: github/gh-aw/workflows/test.md@main"},
		{name: "GitHub URL", input: "run: curl https://github.com/githubnext/gh-aw/releases/latest", want: "run: curl https://github.com/github/gh-aw/releases/latest"},
		{name: "git URL", input: "run: git clone https://github.com/githubnext/gh-aw.git", want: "run: git clone https://github.com/github/gh-aw.git"},
		{name: "ambiguous trailing period", input: "# Migrated from githubnext/gh-aw."},
		{name: "ambiguous git trailing period", input: "# Migrated from githubnext/gh-aw.git."},
		{name: "trailing comma", input: "# Migrated from githubnext/gh-aw, previously hosted elsewhere.", want: "# Migrated from github/gh-aw, previously hosted elsewhere."},
		{name: "quoted trailing period", input: "# Migrated from 'githubnext/gh-aw'.", want: "# Migrated from 'github/gh-aw'."},
		{name: "quoted git trailing period", input: "# Migrated from 'githubnext/gh-aw.git'.", want: "# Migrated from 'github/gh-aw.git'."},
		{name: "git trailing comma", input: "# Migrated from githubnext/gh-aw.git, previously hosted elsewhere.", want: "# Migrated from github/gh-aw.git, previously hosted elsewhere."},
		{name: "dot suffix trailing comma", input: "# Migrated from githubnext/gh-aw.test, previously hosted elsewhere."},
		{name: "quoted repository", input: "env: {REPO: 'githubnext/gh-aw'}", want: "env: {REPO: 'github/gh-aw'}"},
		{name: "repeated references", input: "env: {REPOS: 'githubnext/gh-aw githubnext/gh-aw'}", want: "env: {REPOS: 'github/gh-aw github/gh-aw'}"},
		{name: "mixed references", input: "env: {REPOS: 'githubnext/gh-aw-cao githubnext/gh-aw githubnext/gh-aw-cao-bug-bash'}", want: "env: {REPOS: 'githubnext/gh-aw-cao github/gh-aw githubnext/gh-aw-cao-bug-bash'}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "---\n" + tt.input + "\n---\n\n# Test\n\ngithubnext/gh-aw\n"
			want := content
			if tt.want != "" {
				want = "---\n" + tt.want + "\n---\n\n# Test\n\ngithubnext/gh-aw\n"
			}

			codemod := getInstallScriptURLCodemod()
			result, applied, err := codemod.Apply(content, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want != "", applied)
			assert.Equal(t, want, result)

			again, appliedAgain, err := codemod.Apply(result, nil)
			require.NoError(t, err)
			assert.False(t, appliedAgain, "Migration should be idempotent")
			assert.Equal(t, result, again)
		})
	}
}
