//go:build !integration

package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractExperimentName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "remote ref with origin prefix",
			ref:      "origin/experiments/my-feature",
			expected: "my-feature",
		},
		{
			name:     "local ref without origin prefix",
			ref:      "experiments/my-feature",
			expected: "my-feature",
		},
		{
			name:     "nested experiment name",
			ref:      "experiments/team/feature-x",
			expected: "team/feature-x",
		},
		{
			name:     "remote nested ref",
			ref:      "origin/experiments/team/feature-x",
			expected: "team/feature-x",
		},
		{
			name:     "unrelated branch returns empty",
			ref:      "origin/main",
			expected: "",
		},
		{
			name:     "feature branch without prefix returns empty",
			ref:      "feature/my-feature",
			expected: "",
		},
		{
			name:     "bare experiments prefix returns empty",
			ref:      "experiments/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExperimentName(tt.ref)
			assert.Equal(t, tt.expected, got, "extractExperimentName(%q)", tt.ref)
		})
	}
}

func TestComputeExperimentStatus(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	thirtyOneDaysAgo := time.Now().AddDate(0, 0, -(experimentsStaleThresholdDays + 1)).Format("2006-01-02")
	twentyNineDaysAgo := time.Now().AddDate(0, 0, -(experimentsStaleThresholdDays - 1)).Format("2006-01-02")

	tests := []struct {
		name     string
		dateStr  string
		expected string
	}{
		{
			name:     "today is active",
			dateStr:  today,
			expected: "active",
		},
		{
			name:     "29 days ago is active",
			dateStr:  twentyNineDaysAgo,
			expected: "active",
		},
		{
			name:     "31 days ago is stale",
			dateStr:  thirtyOneDaysAgo,
			expected: "stale",
		},
		{
			name:     "empty string returns unknown",
			dateStr:  "",
			expected: "unknown",
		},
		{
			name:     "invalid date returns unknown",
			dateStr:  "not-a-date",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeExperimentStatus(tt.dateStr)
			assert.Equal(t, tt.expected, got, "computeExperimentStatus(%q)", tt.dateStr)
		})
	}
}

func TestParseForEachRefOutput(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		name          string
		output        string
		expectedNames []string
		expectedCount int
	}{
		{
			name:          "parses single remote branch",
			output:        "origin/experiments/feature-a|Alice|" + today + "|Add initial feature\n",
			expectedNames: []string{"feature-a"},
			expectedCount: 1,
		},
		{
			name: "parses multiple branches",
			output: strings.Join([]string{
				"origin/experiments/feature-a|Alice|" + today + "|Feature A",
				"origin/experiments/feature-b|Bob|2024-01-01|Feature B",
			}, "\n") + "\n",
			expectedNames: []string{"feature-a", "feature-b"},
			expectedCount: 2,
		},
		{
			name: "deduplicates local and remote refs for same experiment",
			output: strings.Join([]string{
				"origin/experiments/feature-a|Alice|" + today + "|From remote",
				"experiments/feature-a|Alice|" + today + "|From local",
			}, "\n") + "\n",
			expectedNames: []string{"feature-a"},
			expectedCount: 1,
		},
		{
			name:          "empty output returns empty slice",
			output:        "",
			expectedNames: []string{},
			expectedCount: 0,
		},
		{
			name: "ignores unrelated branches",
			output: strings.Join([]string{
				"origin/main|Alice|" + today + "|Main branch",
				"origin/feature/foo|Bob|" + today + "|Feature branch",
			}, "\n") + "\n",
			expectedNames: []string{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseForEachRefOutput(tt.output)
			require.Len(t, got, tt.expectedCount, "expected %d experiments", tt.expectedCount)

			for i, name := range tt.expectedNames {
				assert.Equal(t, name, got[i].Name, "experiment[%d].Name", i)
				assert.Equal(t, experimentsBranchPrefix+name, got[i].Branch, "experiment[%d].Branch", i)
			}
		})
	}
}

func TestParsePagedJSONArray(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name          string
		input         string
		expectedCount int
		shouldErr     bool
	}{
		{
			name:          "single page",
			input:         `[{"name":"a"},{"name":"b"}]`,
			expectedCount: 2,
		},
		{
			name:          "two pages",
			input:         `[{"name":"a"}][{"name":"b"},{"name":"c"}]`,
			expectedCount: 3,
		},
		{
			name:          "empty array",
			input:         `[]`,
			expectedCount: 0,
		},
		{
			name:      "invalid JSON",
			input:     `{not valid}`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePagedJSONArray[item](tt.input)
			if tt.shouldErr {
				assert.Error(t, err, "should return an error for invalid JSON")
				return
			}
			require.NoError(t, err, "should parse successfully")
			assert.Len(t, got, tt.expectedCount, "expected %d items", tt.expectedCount)
		})
	}
}

func TestExperimentInfoJSONOutput(t *testing.T) {
	experiments := []ExperimentInfo{
		{
			Name:       "my-feature",
			Author:     "Alice",
			LastCommit: "2024-01-15",
			Status:     "stale",
			Branch:     "experiments/my-feature",
		},
	}

	jsonBytes, err := json.MarshalIndent(experiments, "", "  ")
	require.NoError(t, err, "should marshal ExperimentInfo to JSON")

	var result []map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &result), "should unmarshal JSON back")

	require.Len(t, result, 1, "should have 1 experiment")
	assert.Equal(t, "my-feature", result[0]["name"], "name field should match")
	assert.Equal(t, "Alice", result[0]["author"], "author field should match")
	assert.Equal(t, "2024-01-15", result[0]["last_commit"], "last_commit field should match")
	assert.Equal(t, "stale", result[0]["status"], "status field should match")
	assert.Equal(t, "experiments/my-feature", result[0]["branch"], "branch field should match")
}

func TestExperimentDetailsJSONOutput(t *testing.T) {
	details := ExperimentDetails{
		Name:        "my-feature",
		Branch:      "experiments/my-feature",
		Author:      "Alice",
		LastCommit:  "2024-01-15",
		Status:      "stale",
		CommitCount: 3,
		Commits: []ExperimentCommit{
			{SHA: "abc1234", Message: "Initial commit", Author: "Alice", Date: "2024-01-15"},
		},
		PRs: []ExperimentPR{
			{Number: 42, Title: "Experiment: my feature", State: "open", URL: "https://github.com/owner/repo/pull/42"},
		},
	}

	jsonBytes, err := json.MarshalIndent(details, "", "  ")
	require.NoError(t, err, "should marshal ExperimentDetails to JSON")

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &result), "should unmarshal JSON back")

	assert.Equal(t, "my-feature", result["name"], "name field should match")
	assert.EqualValues(t, 3, result["commit_count"], "commit_count should match")

	commits, ok := result["commits"].([]any)
	require.True(t, ok, "commits should be an array")
	require.Len(t, commits, 1, "should have 1 commit")

	prs, ok := result["prs"].([]any)
	require.True(t, ok, "prs should be an array")
	require.Len(t, prs, 1, "should have 1 PR")
}

func TestNewExperimentsCommand(t *testing.T) {
	cmd := NewExperimentsCommand()
	require.NotNil(t, cmd, "command should be created")
	assert.Equal(t, "experiments", cmd.Name(), "command name should be experiments")

	subCmds := cmd.Commands()
	subNames := make([]string, 0, len(subCmds))
	for _, sub := range subCmds {
		subNames = append(subNames, sub.Name())
	}

	assert.Contains(t, subNames, "list", "should have list subcommand")
	assert.Contains(t, subNames, "analyze", "should have analyze subcommand")
}

func TestExperimentsListSubcommandFlags(t *testing.T) {
	cmd := NewExperimentsListSubcommand()
	require.NotNil(t, cmd, "list subcommand should be created")

	assert.NotNil(t, cmd.Flag("json"), "should have --json flag")
	assert.NotNil(t, cmd.Flag("repo"), "should have --repo flag")
}

func TestExperimentsAnalyzeSubcommandFlags(t *testing.T) {
	cmd := NewExperimentsAnalyzeSubcommand()
	require.NotNil(t, cmd, "analyze subcommand should be created")

	assert.NotNil(t, cmd.Flag("json"), "should have --json flag")
	assert.NotNil(t, cmd.Flag("repo"), "should have --repo flag")
}

func TestExperimentsAnalyzeRequiresArg(t *testing.T) {
	cmd := NewExperimentsAnalyzeSubcommand()
	require.NotNil(t, cmd, "analyze subcommand should be created")

	// ExactArgs(1) should reject zero args
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "analyze should require exactly 1 argument")
}
