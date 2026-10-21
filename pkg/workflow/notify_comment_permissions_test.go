package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConclusionJobDurationPermissions(t *testing.T) {
	for _, failedJobsEnabled := range []string{"false", "true"} {
		for _, failureIssueEnabled := range []string{"false", "true"} {
			t.Run("failed_jobs_"+failedJobsEnabled+"_failure_issue_"+failureIssueEnabled, func(t *testing.T) {
				failedJobs := TemplatableBool(failedJobsEnabled)
				failureIssue := TemplatableBool(failureIssueEnabled)
				data := &WorkflowData{
					SafeOutputs: &SafeOutputsConfig{
						ReportFailedJobs:     &failedJobs,
						ReportFailureAsIssue: &failureIssue,
					},
				}

				perms := computeConclusionJobPermissions(data)
				actions, ok := perms.Get(PermissionActions)
				require.True(t, ok)
				require.Equal(t, PermissionRead, actions, "duration collection is independent of failure reporting")
				_, pullRequests := perms.Get(PermissionPullRequests)
				require.Equal(t, failureIssueEnabled == "true", pullRequests, "duration collection must not broaden PR access")

				data.SafeOutputs.DispatchWorkflow = &DispatchWorkflowConfig{}
				actions, ok = computeConclusionJobPermissions(data).Get(PermissionActions)
				require.True(t, ok)
				require.Equal(t, PermissionWrite, actions, "duration collection must preserve existing Actions write access")
			})
		}
	}
}

func TestConclusionJobOIDCIsLimitedToConfiguredOTLP(t *testing.T) {
	for _, worker := range []bool{false, true} {
		data := &WorkflowData{SafeOutputs: &SafeOutputsConfig{}}
		if worker {
			data.Tools = map[string]any{"work-queue": map[string]any{"worker": true}}
		}
		_, ok := computeConclusionJobPermissions(data).Get(PermissionIdToken)
		require.False(t, ok, "admission callback OIDC must not be granted to unrelated conclusion steps")

		data.RawFrontmatter = map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"github-app": map[string]any{"audience": "https://otel.example.com"},
				},
			},
		}
		level, ok := computeConclusionJobPermissions(data).Get(PermissionIdToken)
		require.True(t, ok)
		require.Equal(t, PermissionWrite, level)
	}
}
