//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankDarwinVariants(t *testing.T) {
	ranking := rankDarwinVariants(
		[]string{"control", "challenger", "third"},
		map[string]int{"control": 2, "challenger": 5, "third": 5},
	)

	require.Len(t, ranking, 3)
	assert.Equal(t, "challenger", ranking[0].Name)
	assert.Equal(t, "third", ranking[1].Name)
	assert.Equal(t, "control", ranking[2].Name)
	assert.True(t, ranking[2].CurrentControl)
}

func TestBuildDarwinNextVariantsDefaultsToWinnerFirst(t *testing.T) {
	ranking := []DarwinVariantScore{
		{Name: "challenger", Count: 8},
		{Name: "control", Count: 5},
		{Name: "third", Count: 1},
	}

	next := buildDarwinNextVariants("challenger", []string{"control", "challenger", "third"}, ranking, nil)
	assert.Equal(t, []string{"challenger", "control", "third"}, next)
}

func TestBuildDarwinNextVariantsUsesRequestedPopulation(t *testing.T) {
	ranking := []DarwinVariantScore{{Name: "control", Count: 5}, {Name: "challenger", Count: 3}}

	next := buildDarwinNextVariants("control", []string{"control", "challenger"}, ranking, []string{"mutant-a", "control", "mutant-b"})
	assert.Equal(t, []string{"control", "mutant-a", "mutant-b"}, next)
}

func TestFindOrBuildExperimentStatsAddsMissingDeclaredVariants(t *testing.T) {
	stats := findOrBuildExperimentStats([]ExperimentVariantStats{
		{
			Name:     "style",
			Variants: map[string]int{"control": 4},
			Total:    4,
		},
	}, "style", []string{"control", "challenger"})

	assert.Equal(t, 4, stats.Total)
	assert.Equal(t, 4, stats.Variants["control"])
	assert.Equal(t, 0, stats.Variants["challenger"])
}

func TestApplyDarwinPromotionUpdatesBareArrayExperiment(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "workflow.md")
	err := os.WriteFile(workflowPath, []byte(`---
experiments:
  style: [control, challenger]
---
Body
`), 0o644)
	require.NoError(t, err)

	err = applyDarwinPromotion(workflowPath, "style", []string{"challenger", "control"})
	require.NoError(t, err)

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	cfg, err := workflow.ParseFrontmatterConfig(result.Frontmatter)
	require.NoError(t, err)
	require.NotNil(t, cfg.ExperimentConfigs["style"])
	assert.Equal(t, []string{"challenger", "control"}, cfg.ExperimentConfigs["style"].Variants)
}

func TestApplyDarwinPromotionUpdatesObjectExperiment(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "workflow.md")
	err := os.WriteFile(workflowPath, []byte(`---
experiments:
  style:
    variants: [control, challenger]
    min_samples: 10
---
Body
`), 0o644)
	require.NoError(t, err)

	err = applyDarwinPromotion(workflowPath, "style", []string{"challenger", "control"})
	require.NoError(t, err)

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	cfg, err := workflow.ParseFrontmatterConfig(result.Frontmatter)
	require.NoError(t, err)
	require.NotNil(t, cfg.ExperimentConfigs["style"])
	assert.Equal(t, []string{"challenger", "control"}, cfg.ExperimentConfigs["style"].Variants)
	assert.Equal(t, 10, cfg.ExperimentConfigs["style"].MinSamples)
}

func TestWriteDarwinArchive(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "archive", "workflow", "style", "entry.json")
	archive := &DarwinArchive{
		WorkflowID:      "workflow",
		WorkflowPath:    "/tmp/workflow.md",
		ExperimentName:  "style",
		Branch:          "experiments/workflow",
		ArchivedAt:      "20260628T122052Z",
		Winner:          "control",
		CurrentVariants: []string{"control", "challenger"},
		NextVariants:    []string{"control", "mutant"},
		Ranking:         []DarwinVariantScore{{Name: "control", Count: 4}},
		Analysis:        ExperimentAnalysis{ExperimentName: "style", Recommendation: "EXTEND"},
		State:           &ExperimentState{Counts: map[string]map[string]int{"style": {"control": 4}}},
	}

	err := writeDarwinArchive(archivePath, archive)
	require.NoError(t, err)

	content, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	var got DarwinArchive
	require.NoError(t, json.Unmarshal(content, &got))
	assert.Equal(t, "workflow", got.WorkflowID)
	assert.Equal(t, "control", got.Winner)
}
