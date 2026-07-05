package cli

import (
	"fmt"
	"math"

	"github.com/github/gh-aw/pkg/logger"
)

// maxSafeFloat64Int is the largest integer that can be represented exactly as
// float64 (2^53). GitHub run IDs decoded from JSON arrive as float64; any value
// beyond this threshold cannot be round-tripped to int64 without precision loss.
const maxSafeFloat64Int = 1 << 53

var outcomeEvalWorkflowLog = logger.New("cli:outcome_eval_workflow")
var workflowOutcomeGHAPIGet = ghAPIGet

// evalDispatchWorkflow checks whether a dispatched workflow run completed successfully.
// It looks for a run_id in the item metadata and queries the workflow run status.
// Spec: specs/safe-output-outcome-evaluation.md §20
func evalDispatchWorkflow(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	outcomeEvalWorkflowLog.Printf("Evaluating dispatch_workflow: repo=%s, url=%s", repo, item.URL)

	report := OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      repo,
	}

	// Extract run_id from metadata if available.
	// JSON numbers unmarshal as float64; convert carefully to int64 to avoid
	// precision loss for large GitHub run IDs (which can exceed 2^32).
	var runID int64
	if item.Metadata != nil {
		if v, ok := item.Metadata["run_id"]; ok {
			switch id := v.(type) {
			case float64:
				// Guard against float64 values that cannot round-trip to int64.
				// math.MaxInt64 is not representable exactly as float64; use 2^53
				// (maxSafeFloat64Int) as the safe upper bound for lossless conversion.
				if id > 0 && id <= maxSafeFloat64Int && id == math.Trunc(id) {
					runID = int64(id)
				}
			case int:
				runID = int64(id)
			case int64:
				runID = id
			}
		}
	}

	if runID <= 0 {
		// No run ID available — workflow may not have been dispatched or ID not captured
		report.Result = OutcomePending
		report.Detail = "no run ID available; dispatch may still be queued"
		return report
	}

	data, err := workflowOutcomeGHAPIGet(fmt.Sprintf("actions/runs/%d", runID), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	status, _ := data["status"].(string)
	conclusion, _ := data["conclusion"].(string)
	outcomeEvalWorkflowLog.Printf("dispatch_workflow run %d: status=%s, conclusion=%s", runID, status, conclusion)

	switch {
	case status == "completed" && conclusion == "success":
		report.Result = OutcomeAccepted
		report.Detail = "workflow run completed with success"
	case status == "completed" && (conclusion == "failure" || conclusion == "timed_out" || conclusion == "cancelled" || conclusion == "action_required"):
		// action_required means the run is blocked and requires manual intervention;
		// treat it as rejected rather than ignored since it does not self-resolve.
		report.Result = OutcomeRejected
		report.Detail = "workflow run completed with " + conclusion
	case status == "completed":
		// neutral and skipped indicate the run did not contribute meaningful output.
		report.Result = OutcomeIgnored
		report.Detail = "workflow run completed with " + conclusion
	default:
		report.Result = OutcomePending
		report.Detail = "workflow run status: " + status
	}
	return report
}

// evalUpdateDiscussion checks whether a discussion edit stuck.
// Full evaluation requires GraphQL (same pattern as evalCloseDiscussion).
// Until the GraphQL evaluator is implemented this returns OutcomeIgnored so that
// callers do not retry indefinitely.
// Spec: specs/safe-output-outcome-evaluation.md §12
func evalUpdateDiscussion(item CreatedItemReport, repoOverride string) OutcomeReport {
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomeIgnored,
		Detail:    "discussion update check requires GraphQL (not yet implemented); outcome is advisory only",
	}
}
