package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const healthMaximumLifecycleRequests = 40

type HealthPRLifecycleEvidence struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	Number    int       `json:"number,omitempty"`
	EventAt   time.Time `json:"event_at,omitzero"`
	Attempt   int       `json:"attempt"`
	UpdatedAt time.Time `json:"updated_at"`
}

type healthLifecycleResponse struct {
	data []byte
	err  error
}

type healthLifecyclePR struct {
	Number   int       `json:"number"`
	State    string    `json:"state"`
	ClosedAt time.Time `json:"closed_at"`
	MergedAt time.Time `json:"merged_at"`
	Head     struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func healthLifecycleAPI(collection *HealthCollection, endpoint string, extra ...string) ([]byte, error) {
	if cached, ok := collection.lifecycleResponses[endpoint]; ok {
		return cached.data, cached.err
	}
	if collection.lifecycleResponses == nil {
		collection.lifecycleResponses = make(map[string]healthLifecycleResponse)
	}
	if collection.LifecycleRequests >= healthMaximumLifecycleRequests {
		return nil, errors.New("health PR lifecycle request limit reached")
	}
	before := collection.APIRequests
	data, err := healthAPI(collection, endpoint, extra...)
	collection.LifecycleRequests += collection.APIRequests - before
	collection.lifecycleResponses[endpoint] = healthLifecycleResponse{data: data, err: err}
	return data, err
}

// Like logs' obsoletePRWindow, correlate exact commits and timestamped events,
// never a branch's present-day absence or a PR's present-day closed state alone.
func classifyHealthPRLifecycle(run WorkflowRun, collection *HealthCollection) HealthPRLifecycleEvidence {
	evidence := HealthPRLifecycleEvidence{Status: "retained", Reason: "no_matching_lifecycle_event",
		Attempt: run.Attempt, UpdatedAt: run.UpdatedAt}
	start, eligible := obsoletePRWindow(run)
	if !eligible || run.HeadBranch == "" || run.UpdatedAt.Before(start) {
		evidence.Status, evidence.Reason = "unknown", "missing_run_identity_or_timestamps"
		return evidence
	}
	pr, err := fetchHealthLifecyclePR(run, collection)
	if err != nil {
		evidence.Status, evidence.Reason = "unknown", err.Error()
		return evidence
	}
	evidence.Number = pr.Number
	if pr.State == "closed" && !pr.ClosedAt.After(run.UpdatedAt) {
		evidence.EventAt, evidence.Reason = pr.ClosedAt, "pr_closed"
		if !pr.MergedAt.IsZero() && !pr.MergedAt.After(run.UpdatedAt) {
			evidence.EventAt, evidence.Reason = pr.MergedAt, "pr_merged"
		}
		// A failure needs an explicit checkout diagnostic below; a cancellation
		// needs closure during this attempt, not before it or after it ended.
		if run.Conclusion == "failure" || evidence.EventAt.After(start) {
			return verifyHealthLifecycleJobs(run, evidence, collection)
		}
	}
	if run.Conclusion == "failure" {
		return evidence
	}
	return classifyHealthHeadDeletion(run, start, evidence, collection)
}

func fetchHealthLifecyclePR(run WorkflowRun, collection *HealthCollection) (healthLifecyclePR, error) {
	data, err := healthLifecycleAPI(collection, fmt.Sprintf("%s/commits/%s/pulls?per_page=100",
		healthRepositoryEndpoint(collection.repoOverride), url.PathEscape(run.HeadSha)))
	if err != nil {
		return healthLifecyclePR{}, err
	}
	var prs []healthLifecyclePR
	if err := json.Unmarshal(data, &prs); err != nil || prs == nil || len(prs) >= 100 {
		return healthLifecyclePR{}, errors.New("invalid_or_incomplete_commit_pull_requests")
	}
	var selected healthLifecyclePR
	matching := 0
	for _, pr := range prs {
		if pr.Head.Ref == "" || pr.Head.Ref == run.HeadBranch {
			selected = pr
			matching++
		}
	}
	if matching != 1 || selected.Number <= 0 || selected.Head.Ref != run.HeadBranch {
		return healthLifecyclePR{}, errors.New("missing_or_ambiguous_pull_request")
	}
	pr := selected
	if pr.State != "open" && pr.State != "closed" || pr.State == "closed" && pr.ClosedAt.IsZero() {
		return healthLifecyclePR{}, errors.New("missing_pull_request_state_or_timestamp")
	}
	return pr, nil
}

func classifyHealthHeadDeletion(run WorkflowRun, start time.Time, evidence HealthPRLifecycleEvidence, collection *HealthCollection) HealthPRLifecycleEvidence {
	// A current 404 has no historical timestamp. Use the PR's deletion events.
	var restoredAt time.Time
	for page := 1; page <= 2; page++ {
		data, err := healthLifecycleAPI(collection, fmt.Sprintf("%s/issues/%d/events?per_page=100&page=%d",
			healthRepositoryEndpoint(collection.repoOverride), evidence.Number, page))
		if err != nil {
			evidence.Status, evidence.Reason = "unknown", err.Error()
			return evidence
		}
		var events []struct {
			Event     string    `json:"event"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := json.Unmarshal(data, &events); err != nil || events == nil {
			evidence.Status, evidence.Reason = "unknown", "invalid_branch_lifecycle_events"
			return evidence
		}
		for _, event := range events {
			if (event.Event == "head_ref_deleted" || event.Event == "head_ref_restored") && event.CreatedAt.IsZero() {
				evidence.Status, evidence.Reason = "unknown", "missing_branch_event_timestamp"
				return evidence
			}
			if event.Event == "head_ref_deleted" && event.CreatedAt.After(start) && !event.CreatedAt.After(run.UpdatedAt) {
				if evidence.Reason != "pr_head_deleted" || event.CreatedAt.After(evidence.EventAt) {
					evidence.EventAt, evidence.Reason = event.CreatedAt, "pr_head_deleted"
				}
			}
			if event.Event == "head_ref_restored" && !event.CreatedAt.After(run.UpdatedAt) && event.CreatedAt.After(restoredAt) {
				restoredAt = event.CreatedAt
			}
		}
		if len(events) < 100 {
			if evidence.Reason == "pr_head_deleted" && !restoredAt.After(evidence.EventAt) {
				return verifyHealthLifecycleJobs(run, evidence, collection)
			}
			evidence.EventAt = time.Time{}
			evidence.Reason = "no_matching_lifecycle_event"
			return evidence
		}
	}
	evidence.Status, evidence.Reason = "unknown", "branch_lifecycle_events_incomplete"
	return evidence
}

func verifyHealthLifecycleJobs(run WorkflowRun, evidence HealthPRLifecycleEvidence, collection *HealthCollection) HealthPRLifecycleEvidence {
	if err := collectHealthRepresentativeJobs(run, collection); err != nil {
		evidence.Status, evidence.Reason = "unknown", err.Error()
		return evidence
	}
	jobs := collection.jobs[healthRunAttempt{id: run.DatabaseID, attempt: run.Attempt}]
	if jobs == nil || !jobs.complete {
		evidence.Status, evidence.Reason = "unknown", "incomplete_lifecycle_job_evidence"
		return evidence
	}
	var aftermath []JobInfoWithDuration
	var checkoutCompleted time.Time
	for _, job := range jobs.details {
		if job.Status != "completed" || job.Conclusion == "" {
			evidence.Status, evidence.Reason = "unknown", "incomplete_lifecycle_job_status"
			return evidence
		}
		if job.Conclusion == "timed_out" || job.Conclusion == "startup_failure" {
			evidence.Reason = "genuine_job_failure"
			return evidence
		}
		if job.Conclusion != "failure" {
			if healthHasGenuineStepFailure(job) {
				evidence.Reason = "genuine_step_failure"
				return evidence
			}
			continue
		}
		if run.Conclusion == "failure" && job.Name == "conclusion" && healthOnlyFailedStep(job, "Handle agent failure") {
			aftermath = append(aftermath, job)
			continue
		}
		if run.Conclusion != "failure" || !healthClosureCheckoutJob(job, evidence.EventAt) {
			evidence.Reason = "genuine_job_failure"
			return evidence
		}
		confirmed, err := healthClosedCheckoutLog(job.ID, evidence.Number, collection)
		if err != nil {
			evidence.Status, evidence.Reason = "unknown", err.Error()
			return evidence
		}
		if !confirmed {
			evidence.Reason = "checkout_closure_not_confirmed"
			return evidence
		}
		checkoutCompleted = job.CompletedAt
	}
	if run.Conclusion == "failure" && checkoutCompleted.IsZero() {
		evidence.Status, evidence.Reason = "unknown", "missing_failed_checkout_job"
		return evidence
	}
	if !healthValidClosureAftermath(aftermath, checkoutCompleted) {
		evidence.Reason = "unconfirmed_failure_aftermath"
		return evidence
	}
	if run.Conclusion == "cancelled" && !healthHasTimedCancellation(jobs.details, evidence.EventAt) {
		evidence.Status, evidence.Reason = "unknown", "missing_correlated_cancelled_job"
		return evidence
	}
	evidence.Status = "ignored"
	return evidence
}

func healthValidClosureAftermath(jobs []JobInfoWithDuration, checkoutCompleted time.Time) bool {
	for _, job := range jobs {
		for _, step := range job.Steps {
			if step.Conclusion == "failure" && (step.StartedAt.IsZero() || step.StartedAt.Before(checkoutCompleted)) {
				return false
			}
		}
	}
	return true
}

func healthHasTimedCancellation(jobs []JobInfoWithDuration, eventAt time.Time) bool {
	for _, job := range jobs {
		if job.Conclusion == "cancelled" && !job.CompletedAt.IsZero() && !eventAt.After(job.CompletedAt) {
			return true
		}
	}
	return false
}

func healthHasGenuineStepFailure(job JobInfoWithDuration) bool {
	for _, step := range job.Steps {
		if isClusterFailureConclusion(step.Conclusion) && step.Conclusion != "cancelled" {
			return true
		}
	}
	return false
}

func healthClosedCheckoutLog(jobID int64, prNumber int, collection *HealthCollection) (bool, error) {
	data, err := healthLifecycleAPI(collection, fmt.Sprintf("%s/actions/jobs/%d/logs",
		healthRepositoryEndpoint(collection.repoOverride), jobID), "--allow-escape-sequences")
	if err != nil {
		return false, err
	}
	diagnostic := fmt.Sprintf("PR #%d closed before its head could be fetched. Stopping rather than running on the previous checkout.", prNumber)
	return strings.Contains(string(data), diagnostic), nil
}

func healthClosureCheckoutJob(job JobInfoWithDuration, closedAt time.Time) bool {
	if job.Name != "agent" || !healthOnlyFailedStep(job, "Checkout PR branch") || job.CompletedAt.IsZero() {
		return false
	}
	for _, step := range job.Steps {
		if step.Conclusion == "failure" && (step.CompletedAt.IsZero() || closedAt.After(step.CompletedAt)) {
			return false
		}
	}
	return true
}

func healthOnlyFailedStep(job JobInfoWithDuration, name string) bool {
	if job.ID <= 0 || len(job.Steps) == 0 {
		return false
	}
	found := false
	for _, step := range job.Steps {
		if !isClusterFailureConclusion(step.Conclusion) {
			continue
		}
		if step.Conclusion != "failure" || step.Name != name {
			return false
		}
		found = true
	}
	return found
}

func filterHealthPRLifecycleRun(run WorkflowRun, collection *HealthCollection) bool {
	if run.Event != "pull_request" || run.Status != "completed" ||
		(run.Conclusion != "cancelled" && run.Conclusion != "failure") {
		return false
	}
	if collection.PRLifecycleEvidence == nil {
		collection.PRLifecycleEvidence = make(map[int64]HealthPRLifecycleEvidence)
	}
	evidence, checked := collection.PRLifecycleEvidence[run.DatabaseID]
	if !checked || evidence.Attempt != run.Attempt || !evidence.UpdatedAt.Equal(run.UpdatedAt) {
		evidence = classifyHealthPRLifecycle(run, collection)
		collection.PRLifecycleEvidence[run.DatabaseID] = evidence
		if evidence.Status == "unknown" {
			healthLog.Printf("Health PR lifecycle evidence unavailable: run=%d reason=%s", run.DatabaseID, evidence.Reason)
		}
	}
	return evidence.Status == "ignored"
}
