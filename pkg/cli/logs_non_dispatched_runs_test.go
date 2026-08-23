//go:build !integration

package cli

import "testing"

// TestIsNonDispatchedConclusion verifies that conclusions for runs which never
// dispatched a job are recognised, so they are excluded from executed-run metrics.
func TestIsNonDispatchedConclusion(t *testing.T) {
	tests := []struct {
		conclusion string
		want       bool
	}{
		{conclusion: "skipped", want: true},
		{conclusion: "action_required", want: true},
		{conclusion: "success", want: false},
		{conclusion: "failure", want: false},
		{conclusion: "cancelled", want: false},
		{conclusion: "timed_out", want: false},
		{conclusion: "neutral", want: false},
		{conclusion: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.conclusion, func(t *testing.T) {
			if got := isNonDispatchedConclusion(tt.conclusion); got != tt.want {
				t.Errorf("isNonDispatchedConclusion(%q) = %v, want %v", tt.conclusion, got, tt.want)
			}
		})
	}
}

// TestIsCompletedNonSkippedRunExcludesNonDispatched verifies that runs held for
// manual approval (action_required) are not treated as executed runs. Command
// workflows such as `q` accumulate large numbers of these runs when a bot actor
// comments on an issue, which previously dragged their reported success rate
// down to near zero.
func TestIsCompletedNonSkippedRunExcludesNonDispatched(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       bool
	}{
		{name: "completed success", status: "completed", conclusion: "success", want: true},
		{name: "completed failure", status: "completed", conclusion: "failure", want: true},
		{name: "completed skipped", status: "completed", conclusion: "skipped", want: false},
		{name: "completed action_required", status: "completed", conclusion: "action_required", want: false},
		{name: "in progress", status: "in_progress", conclusion: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := WorkflowRun{Status: tt.status, Conclusion: tt.conclusion}
			if got := isCompletedNonSkippedRun(run); got != tt.want {
				t.Errorf("isCompletedNonSkippedRun(%+v) = %v, want %v", run, got, tt.want)
			}
		})
	}
}

// TestCalculateWorkflowHealthOnDispatchedRuns documents the reporting impact of
// the fix: once non-dispatched runs are filtered out of the run set, the success
// rate reflects only runs that actually executed the agent.
func TestCalculateWorkflowHealthOnDispatchedRuns(t *testing.T) {
	all := []WorkflowRun{
		{Status: "completed", Conclusion: "success"},
		{Status: "completed", Conclusion: "failure"},
	}
	for i := 0; i < 20; i++ {
		all = append(all, WorkflowRun{Status: "completed", Conclusion: "action_required"})
	}

	dispatched := make([]WorkflowRun, 0, len(all))
	for _, run := range all {
		if isNonDispatchedConclusion(run.Conclusion) {
			continue
		}
		dispatched = append(dispatched, run)
	}

	health := CalculateWorkflowHealth("q", dispatched, 50)
	if health.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", health.TotalRuns)
	}
	if health.SuccessRate != 50 {
		t.Errorf("SuccessRate = %v, want 50", health.SuccessRate)
	}
}
