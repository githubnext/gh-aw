//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

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

func TestParseExperimentState(t *testing.T) {
	tests := []struct {
		name            string
		input           []byte
		wantExperiments int
		wantTotalRuns   int
		wantLastRun     string
	}{
		{
			name: "valid state with runs",
			input: []byte(`{
				"counts": {"feature": {"A": 3, "B": 2}},
				"runs": [
					{"run_id": "1", "timestamp": "2024-06-01T10:00:00Z", "assignments": {"feature": "A"}},
					{"run_id": "2", "timestamp": "2024-06-15T12:00:00Z", "assignments": {"feature": "B"}}
				]
			}`),
			wantExperiments: 1,
			wantTotalRuns:   2,
			wantLastRun:     "2024-06-15",
		},
		{
			name: "valid state without runs array",
			input: []byte(`{
				"counts": {"exp1": {"yes": 5, "no": 5}, "exp2": {"on": 3, "off": 7}}
			}`),
			wantExperiments: 2,
			wantTotalRuns:   20,
			wantLastRun:     "",
		},
		{
			name:            "empty JSON object",
			input:           []byte(`{}`),
			wantExperiments: 0,
			wantTotalRuns:   0,
			wantLastRun:     "",
		},
		{
			name:            "invalid JSON returns empty state",
			input:           []byte(`not json`),
			wantExperiments: 0,
			wantTotalRuns:   0,
			wantLastRun:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := parseExperimentState(tt.input)
			require.NotNil(t, state, "state should never be nil")
			assert.Len(t, state.Counts, tt.wantExperiments, "experiment count")
			assert.Equal(t, tt.wantTotalRuns, experimentTotalRuns(state), "total runs")
			assert.Equal(t, tt.wantLastRun, experimentLastRun(state), "last run date")
		})
	}
}

func TestExperimentDetailsFromState(t *testing.T) {
	state := &ExperimentState{
		Counts: map[string]map[string]int{
			"style":   {"concise": 4, "detailed": 6},
			"feature": {"on": 5, "off": 5},
		},
		Runs: []ExperimentRunRecord{
			{RunID: "r1", Timestamp: "2024-05-01T00:00:00Z", Assignments: map[string]string{"style": "concise", "feature": "on"}},
			{RunID: "r2", Timestamp: "2024-05-02T00:00:00Z", Assignments: map[string]string{"style": "detailed", "feature": "off"}},
		},
	}

	details := experimentDetailsFromState("my-workflow", "experiments/my-workflow", state)
	require.NotNil(t, details, "details should not be nil")
	assert.Equal(t, "my-workflow", details.WorkflowID, "workflow ID")
	assert.Equal(t, "experiments/my-workflow", details.Branch, "branch")
	assert.Equal(t, 2, details.TotalRuns, "total runs from runs array")
	assert.Len(t, details.Experiments, 2, "should have 2 experiment entries")
	assert.Len(t, details.RecentRuns, 2, "should have 2 recent runs")

	// Experiments are sorted by name.
	assert.Equal(t, "feature", details.Experiments[0].Name, "first experiment sorted by name")
	assert.Equal(t, 10, details.Experiments[0].Total, "feature total")
	assert.Equal(t, "style", details.Experiments[1].Name, "second experiment sorted by name")
	assert.Equal(t, 10, details.Experiments[1].Total, "style total")
}

func TestExperimentTotalRunsFallback(t *testing.T) {
	// When no runs array present, sum variant counts.
	state := &ExperimentState{
		Counts: map[string]map[string]int{
			"exp": {"A": 3, "B": 4},
		},
	}
	assert.Equal(t, 7, experimentTotalRuns(state), "total from counts fallback")
}

func TestFormatAssignments(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "nil map returns dash",
			input:    nil,
			expected: "-",
		},
		{
			name:     "empty map returns dash",
			input:    map[string]string{},
			expected: "-",
		},
		{
			name:     "single entry",
			input:    map[string]string{"style": "concise"},
			expected: "style=concise",
		},
		{
			name:     "multiple entries sorted by key",
			input:    map[string]string{"z": "last", "a": "first"},
			expected: "a=first, z=last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatAssignments(tt.input), "formatAssignments(%v)", tt.input)
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
			WorkflowID:  "my-workflow",
			Branch:      "experiments/my-workflow",
			Experiments: 2,
			TotalRuns:   15,
			LastRun:     "2024-06-15",
		},
	}

	jsonBytes, err := json.MarshalIndent(experiments, "", "  ")
	require.NoError(t, err, "should marshal ExperimentInfo to JSON")

	var result []map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &result), "should unmarshal JSON back")

	require.Len(t, result, 1, "should have 1 experiment")
	assert.Equal(t, "my-workflow", result[0]["workflow_id"], "workflow_id field should match")
	assert.Equal(t, "experiments/my-workflow", result[0]["branch"], "branch field should match")
	assert.EqualValues(t, 2, result[0]["experiments"], "experiments count should match")
	assert.EqualValues(t, 15, result[0]["total_runs"], "total_runs should match")
	assert.Equal(t, "2024-06-15", result[0]["last_run"], "last_run should match")
}

func TestExperimentDetailsJSONOutput(t *testing.T) {
	details := ExperimentDetails{
		WorkflowID: "my-workflow",
		Branch:     "experiments/my-workflow",
		TotalRuns:  10,
		Experiments: []ExperimentVariantStats{
			{
				Name:     "style",
				Variants: map[string]int{"concise": 6, "detailed": 4},
				Total:    10,
			},
		},
		RecentRuns: []ExperimentRunRecord{
			{RunID: "123", Timestamp: "2024-06-01T00:00:00Z", Assignments: map[string]string{"style": "concise"}},
		},
	}

	jsonBytes, err := json.MarshalIndent(details, "", "  ")
	require.NoError(t, err, "should marshal ExperimentDetails to JSON")

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &result), "should unmarshal JSON back")

	assert.Equal(t, "my-workflow", result["workflow_id"], "workflow_id should match")
	assert.EqualValues(t, 10, result["total_runs"], "total_runs should match")

	experiments, ok := result["experiments"].([]any)
	require.True(t, ok, "experiments should be an array")
	require.Len(t, experiments, 1, "should have 1 experiment")

	recentRuns, ok := result["recent_runs"].([]any)
	require.True(t, ok, "recent_runs should be an array")
	require.Len(t, recentRuns, 1, "should have 1 recent run")
}

func TestNewExperimentsCommand(t *testing.T) {
	cmd := NewExperimentsCommand()
	require.NotNil(t, cmd, "command should be created")
	assert.Equal(t, "experiments", cmd.Name(), "command name should be experiments")
	assert.True(t, cmd.Hidden, "experiments command should be hidden")

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

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "analyze should require exactly 1 argument")
}
