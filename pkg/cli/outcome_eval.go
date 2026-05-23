package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var outcomeEvalLog = logger.New("cli:outcome_eval")

// OutcomeResult classifies what happened to a safe output after execution.
type OutcomeResult string

const (
	OutcomeAccepted  OutcomeResult = "accepted"
	OutcomeRejected  OutcomeResult = "rejected"
	OutcomeIgnored   OutcomeResult = "ignored"
	OutcomePending   OutcomeResult = "pending"
	OutcomeLifecycle OutcomeResult = "lifecycle"
	OutcomeError     OutcomeResult = "error"
)

// OutcomeReport is the result of evaluating one safe output item.
type OutcomeReport struct {
	Type               string        `json:"type" console:"header:Type"`
	ObjectURL          string        `json:"object_url,omitempty" console:"header:URL,omitempty"`
	ObjectNumber       int           `json:"object_number,omitempty" console:"header:#,omitempty"`
	Repo               string        `json:"repo,omitempty" console:"header:Repo,omitempty"`
	Result             OutcomeResult `json:"result" console:"header:Outcome"`
	Detail             string        `json:"detail,omitempty" console:"header:Detail,omitempty"`
	TimeToOutcomeHours float64       `json:"time_to_outcome_hours,omitempty" console:"header:Time,omitempty"`
	HumanComments      int           `json:"human_comments,omitempty" console:"header:Comments,omitempty"`
	HumanEdits         int           `json:"human_edits,omitempty" console:"header:Edits,omitempty"`
	HumanReviews       int           `json:"human_reviews,omitempty" console:"header:Reviews,omitempty"`
	ZeroTouch          bool          `json:"zero_touch,omitempty" console:"header:Zero-touch,omitempty"`
	CreatedAt          string        `json:"created_at" console:"-"`
	CheckedAt          string        `json:"checked_at" console:"-"`
	EvalError          string        `json:"eval_error,omitempty" console:"-"`
}

// OutcomeSummary aggregates outcomes across multiple safe output items.
type OutcomeSummary struct {
	Total                  int     `json:"total" console:"header:Total"`
	Accepted               int     `json:"accepted" console:"header:Accepted"`
	Rejected               int     `json:"rejected" console:"header:Rejected"`
	Ignored                int     `json:"ignored" console:"header:Ignored"`
	Pending                int     `json:"pending" console:"header:Pending"`
	Lifecycle              int     `json:"lifecycle" console:"header:Lifecycle"`
	Errors                 int     `json:"errors" console:"header:Errors"`
	ZeroTouch              int     `json:"zero_touch" console:"header:Zero-touch"`
	AcceptanceRate         float64 `json:"acceptance_rate" console:"header:Acceptance Rate"`
	WasteRate              float64 `json:"waste_rate" console:"header:Waste Rate"`
	ZeroTouchRate          float64 `json:"zero_touch_rate" console:"header:Zero-touch Rate"`
	MedianTimeToOutcome    float64 `json:"median_time_to_outcome_hours,omitempty"`
	CostPerAcceptedOutcome float64 `json:"cost_per_accepted_outcome,omitempty"`
}

// outcomeEvaluator is a function that evaluates one safe output item.
type outcomeEvaluator func(item CreatedItemReport, repoOverride string) OutcomeReport

// outcomeEvaluators maps safe output types to their evaluator functions.
var outcomeEvaluators = map[string]outcomeEvaluator{
	"create_pull_request":                   evalCreatePullRequest,
	"create_issue":                          evalCreateIssue,
	"add_comment":                           evalAddComment,
	"add_labels":                            evalAddLabels,
	"assign_to_agent":                       evalAssignToAgent,
	"close_issue":                           evalCloseSticky,
	"close_pull_request":                    evalCloseSticky,
	"close_discussion":                      evalCloseDiscussion,
	"create_discussion":                     evalCreateDiscussion,
	"hide_comment":                          evalHideComment,
	"assign_milestone":                      evalAssignMilestone,
	"create_pull_request_review_comment":    evalReviewComment,
	"resolve_pull_request_review_thread":    evalResolveThread,
	"mark_pull_request_as_ready_for_review": evalMarkReady,
	"push_to_pull_request_branch":           evalPushToPRBranch,
}

// EvaluateOutcomes checks the current state of all safe output items from a run.
func EvaluateOutcomes(items []CreatedItemReport, repoOverride string) []OutcomeReport {
	outcomeEvalLog.Printf("Evaluating outcomes: items=%d, repo_override=%q", len(items), repoOverride)
	if repoOverride == "" {
		slug, err := GetCurrentRepoSlug()
		if err == nil {
			repoOverride = slug
			outcomeEvalLog.Printf("Resolved repo override from current repo slug: %s", repoOverride)
		}
	}

	reports := make([]OutcomeReport, 0, len(items))
	skipped := 0
	for _, item := range items {
		if item.Type == "noop" || item.Type == "missing_tool" || item.Type == "missing_data" || item.Type == "report_incomplete" {
			skipped++
			continue
		}
		repo := item.Repo
		if repo == "" {
			repo = repoOverride
		}
		eval, ok := outcomeEvaluators[item.Type]
		if !ok {
			outcomeEvalLog.Printf("No evaluator registered for type %q, using generic sticky", item.Type)
			eval = evalGenericSticky
		}
		report := eval(item, repo)
		report.CreatedAt = item.Timestamp
		report.CheckedAt = time.Now().UTC().Format(time.RFC3339)
		reports = append(reports, report)
	}
	outcomeEvalLog.Printf("Outcome evaluation complete: reports=%d, skipped=%d", len(reports), skipped)
	return reports
}

// ComputeOutcomeSummary aggregates outcome reports into a summary.
func ComputeOutcomeSummary(reports []OutcomeReport, totalCost float64) OutcomeSummary {
	s := OutcomeSummary{Total: len(reports)}
	var times []float64
	for _, r := range reports {
		switch r.Result {
		case OutcomeAccepted:
			s.Accepted++
			if r.ZeroTouch {
				s.ZeroTouch++
			}
		case OutcomeRejected:
			s.Rejected++
		case OutcomeIgnored:
			s.Ignored++
		case OutcomePending:
			s.Pending++
		case OutcomeLifecycle:
			s.Lifecycle++
		case OutcomeError:
			s.Errors++
		}
		if r.TimeToOutcomeHours > 0 {
			times = append(times, r.TimeToOutcomeHours)
		}
	}
	resolved := s.Accepted + s.Rejected
	if resolved > 0 {
		s.AcceptanceRate = float64(s.Accepted) / float64(resolved)
	}
	if s.Total > 0 {
		s.WasteRate = float64(s.Rejected) / float64(s.Total)
	}
	if s.Accepted > 0 {
		s.ZeroTouchRate = float64(s.ZeroTouch) / float64(s.Accepted)
		if totalCost > 0 {
			s.CostPerAcceptedOutcome = totalCost / float64(s.Accepted)
		}
	}
	if len(times) > 0 {
		s.MedianTimeToOutcome = medianFloat(times)
	}
	return s
}

// normalizeRepoForAPI splits a repo string of the form "[HOST/]owner/repo" into
// the owner/repo portion and an optional host. Most callers pass plain "owner/repo",
// but GHES and Proxima installs may supply "HOST/owner/repo".
func normalizeRepoForAPI(repo string) (ownerRepo string, host string) {
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) == 3 {
		// HOST/owner/repo
		return parts[1] + "/" + parts[2], parts[0]
	}
	return repo, ""
}

// ghAPIGet calls the GitHub REST API via gh cli and returns the parsed JSON.
func ghAPIGet(endpoint string, repo string) (map[string]any, error) {
	ownerRepo, host := normalizeRepoForAPI(repo)
	outcomeEvalLog.Printf("gh api GET: repo=%s, endpoint=%s, host=%q", ownerRepo, endpoint, host)
	args := []string{"api", fmt.Sprintf("repos/%s/%s", ownerRepo, endpoint)}
	var output []byte
	var err error
	if host != "" {
		output, err = workflow.RunGHWithHost("Checking outcome...", host, args...)
	} else {
		output, err = workflow.RunGH("Checking outcome...", args...)
	}
	if err != nil {
		outcomeEvalLog.Printf("gh api GET failed: endpoint=%s, err=%v", endpoint, err)
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parsing response for %s: %w", endpoint, err)
	}
	return result, nil
}

// ghAPIGetArray calls the GitHub REST API and returns a JSON array.
func ghAPIGetArray(endpoint string, repo string) ([]map[string]any, error) {
	ownerRepo, host := normalizeRepoForAPI(repo)
	args := []string{"api", fmt.Sprintf("repos/%s/%s", ownerRepo, endpoint)}
	var output []byte
	var err error
	if host != "" {
		output, err = workflow.RunGHWithHost("Checking outcome...", host, args...)
	} else {
		output, err = workflow.RunGH("Checking outcome...", args...)
	}
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var result []map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parsing response for %s: %w", endpoint, err)
	}
	return result, nil
}

// timeBetween computes hours between two ISO timestamps.
func timeBetween(from, to string) float64 {
	t1, err1 := time.Parse(time.RFC3339, from)
	t2, err2 := time.Parse(time.RFC3339, to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return t2.Sub(t1).Hours()
}

// medianFloat returns the median of a float slice.
func medianFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	n := len(vals)
	sorted := make([]float64, n)
	copy(sorted, vals)
	for i := range sorted {
		for j := i + 1; j < n; j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// parseNumberFromURL extracts a number from a GitHub URL like
// https://github.com/owner/repo/pull/42 or .../issues/108
func parseNumberFromURL(url string) int {
	parts := strings.Split(url, "/")
	for i := range slices.Backward(parts) {
		var n int
		if _, err := fmt.Sscanf(parts[i], "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// parseRepoFromURL extracts owner/repo from a GitHub URL.
func parseRepoFromURL(url string) string {
	// https://github.com/owner/repo/...
	const prefix = "github.com/"
	_, rest, found := strings.Cut(url, prefix)
	if !found {
		return ""
	}
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

// isBotUser returns true if the login looks like a bot account.
func isBotUser(login string) bool {
	return strings.HasSuffix(login, "[bot]") || login == "github-actions" || login == "copilot-swe-agent"
}

// resolveItemRepo returns the repo to use for API calls, preferring the item's repo field.
func resolveItemRepo(item CreatedItemReport, repoOverride string) string {
	if item.Repo != "" {
		return item.Repo
	}
	if item.URL != "" {
		if r := parseRepoFromURL(item.URL); r != "" {
			return r
		}
	}
	return repoOverride
}

// resolveItemNumber returns the object number, trying item.Number first then URL parsing.
func resolveItemNumber(item CreatedItemReport) int {
	if item.Number > 0 {
		return item.Number
	}
	if item.URL != "" {
		return parseNumberFromURL(item.URL)
	}
	return 0
}
