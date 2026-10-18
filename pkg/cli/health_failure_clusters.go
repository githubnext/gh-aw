package cli

import (
	"path/filepath"
	"strconv"

	"github.com/github/gh-aw/pkg/workflow"
)

// Reuse logs clustering without downloading artifacts or retaining full cached analyses.
func buildHealthFailureClusters(runs []WorkflowRun, config HealthConfig) ([]FailureCluster, *FailureInvestigation) {
	runs = currentHealthRuns(runs, healthRevisionStatuses(runs, config.collection))
	var failures []RunData
	intentional := make(map[string]bool)
	for _, run := range runs {
		if !isClusterFailureConclusion(run.Conclusion) {
			continue
		}
		data := RunData{
			RunID: run.DatabaseID, Repository: run.Repository, URL: run.URL,
			WorkflowName: run.WorkflowName, WorkflowPath: run.WorkflowPath,
			Conclusion: run.Conclusion, CreatedAt: run.CreatedAt,
		}
		if config.RepoOverride == "" {
			value, known := intentional[run.WorkflowPath]
			if !known {
				value = workflow.IsIntentionalFailure(run.WorkflowPath)
				intentional[run.WorkflowPath] = value
			}
			data.IntentionalFailure = value
		}
		enrichHealthFailureFromCache(&data, run, config)
		failures = append(failures, data)
	}
	clusters := annotateFailureClusters(failures)
	return clusters, buildFailureInvestigation(failures, clusters)
}

func enrichHealthFailureFromCache(data *RunData, run WorkflowRun, config HealthConfig) {
	summary, dir := loadHealthRunSummary(run, config)
	if summary == nil {
		return
	}
	pr := ProcessedRun{
		Run: summary.Run, JobDetails: summary.JobDetails, MCPFailures: summary.MCPFailures,
		MissingTools: summary.MissingTools, MissingData: summary.MissingData, FirewallAnalysis: summary.FirewallAnalysis,
	}
	var agg logsAggregate
	data.FailureKind = agg.classifyRunFailure(pr)
	data.FailureSignals = failureSignalsForRun(pr)
	data.LogsPath = dir
}

func loadHealthRunSummary(run WorkflowRun, config HealthConfig) (*RunSummary, string) {
	root := config.LogsDir
	if root == "" {
		root = defaultLogsOutputDir
	}
	dir := filepath.Join(root, "run-"+strconv.FormatInt(run.DatabaseID, 10))
	summary, ok := loadRunSummary(dir, config.Verbose)
	if !ok {
		return nil, dir // Cache loader logs missing or invalid evidence.
	}
	cached := summary.Run
	if summary.RunID != run.DatabaseID || cached.DatabaseID != run.DatabaseID ||
		run.URL == "" || cached.URL != run.URL || cached.Status != run.Status || cached.Conclusion != run.Conclusion ||
		cached.HeadSha != run.HeadSha || cached.WorkflowPath != run.WorkflowPath ||
		cached.Repository != run.Repository || cached.Attempt != run.Attempt || !cached.UpdatedAt.Equal(run.UpdatedAt) {
		healthLog.Printf("Ignoring stale health evidence for run %d", run.DatabaseID)
		return nil, dir
	}
	return summary, dir
}
