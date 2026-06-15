//go:build !integration

package cli

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	ghmapping "github.com/github/gh-aw/pkg/github"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutcomesCommand_AddsHistorySubcommand(t *testing.T) {
	cmd := NewOutcomesCommand()
	require.NotNil(t, cmd)

	historyCmd, _, err := cmd.Find([]string{"history"})
	require.NoError(t, err)
	assert.Equal(t, "history", historyCmd.Name())
}

func TestBuildHistoricalObjectiveReport(t *testing.T) {
	mapping := &ghmapping.ObjectiveMapping{
		LabelToValue: map[string]int{
			"automation":    40,
			"testing":       65,
			"observability": 70,
		},
		MultiLabelLogic: "max",
	}

	items := []historicalGitHubItem{
		{
			Number:   1,
			Title:    "Automation only",
			URL:      "https://example.com/1",
			ClosedAt: "2026-06-01T00:00:00Z",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "automation"}},
		},
		{
			Number:   2,
			Title:    "Observability with testing",
			URL:      "https://example.com/2",
			ClosedAt: "2026-06-02T00:00:00Z",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "observability"}, {Name: "testing"}},
		},
		{
			Number:   3,
			Title:    "No mapped labels",
			URL:      "https://example.com/3",
			ClosedAt: "2026-06-03T00:00:00Z",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "docs"}},
		},
	}

	report := buildHistoricalObjectiveReport(historySourceIssues, items, mapping)

	assert.Equal(t, historySourceIssues, report.Source)
	assert.Equal(t, 3, report.SampleSize)
	assert.Equal(t, 2, report.ScoredItems)
	assert.Equal(t, 110, report.TotalObjectiveValue)
	require.Len(t, report.ObjectiveBuckets, 3)
	assert.Equal(t, "observability", report.ObjectiveBuckets[0].Label)
	assert.Equal(t, 70, report.ObjectiveBuckets[0].ContributedValue)
	assert.Equal(t, "testing", report.ObjectiveBuckets[1].Label)
	assert.Equal(t, "automation", report.ObjectiveBuckets[2].Label)
	require.Len(t, report.RepresentativeItems, 2)
	assert.Equal(t, 2, report.RepresentativeItems[0].Number)
	assert.Equal(t, 1, report.RepresentativeItems[1].Number)
}

func TestRunOutcomesHistory_JSON(t *testing.T) {
	oldRunGH := outcomesHistoryRunGH
	defer func() { outcomesHistoryRunGH = oldRunGH }()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	outcomesHistoryRunGH = func(spinnerMessage string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return []byte(`[
				{"number":101,"title":"Issue one","url":"https://example.com/issues/101","closedAt":"2026-06-08T00:00:00Z","labels":[{"name":"automation"}]}
			]`), nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
			return []byte(`[
				{"number":202,"title":"PR two","url":"https://example.com/pull/202","mergedAt":"2026-06-08T00:00:00Z","labels":[{"name":"testing"}]}
			]`), nil
		}
		return nil, assert.AnError
	}

	require.NoError(t, os.Setenv("OBJECTIVE_MAPPING_JSON", `{"label_to_value":{"automation":40,"testing":65},"multi_label_logic":"max"}`))
	defer os.Unsetenv("OBJECTIVE_MAPPING_JSON")

	err = RunOutcomesHistory(OutcomesHistoryConfig{RepoOverride: "owner/repo", JSONOutput: true, Limit: 10, Source: historySourceAll})
	require.NoError(t, err)
	require.NoError(t, w.Close())

	output, err := io.ReadAll(r)
	require.NoError(t, err)

	var data historicalObjectivesData
	require.NoError(t, json.Unmarshal(output, &data))
	require.NotNil(t, data.Issues)
	require.NotNil(t, data.PRs)
	assert.Equal(t, 40, data.Issues.TotalObjectiveValue)
	assert.Equal(t, 65, data.PRs.TotalObjectiveValue)
}

func TestRunOutcomesHistory_PrettyOutput(t *testing.T) {
	oldRunGH := outcomesHistoryRunGH
	defer func() { outcomesHistoryRunGH = oldRunGH }()

	outcomesHistoryRunGH = func(spinnerMessage string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return []byte(`[
				{"number":101,"title":"Automation issue","url":"https://example.com/issues/101","closedAt":"2026-06-08T00:00:00Z","labels":[{"name":"automation"}]}
			]`), nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
			return []byte(`[
				{"number":202,"title":"Testing PR","url":"https://example.com/pull/202","mergedAt":"2026-06-08T00:00:00Z","labels":[{"name":"testing"}]}
			]`), nil
		}
		return nil, assert.AnError
	}

	require.NoError(t, os.Setenv("OBJECTIVE_MAPPING_JSON", `{"label_to_value":{"automation":40,"testing":65},"multi_label_logic":"max"}`))
	t.Cleanup(func() { os.Unsetenv("OBJECTIVE_MAPPING_JSON") })

	stderr := testutil.CaptureStderr(t, func() {
		err := RunOutcomesHistory(OutcomesHistoryConfig{RepoOverride: "owner/repo", JSONOutput: false, Limit: 10, Source: historySourceAll})
		require.NoError(t, err)
	})

	// Top-level section header.
	assert.Contains(t, stderr, "Objective history for owner/repo (limit 10)")

	// Issues section header.
	assert.Contains(t, stderr, "ISSUES")

	// Issues stats.
	assert.Contains(t, stderr, "Sample size: 1")
	assert.Contains(t, stderr, "Scored items: 1")
	assert.Contains(t, stderr, "Total objective value: 40")

	// Issues bucket table title and headers.
	assert.Contains(t, stderr, "Top objective buckets")
	assert.Contains(t, stderr, "Bucket")
	assert.Contains(t, stderr, "Mapped Value")
	assert.Contains(t, stderr, "Contributed Value")

	// Issues representative items table title and headers.
	assert.Contains(t, stderr, "Representative items")
	assert.Contains(t, stderr, "Number")
	assert.Contains(t, stderr, "Title")

	// PRs section header.
	assert.Contains(t, stderr, "PRS")

	// PRs stats.
	assert.Contains(t, stderr, "Total objective value: 65")
}

func TestNewOutcomesHistorySubcommand_InheritsGlobalVerboseFlag(t *testing.T) {
	cmd := NewOutcomesHistorySubcommand()
	require.NotNil(t, cmd)

	assert.Nil(t, cmd.Flags().Lookup("verbose"))

	root := &cobra.Command{Use: "gh aw"}
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output showing detailed information")
	root.AddCommand(cmd)

	inherited := cmd.InheritedFlags().Lookup("verbose")
	require.NotNil(t, inherited)
	assert.Contains(t, inherited.Usage, "verbose output")
}
