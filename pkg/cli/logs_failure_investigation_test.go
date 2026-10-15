//go:build !integration

package cli

import (
	"slices"
	"strconv"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureSignalsTypedLegacyContext(t *testing.T) {
	t.Parallel()
	for _, names := range [][2]string{{"agent", "Start MCP Gateway"}, {`odd/"job`, "step\n\"quoted\""}} {
		value := failureClusterKey(names[0], names[1])
		signal := enrichFailureSignal(FailureSignal{Kind: "failed_step", Value: value})
		assert.Equal(t, names[0], signal.Job)
		assert.Equal(t, names[1], signal.Step)
		assert.Equal(t, value, signal.Value)
		old := []RunData{{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{{Kind: signal.Kind, Value: value}}}}
		typed := []RunData{{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{signal}}}
		assert.Equal(t, annotateFailureClusters(old), annotateFailureClusters(typed))
		assert.Equal(t, signal, old[0].FailureSignals[0])
	}
	for _, value := range []string{"legacy opaque value", `"job"/broken`, `"job"x"step"`} {
		signal := FailureSignal{Kind: "failed_step", Value: value}
		assert.Equal(t, signal, enrichFailureSignal(signal))
	}
}

func TestFailureClustersPrioritizeBoundaries(t *testing.T) {
	t.Parallel()
	aftermath := FailureSignal{Kind: "failed_step", Value: failureClusterKey("conclusion", constants.AgentFailureStepName)}
	runs := []RunData{
		{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{aftermath}},
		{RunID: 2, Conclusion: "failure", FailureSignals: []FailureSignal{aftermath}},
		{RunID: 3, Conclusion: "failure", FailureSignals: []FailureSignal{{Kind: "failed_step", Value: failureClusterKey("agent", "Start MCP Gateway")}}},
		{RunID: 4, Conclusion: "failure", FailureSignals: []FailureSignal{{Kind: "blocked_domain", Value: "example.test"}}},
		{RunID: 5, Conclusion: "failure"},
		{RunID: 6, Conclusion: "failure", IntentionalFailure: true, FailureSignals: []FailureSignal{{Kind: "failed_job", Value: "setup"}}},
	}
	clusters := annotateFailureClusters(runs)
	var roles []string
	for _, cluster := range clusters {
		roles = append(roles, cluster.SignalRole)
	}
	assert.Equal(t, []string{"failure_boundary", "context", "metadata_only", "aftermath", "failure_boundary"}, roles)
	plan := buildFailureInvestigation(runs, clusters)
	require.NotNil(t, plan)
	require.NotEmpty(t, plan.SelectedRuns)
	assert.Equal(t, int64(3), plan.SelectedRuns[0].RunID)
	for _, target := range plan.SelectedRuns {
		assert.NotEqual(t, int64(6), target.RunID)
	}
}

func TestFailureClustersRepresentativeDiversity(t *testing.T) {
	t.Parallel()
	var runs []RunData
	for i := int64(1); i <= 10; i++ {
		model := "a"
		if i <= 2 {
			model = strconv.FormatInt(i, 10)
		}
		runs = append(runs, RunData{RunID: i, Conclusion: "failure", WorkflowName: "test",
			EngineID: "copilot", Model: model, FailureSignals: []FailureSignal{{Kind: "mcp_failure", Value: "github"}}})
	}
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 1)
	var ids []int64
	var models []string
	for _, target := range clusters[0].RepresentativeRuns {
		ids = append(ids, target.RunID)
		models = append(models, target.Model)
		assert.Equal(t, "test", target.WorkflowName)
		assert.Equal(t, "copilot", target.Engine)
	}
	assert.Equal(t, []int64{10, 2, 1}, ids)
	assert.Len(t, slices.Compact(models), 3)
}

func TestFailureInvestigationBudgetCoverageAndDeterminism(t *testing.T) {
	t.Parallel()
	var runs []RunData
	for i := int64(1); i <= 7; i++ {
		runs = append(runs, RunData{RunID: i, Conclusion: "failure", FailureSignals: []FailureSignal{
			{Kind: "mcp_failure", Value: strconv.FormatInt(i, 10)},
			{Kind: "failed_step", Value: failureClusterKey("conclusion", constants.AgentFailureStepName)},
		}})
	}
	runs = append(runs,
		RunData{RunID: 8, Conclusion: "failure", IntentionalFailure: true, FailureSignals: []FailureSignal{{Kind: "mcp_failure", Value: "intentional"}}},
		RunData{RunID: 9, Conclusion: "failure"},
		RunData{RunID: 10, Conclusion: "success"})
	clusters := annotateFailureClusters(runs)
	plan := buildFailureInvestigation(runs, clusters)
	require.NotNil(t, plan)
	assert.Equal(t, 9, plan.UniqueFailedRuns)
	assert.Equal(t, 8, plan.ObservedEvidenceRuns)
	assert.Equal(t, 1, plan.MetadataOnlyRuns)
	assert.Equal(t, 1, plan.IntentionalFailureRuns)
	assert.Len(t, plan.SelectedRuns, 5)
	var selected []int64
	for _, target := range plan.SelectedRuns {
		selected = append(selected, target.RunID)
		assert.Len(t, target.ClusterIDs, 2)
	}
	assert.Equal(t, []int64{7, 6, 5, 4, 3}, selected)
	assert.Equal(t, 9, plan.EligibleClusters)
	assert.Equal(t, 6, plan.CoveredClusters)
	assert.Equal(t, 3, plan.RemainingClusters)
	assert.Equal(t, 7, plan.CoveredRuns, "shared aftermath reaches seven runs, not a sum of overlapping memberships")
	slices.Reverse(runs)
	assert.Equal(t, plan, buildFailureInvestigation(runs, annotateFailureClusters(runs)))
}

func TestFailureInvestigationUnknownIntentionalAndDuplicateRuns(t *testing.T) {
	t.Parallel()
	assert.Nil(t, buildFailureInvestigation(nil, annotateFailureClusters(nil)))
	runs := []RunData{{RunID: 1, Conclusion: "failure"}, {RunID: 2, Conclusion: "failure"}, {RunID: 2, Conclusion: "failure"}}
	plan := buildFailureInvestigation(runs, annotateFailureClusters(runs))
	require.NotNil(t, plan)
	assert.Equal(t, 2, plan.UniqueFailedRuns)
	assert.Equal(t, 2, plan.MetadataOnlyRuns)
	assert.Len(t, plan.SelectedRuns, 1)
	assert.Equal(t, int64(2), plan.SelectedRuns[0].RunID)
	assert.Equal(t, 2, plan.CoveredRuns)
	for i := range runs {
		runs[i].IntentionalFailure = true
	}
	plan = buildFailureInvestigation(runs, annotateFailureClusters(runs))
	assert.Equal(t, 2, plan.IntentionalFailureRuns)
	assert.Zero(t, plan.EligibleClusters)
	assert.Empty(t, plan.SelectedRuns)
}

func TestFailureInvestigationOverlapSelectsRichestRunOnce(t *testing.T) {
	t.Parallel()
	data := failureClusterTestLogsData()
	require.NotNil(t, data.FailureInvestigation)
	require.Len(t, data.FailureInvestigation.SelectedRuns, 1)
	assert.Equal(t, int64(2), data.FailureInvestigation.SelectedRuns[0].RunID)
	assert.Len(t, data.FailureInvestigation.SelectedRuns[0].ClusterIDs, 2)
	assert.Equal(t, 2, data.FailureInvestigation.CoveredClusters)
	assert.Zero(t, data.FailureInvestigation.RemainingClusters)
}

func TestFailureInvestigationLogsPromptGuidance(t *testing.T) {
	t.Parallel()
	for _, required := range []string{
		"failure_investigation.selected_runs", "failure_clusters", "representative_runs", "failure_boundary",
		"aftermath", "signal.job/step", "unique run IDs", "symptoms", "intentional_failure",
		"metadata_only", "jq", `"partial": true`, "end_date", "before_run_id", "audit_path/logs_path", "token_usage", "aic",
		"Read the returned file_path first",
	} {
		assert.Contains(t, logsToolDescription, required)
	}
}

func TestFailureInvestigationUsesHostQualifiedRunIdentities(t *testing.T) {
	t.Parallel()
	signal := FailureSignal{Kind: "mcp_failure", Value: "github"}
	runs := []RunData{
		{RunID: 42, Repository: "owner/repo", URL: "https://github.com/owner/repo/actions/runs/42", Conclusion: "failure", FailureSignals: []FailureSignal{signal}},
		{RunID: 42, Repository: "owner/repo", URL: "https://ghe.example/owner/repo/actions/runs/42", Conclusion: "failure", FailureSignals: []FailureSignal{signal}},
	}
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 1)
	assert.Equal(t, 2, clusters[0].Count)
	assert.Equal(t, []int64{42, 42}, clusters[0].RunIDs)
	assert.Equal(t, []FailureRunIdentity{
		{Host: "ghe.example", Repository: "owner/repo", RunID: 42},
		{Host: "github.com", Repository: "owner/repo", RunID: 42},
	}, clusters[0].RunIdentities)
	require.Len(t, clusters[0].RepresentativeRuns, 2)

	plan := buildFailureInvestigation(runs, clusters)
	require.NotNil(t, plan)
	assert.Equal(t, 2, plan.UniqueFailedRuns)
	assert.Equal(t, 2, plan.CoveredRuns)
	require.Len(t, plan.SelectedRuns, 1)
	assert.Equal(t, []string{clusters[0].ID}, plan.SelectedRuns[0].ClusterIDs)

	metadataRuns := []RunData{
		{RunID: 42, Repository: "owner/repo", URL: "https://github.com/owner/repo/actions/runs/42", Conclusion: "failure"},
		{RunID: 42, Repository: "owner/repo", URL: "https://ghe.example/owner/repo/actions/runs/42", Conclusion: "failure"},
		{RunID: 42, Repository: "owner/repo", URL: "https://github.com/owner/repo/actions/runs/42", Conclusion: "failure"},
	}
	metadataClusters := annotateFailureClusters(metadataRuns)
	require.Len(t, metadataClusters, 2)
	metadataPlan := buildFailureInvestigation(metadataRuns, metadataClusters)
	require.NotNil(t, metadataPlan)
	assert.Equal(t, 2, metadataPlan.UniqueFailedRuns)
	assert.Equal(t, 2, metadataPlan.MetadataOnlyRuns)
	assert.Equal(t, 2, metadataPlan.CoveredRuns)
	require.Len(t, metadataPlan.SelectedRuns, 2)
}
