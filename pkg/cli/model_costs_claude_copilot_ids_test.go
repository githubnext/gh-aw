//go:build !integration

package cli

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/workflow"
)

// TestCopilotClaudeModelIDsMatchCatalog fails when the Copilot Claude model ids that the
// compiler uses for Claude Code's modelOverrides map drift from the embedded catalog.
func TestCopilotClaudeModelIDsMatchCatalog(t *testing.T) {
	t.Parallel()
	var catalog modelsCatalogData
	require.NoError(t, json.Unmarshal(modelsJSON, &catalog))
	provider, ok := catalog.Providers["github-copilot"]
	require.True(t, ok, "models.json should have a github-copilot provider")

	var catalogIDs []string
	for id := range provider.Models {
		if strings.HasPrefix(id, "claude-") {
			catalogIDs = append(catalogIDs, id)
		}
	}
	slices.Sort(catalogIDs)

	compilerIDs := workflow.CopilotClaudeModelIDs()
	slices.Sort(compilerIDs)
	assert.Equal(t, catalogIDs, compilerIDs, "update copilotClaudeModelIDs in pkg/workflow/claude_copilot_model_overrides.go to match pkg/cli/data/models.json")
}
