//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoMemoryTemplate_DoesNotContainMarkdownHeader(t *testing.T) {
	data, err := os.ReadFile("../../actions/setup/md/repo_memory_prompt.md")
	require.NoError(t, err, "should read repo memory template file")

	templateContent := string(data)
	assert.NotContains(t, templateContent, "## Repo Memory Available", "template should not include a markdown header in <repo-memory> section")
}
