package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/workqueue"
	"go.yaml.in/yaml/v3"
)

// HealthWorkQueue reports current ledger evidence separately from sampled Actions runs.
type HealthWorkQueue struct {
	Status    string                `json:"status"`
	Workflows []string              `json:"workflows,omitempty"`
	Warnings  []string              `json:"warnings,omitempty"`
	Stats     *workQueueStatsReport `json:"stats,omitempty"`
}

var healthReadWorkQueue = func(ctx context.Context, branch workqueue.Branch) ([]workqueue.QueueCommit, error) {
	return branch.Read(ctx)
}

func (report *HealthWorkQueue) warn(err error) {
	report.Warnings = append(report.Warnings, err.Error())
	healthLog.Printf("Work-queue health unavailable: %v", err)
}

func collectHealthWorkQueue(collection *HealthCollection) {
	if len(collection.workflows) == 0 {
		return
	}
	report := &HealthWorkQueue{Status: "unavailable"}
	collection.WorkQueue = report
	if collection.stopped {
		report.warn(errors.New("work-queue detection not completed: health API collection stopped"))
		return
	}
	out, err := healthAPI(collection, healthRepositoryEndpoint(collection.repoOverride))
	if err != nil {
		report.warn(fmt.Errorf("cannot resolve work-queue repository: %w", err))
		return
	}
	var repository struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(out, &repository); err != nil || repository.FullName == "" || repository.DefaultBranch == "" {
		report.warn(errors.New("cannot resolve work-queue repository: missing or invalid repository metadata"))
		return
	}
	enableHealthLocalGit(WorkflowRun{Repository: repository.FullName}, collection)
	detectHealthWorkQueueWorkflows(repository.DefaultBranch, collection, report)
	if len(report.Workflows) == 0 {
		if len(report.Warnings) == 0 {
			collection.WorkQueue = nil
		}
		return
	}
	stats, err := readHealthWorkQueueStats(repository.FullName, collection)
	if err != nil {
		report.warn(err)
		return
	}
	report.Stats = stats
	report.Status = "observed"
	if stats.AdmissionPaused {
		report.Warnings = append(report.Warnings, "work-queue admission is paused")
	}
	if stats.GrantsPaused {
		report.Warnings = append(report.Warnings, "work-queue grants are paused")
	}
	if stats.DeliveryByState["failed"] > 0 {
		report.Warnings = append(report.Warnings, "work-queue has failed deliveries; completion does not establish verified delivery")
	}
	if len(report.Warnings) > 0 {
		report.Status = "attention_required"
	}
}

func detectHealthWorkQueueWorkflows(ref string, collection *HealthCollection, report *HealthWorkQueue) {
	for _, entry := range collection.workflows {
		content, err := healthWorkQueueWorkflowContent(entry.Path, ref, collection)
		if err != nil {
			report.warn(fmt.Errorf("cannot detect work-queue usage in %s: %w", entry.Path, err))
			if collection.stopped {
				break
			}
			continue
		}
		enabled, err := healthWorkflowUsesWorkQueue(content)
		if err != nil {
			report.warn(fmt.Errorf("cannot detect work-queue usage in %s: invalid workflow YAML", entry.Path))
			continue
		}
		if enabled {
			report.Workflows = append(report.Workflows, entry.Path)
		}
	}
}

func readHealthWorkQueueStats(repository string, collection *HealthCollection) (*workQueueStatsReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	branch := workqueue.Branch{Remote: repository, Name: workqueue.DefaultBranch, Host: collection.Host}
	commits, err := healthReadWorkQueue(ctx, branch)
	if err != nil {
		return nil, fmt.Errorf("cannot read work-queue ledger: %w", currentWorkQueueDiagnosticError("read", err))
	}
	at := time.Now().UnixMilli()
	state, err := workqueue.ReplayAt(commits, at)
	if err != nil {
		return nil, currentWorkQueueDiagnosticError("ledger", err)
	}
	if state.Policy == nil {
		return nil, errors.New("work-queue has no installed policy")
	}
	if !strings.EqualFold(state.Repository, repository) {
		return nil, errors.New("work-queue ledger repository differs from the health target")
	}
	stats, err := workQueueStatsAt(state, branch.Name, at)
	if err != nil {
		return nil, currentWorkQueueDiagnosticError("diagnostics", err)
	}
	return &stats, nil
}

func healthWorkQueueWorkflowContent(workflowPath, ref string, collection *HealthCollection) ([]byte, error) {
	sha, err := healthWorkflowFileSHA(workflowPath, ref, collection)
	if err != nil {
		return nil, err
	}
	if collection.localGit != nil {
		content, err := collection.localGit.blob(sha)
		if err == nil {
			collection.LocalRevisionReads++
			return content, nil
		}
		healthLog.Printf("Local work-queue workflow blob unavailable: %v; using GitHub API", err)
	}
	endpoint := path.Join(healthRepositoryEndpoint(collection.repoOverride), "git/blobs", sha)
	content, err := healthAPI(collection, endpoint, "--jq", ".content | @base64d")
	collection.RevisionLookups++
	return content, err
}

func healthWorkflowUsesWorkQueue(content []byte) (bool, error) {
	type environment struct {
		Env map[string]string `yaml:"env"`
	}
	var definition struct {
		Env  map[string]string `yaml:"env"`
		Jobs map[string]struct {
			Env   map[string]string `yaml:"env"`
			Steps []environment     `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(content, &definition); err != nil {
		return false, err
	}
	enabled := func(env environment) bool { return env.Env["GH_AW_WORK_QUEUE_ENABLED"] == "true" }
	if enabled(environment{Env: definition.Env}) {
		return true, nil
	}
	for _, job := range definition.Jobs {
		if enabled(environment{Env: job.Env}) {
			return true, nil
		}
		if slices.ContainsFunc(job.Steps, enabled) {
			return true, nil
		}
	}
	return false, nil
}

func renderHealthWorkQueue(w io.Writer, report *HealthWorkQueue) error {
	if report == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\n[health_work_queue] status=%s workflows=%d; current ledger, independent of run time window\n",
		report.Status, len(report.Workflows)); err != nil {
		return err
	}
	if report.Stats != nil {
		if _, err := fmt.Fprintln(w, workQueueStatsText(*report.Stats)); err != nil {
			return err
		}
		for _, pool := range report.Stats.Pools {
			if _, err := io.WriteString(w, workQueuePoolDiagnosticsText(pool)); err != nil {
				return err
			}
		}
		if reasons := workQueueReasonCounts(report.Stats.DeliveryFailureReasons); reasons != "" {
			if _, err := fmt.Fprintf(w, "Delivery failures: %s\n", reasons); err != nil {
				return err
			}
		}
	}
	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(w, "Warning: %s\n", escapeWorkQueueDisplay(warning)); err != nil {
			return err
		}
	}
	return nil
}
