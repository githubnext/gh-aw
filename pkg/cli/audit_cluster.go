package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AuditClusterReport is a lightweight, deterministic payload for cross-run triage.
// Signatures identify shared observations, not inferred root causes.
type AuditClusterReport struct {
	Schema       string                    `json:"schema"`
	Complete     bool                      `json:"complete"`
	RunsExamined int                       `json:"runs_examined"`
	RunsSelected int                       `json:"runs_selected"`
	Outcomes     map[string]int            `json:"outcomes"`
	StalledAfter string                    `json:"stalled_after"`
	AsOf         time.Time                 `json:"as_of"`
	StartDate    string                    `json:"start_date,omitempty"`
	EndDate      string                    `json:"end_date,omitempty"`
	FailedNoJobs bool                      `json:"failed_no_jobs"`
	Runs         []AuditClusterRun         `json:"runs"`
	Groups       []AuditClusterGroup       `json:"groups"`
	Continuation *AuditClusterContinuation `json:"continuation,omitempty"`
	Errors       []string                  `json:"errors,omitempty"`
}

type AuditClusterRun struct {
	RunID                int64     `json:"run_id"`
	Repository           string    `json:"repository"`
	Hostname             string    `json:"hostname"`
	URL                  string    `json:"url,omitempty"`
	Workflow             string    `json:"workflow,omitempty"`
	WorkflowPath         string    `json:"workflow_path,omitempty"`
	HeadSHA              string    `json:"head_sha,omitempty"`
	Event                string    `json:"event,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitzero"`
	HeadBranch           string    `json:"head_branch,omitempty"`
	Status               string    `json:"status,omitempty"`
	Conclusion           string    `json:"conclusion,omitempty"`
	Outcome              string    `json:"outcome"`
	UpdatedAt            time.Time `json:"updated_at,omitzero"`
	JobsState            string    `json:"jobs_state"`
	JobCount             *int      `json:"job_count"`
	UsageState           string    `json:"usage_state"`
	Tokens               *int      `json:"token_usage"`
	AIC                  *float64  `json:"aic"`
	Turns                *int      `json:"turns"`
	FailedToolExecutions *int      `json:"failed_tool_executions"`
	Signatures           []string  `json:"signatures"`
	Evidence             []string  `json:"evidence,omitempty"`
}

type AuditClusterGroup struct {
	Outcome           string          `json:"outcome"`
	Signature         string          `json:"signature"`
	Count             int             `json:"count"`
	RepresentativeRun AuditClusterRun `json:"representative_run"`
	RunIDs            []int64         `json:"run_ids"`
	RunURLs           []string        `json:"run_urls"`
}

// Arguments can be passed directly to `gh aw audit` to resume collection.
type AuditClusterContinuation struct {
	Arguments             []string `json:"arguments,omitempty"`
	Reason                string   `json:"reason"`
	RequiresNarrowerRange bool     `json:"requires_narrower_range,omitempty"`
}

func validateAuditClusterOptions(cmd *cobra.Command, opts *auditCommandOptions) error {
	opts.cluster = opts.cluster || opts.startDate != "" || opts.endDate != ""
	if opts.diff {
		if opts.cluster {
			return errors.New("--diff cannot be combined with --cluster or clustering date ranges")
		}
		if opts.group {
			return errors.New("--diff cannot be combined with --group")
		}
		for _, flag := range []string{"count", "cluster-offset", "failed-no-jobs", "stalled-after"} {
			if cmd.Flags().Changed(flag) {
				return fmt.Errorf("--%s is not supported with --diff", flag)
			}
		}
		return nil
	}
	if !opts.cluster {
		if opts.stdin {
			return nil
		}
		for _, flag := range []string{"count", "cluster-offset", "failed-no-jobs", "stalled-after"} {
			if cmd.Flags().Changed(flag) {
				return fmt.Errorf("--%s requires --cluster or a date range", flag)
			}
		}
		return nil
	}
	for _, flag := range []string{"group", "parse", "experiment", "variant", "runtime", "evals", "no-baseline", "artifacts", "format"} {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("--%s is not supported with clustering; clustering collects usage artifacts and job metadata only", flag)
		}
	}
	if opts.stdin && (opts.startDate != "" || opts.endDate != "" || opts.clusterOffset != 0) {
		return errors.New("--stdin cannot be combined with a clustering date range or --cluster-offset")
	}
	if opts.count < 1 || opts.count > 1000 {
		return errors.New("--count must be between 1 and 1000")
	}
	if opts.clusterOffset < 0 {
		return errors.New("--cluster-offset must not be negative")
	}
	if opts.clusterOffset >= 1000 {
		return errors.New("--cluster-offset must be less than 1000; narrow the date range to stay within GitHub's search limit")
	}
	if opts.stalledAfter <= 0 {
		return errors.New("--stalled-after must be greater than zero")
	}
	_, _, err := auditClusterDateRange(*opts, time.Now())
	return err
}

func auditClusterDateRange(opts auditCommandOptions, now time.Time) (time.Time, time.Time, error) {
	end := now.UTC()
	startDate, endDate, err := resolveLogsDateRange(opts.startDate, opts.endDate, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if endDate != "" {
		end, err = parseFilterDate(endDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --end-date: %w", err)
		}
	}
	start := end.Add(-24 * time.Hour)
	if startDate != "" {
		start, err = parseFilterDate(startDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --start-date: %w", err)
		}
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, errors.New("--start-date must be earlier than --end-date")
	}
	return start.UTC(), end.UTC(), nil
}

func runAuditCluster(ctx context.Context, cmd *cobra.Command, args []string, opts auditCommandOptions) error {
	report, err := collectAuditClusters(ctx, args, opts, time.Now().UTC())
	if opts.jsonOutput {
		if encodeErr := json.NewEncoder(cmd.OutOrStdout()).Encode(report); encodeErr != nil {
			return errors.Join(err, encodeErr)
		}
		return err
	}
	text, renderErr := renderAuditClusterText(report)
	if renderErr != nil {
		return errors.Join(err, renderErr)
	}
	_, writeErr := fmt.Fprintln(cmd.OutOrStdout(), text)
	return errors.Join(err, writeErr)
}

func renderAuditClusterText(report AuditClusterReport) (string, error) {
	lines := []string{fmt.Sprintf("Clustered %d selected runs (%d examined); complete: %t", report.RunsSelected, report.RunsExamined, report.Complete)}
	for _, group := range report.Groups {
		lines = append(lines, fmt.Sprintf("%s | %q | %d runs | representative: %q",
			group.Outcome, group.Signature, group.Count, group.RepresentativeRun.URL))
	}
	for _, warning := range report.Errors {
		lines = append(lines, fmt.Sprintf("Unknown evidence: %q", warning))
	}
	if report.Continuation != nil {
		continuation, err := json.Marshal(report.Continuation)
		if err != nil {
			return "", fmt.Errorf("failed to encode clustering continuation: %w", err)
		}
		lines = append(lines, fmt.Sprintf("Continuation: %s", continuation))
	}
	return strings.Join(lines, "\n"), nil
}

func auditClusterOutcome(run WorkflowRun, now time.Time, threshold time.Duration) string {
	if run.Status == "completed" {
		switch run.Conclusion {
		case "success", "failure", "cancelled", "skipped", "neutral", "timed_out", "action_required", "startup_failure", "stale":
			return run.Conclusion
		default:
			return "unknown"
		}
	}
	switch run.Status {
	case "queued", "in_progress", "waiting", "pending", "requested":
		if !run.UpdatedAt.IsZero() && now.Sub(run.UpdatedAt) >= threshold {
			return "stalled"
		}
		return "active"
	default:
		return "unknown"
	}
}

func groupAuditClusterRuns(runs []AuditClusterRun) []AuditClusterGroup {
	slices.SortFunc(runs, func(a, b AuditClusterRun) int {
		return strings.Compare(fmt.Sprintf("%s/%s/%020d", a.Hostname, a.Repository, a.RunID), fmt.Sprintf("%s/%s/%020d", b.Hostname, b.Repository, b.RunID))
	})
	groups := make(map[string]*AuditClusterGroup)
	for _, run := range runs {
		for _, signature := range run.Signatures {
			key := run.Outcome + "\x00" + signature
			if groups[key] == nil {
				groups[key] = &AuditClusterGroup{Outcome: run.Outcome, Signature: signature, RepresentativeRun: run}
			}
			group := groups[key]
			group.Count++
			group.RunIDs = append(group.RunIDs, run.RunID)
			group.RunURLs = append(group.RunURLs, run.URL)
		}
	}
	result := make([]AuditClusterGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b AuditClusterGroup) int {
		return strings.Compare(a.Outcome+"\x00"+a.Signature, b.Outcome+"\x00"+b.Signature)
	})
	return result
}
