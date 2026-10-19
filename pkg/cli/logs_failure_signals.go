package cli

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// Enrich legacy cached signals without changing their canonical value or ID.
func enrichFailureSignal(signal FailureSignal) FailureSignal {
	if signal.Kind == "failed_job" {
		signal.Job = signal.Value
	}
	if signal.Kind != "failed_step" {
		return signal
	}
	job, err := strconv.QuotedPrefix(signal.Value)
	if err != nil {
		return signal // Older or custom opaque values remain usable without typed context.
	}
	step, ok := strings.CutPrefix(signal.Value, job+"/")
	if !ok {
		return signal
	}
	jobName, jobErr := strconv.Unquote(job)
	stepName, stepErr := strconv.Unquote(step)
	if jobErr == nil && stepErr == nil {
		signal.Job, signal.Step = jobName, stepName
	}
	return signal
}

func failureSignalRole(signal FailureSignal) string {
	if signal.Job == "conclusion" && signal.Step == constants.AgentFailureStepName {
		return "aftermath"
	}
	switch signal.Kind {
	case "failed_job", "failed_step", "mcp_failure":
		return "failure_boundary"
	case "workflow_failure", "no_jobs":
		return "metadata_only"
	default:
		return "context"
	}
}

func failureSignalRoleRank(role string) int {
	switch role {
	case "failure_boundary":
		return 0
	case "context":
		return 1
	case "metadata_only":
		return 2
	default:
		return 3
	}
}

func failureTarget(run *RunData) FailureClusterTarget {
	return FailureClusterTarget{
		RunID: run.RunID, Repository: run.Repository,
		WorkflowName: run.WorkflowName, WorkflowPath: run.WorkflowPath,
		Engine: run.EngineID, Model: run.Model, CreatedAt: run.CreatedAt,
		URL: run.URL, LogsPath: run.LogsPath, AuditPath: run.AuditPath,
	}
}

// Keep the richest run first, then prefer distinct workflow/engine/model contexts.
func selectFailureRepresentatives(members []*RunData) []FailureClusterTarget {
	selected := make([]FailureClusterTarget, 0, 3)
	contexts := make(map[string]struct{})
	identities := make(map[string]struct{})
	for _, diverse := range []bool{true, false} {
		for _, run := range members {
			identity := failureRunIdentityFor(*run)
			scope := failureClusterKey(identity.Host, identity.Repository, workflowIDFromRun(run.WorkflowPath, run.WorkflowName), run.EngineID, run.Model)
			_, picked := identities[identity.key()]
			_, represented := contexts[scope]
			if picked || (diverse && represented) {
				continue
			}
			selected = append(selected, failureTarget(run))
			contexts[scope], identities[identity.key()] = struct{}{}, struct{}{}
			if len(selected) == 3 {
				return selected
			}
		}
	}
	return selected
}
