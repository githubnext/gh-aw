//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditCommandDescriptionsAreConsistent(t *testing.T) {
	cmd := NewAuditCommand()

	assert.Contains(t, cmd.Short, "workflow runs", "audit short description should describe multiple run inputs")
	assert.Contains(t, cmd.Long, "Audit one or more workflow runs", "audit long description should describe multiple run inputs")
}

func TestTrialCommandUsesStandardExamplesHeading(t *testing.T) {
	cmd := NewTrialCommand(func(string) error { return nil })

	assert.Contains(t, cmd.Long, "Examples:", "trial long help should use the standard examples heading")
	assert.NotContains(t, cmd.Long, "Single workflow:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Multiple workflows (for comparison):", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Workflows from different repositories:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Repository mode examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Repeat and cleanup examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Auto-merge examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Advanced examples:", "trial long help should avoid custom example section headings")
}

func TestUpdateDocsIncludeCoolDownOption(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "should resolve current test file path")

	docsPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "src", "content", "docs", "setup", "cli.md")
	content, err := os.ReadFile(docsPath)
	require.NoError(t, err, "should read CLI setup docs")

	text := string(content)
	updateIndex := strings.Index(text, "#### `update`")
	require.NotEqual(t, -1, updateIndex, "CLI setup docs should contain the update section")

	updateSection := text[updateIndex:]
	assert.Contains(t, updateSection, "`--cool-down`", "update docs options should include --cool-down")
}

func TestSubcommandListingsUseHyphenBullets(t *testing.T) {
	tests := []struct {
		name    string
		longDoc string
	}{
		{name: "mcp", longDoc: NewMCPCommand().Long},
		{name: "project", longDoc: NewProjectCommand().Long},
		{name: "secrets", longDoc: NewSecretsCommand().Long},
		{name: "experiments", longDoc: NewExperimentsCommand().Long},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.longDoc, "Available subcommands:", "command should document available subcommands")
			assert.NotContains(t, tt.longDoc, "  • ", "subcommand list should use '-' bullet style consistently")
		})
	}
}

func TestHelpTextUsesStandardEgPunctuation(t *testing.T) {
	assert.Contains(t, coolDownFlagUsage, "(e.g., 7d", "--cool-down help should use e.g., punctuation")
	assert.Contains(t, NewEnvCommand().Long, "(e.g., default_max_turns)", "env help should use e.g., punctuation")
	assert.Contains(t, NewDomainsCommand().Long, "(e.g., \"node\", \"python\", \"github\")", "domains help should use e.g., punctuation")
	assert.Contains(t, NewChecksCommand().Long, "(e.g., Vercel,", "checks help should use e.g., punctuation")
}
