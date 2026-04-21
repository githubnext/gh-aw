//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgenticObservabilityKitPortfolioReviewContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agentic-observability-kit.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read agentic-observability-kit workflow")

	result, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err, "Workflow frontmatter should parse")

	safeOutputs, ok := result.Frontmatter["safe-outputs"].(map[string]any)
	require.True(t, ok, "Workflow must define safe-outputs")

	uploadAsset, ok := safeOutputs["upload-asset"].(map[string]any)
	require.True(t, ok, "Workflow must configure safe-outputs.upload-asset for chart uploads")

	maxValue := 0
	switch v := uploadAsset["max"].(type) {
	case int:
		maxValue = v
	case int64:
		maxValue = int(v)
	case uint64:
		maxValue = int(v)
	case float64:
		maxValue = int(v)
	}
	require.Positive(t, maxValue, "Workflow must set upload-asset.max")
	assert.GreaterOrEqual(t, maxValue, 4, "Workflow must allow at least 4 uploaded charts")

	allowedExts, ok := uploadAsset["allowed-exts"].([]any)
	require.True(t, ok, "Workflow must define upload-asset.allowed-exts")

	var hasPNG bool
	var hasSVG bool
	for _, ext := range allowedExts {
		if ext == ".png" {
			hasPNG = true
		}
		if ext == ".svg" {
			hasSVG = true
		}
	}
	assert.True(t, hasPNG, "Workflow should allow .png chart uploads")
	assert.True(t, hasSVG, "Workflow should allow .svg chart uploads")

	imports, ok := result.Frontmatter["imports"].([]any)
	require.True(t, ok, "Workflow must define imports")
	assert.Contains(t, imports, "shared/trending-charts-simple.md", "Workflow should import trending chart helpers")

	markdown := result.Markdown
	assert.Contains(t, markdown, "Evidence-Based Repository Portfolio Review", "Workflow should include portfolio review section")
	assert.Contains(t, markdown, "Generate exactly 4 high-quality charts", "Workflow should require exactly 4 charts")
	assert.Contains(t, markdown, "Repository Portfolio Map", "Workflow should include repository portfolio map chart requirement")
	assert.Contains(t, markdown, "Workflow Overlap Matrix", "Workflow should include overlap matrix chart requirement")
}
