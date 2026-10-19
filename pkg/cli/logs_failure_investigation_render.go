package cli

import (
	"io"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

func renderFailureInvestigationToWriter(w io.Writer, plan *FailureInvestigation, prefix string) error {
	if plan == nil {
		return nil
	}
	out := &compactLogsWriter{writer: w}
	out.printf("%s[failure_investigation] unique_failed=%d observed=%d metadata_only=%d non_execution=%d intentional=%d selected=%d/%d clusters=%d/%d remaining=%d symptom_runs=%d\n",
		prefix, plan.UniqueFailedRuns, plan.ObservedEvidenceRuns, plan.MetadataOnlyRuns, plan.NonExecutionRuns, plan.IntentionalFailureRuns,
		len(plan.SelectedRuns), plan.AuditBudget, plan.CoveredClusters, plan.EligibleClusters, plan.RemainingClusters, plan.CoveredRuns)
	out.printf("%sShared symptoms are not confirmed causes. Intentional failures are excluded from recommendations. Full membership: --json; text membership: -v.\n", prefix)
	if plan.NonExecutionRuns > 0 {
		out.printf("%sConfirmed jobless failures retain GitHub's conclusion but have no runtime to audit. Inspect GitHub check-suite/validation and PR lifecycle evidence; no_jobs alone does not establish a cause.\n", prefix)
	}
	out.printf("%sCluster shares describe this failure sample, not fleet health. Wilson 95%% bounds assume independent runs; retries and missing evidence limit interpretation.\n", prefix)
	for _, target := range plan.SelectedRuns {
		out.printf("%s  investigate=%d workflow=%q engine=%q model=%q clusters=%s audit=%q logs=%q command=%q\n",
			prefix, target.RunID, target.WorkflowName, target.Engine, target.Model,
			strings.Join(target.ClusterIDs, ","), target.AuditPath, target.LogsPath, failureClusterAuditCommand(target.FailureClusterTarget))
	}
	return out.err
}

func renderMarkdownFailureInvestigationToWriter(w io.Writer, plan *FailureInvestigation) {
	if plan == nil {
		return
	}
	out := &compactLogsWriter{writer: w}
	out.println("## Suggested Failure Investigations")
	out.println()
	out.printf("Unique failed runs: **%d** (%d observed, %d metadata-only, %d confirmed jobless; %d intentional). "+
		"Inspect these %d runs first; they span %d/%d eligible symptom clusters. "+
		"%d clusters remain. Shared symptoms do not prove a common cause.\n\n",
		plan.UniqueFailedRuns, plan.ObservedEvidenceRuns, plan.MetadataOnlyRuns, plan.NonExecutionRuns, plan.IntentionalFailureRuns,
		len(plan.SelectedRuns), plan.CoveredClusters, plan.EligibleClusters, plan.RemainingClusters)
	if plan.NonExecutionRuns > 0 {
		out.println("Confirmed jobless failures retain GitHub's conclusion but are excluded from runtime audit recommendations. Inspect GitHub check-suite/validation and PR lifecycle evidence; absence of jobs does not establish a cause.")
		out.println()
	}
	out.println("| Run | Workflow | Engine / Model | Clusters | Cached audit | Audit command |")
	out.println("|---|---|---|---|---|---|")
	for _, target := range plan.SelectedRuns {
		out.printf("| %d | %s | %s / %s | %s | %s | %s |\n",
			target.RunID, modelRoutingMarkdownCell(target.WorkflowName), modelRoutingMarkdownCell(target.Engine),
			modelRoutingMarkdownCell(target.Model), strings.Join(target.ClusterIDs, ", "),
			modelRoutingMarkdownCell(target.AuditPath),
			modelRoutingMarkdownCell(failureClusterAuditCommand(target.FailureClusterTarget)))
	}
	out.println()
	if out.err != nil {
		console.PrintErrorMessage("Cannot render failure investigation: " + out.err.Error())
	}
}
