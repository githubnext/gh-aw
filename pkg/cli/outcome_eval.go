package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/github"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var outcomeEvalLog = logger.New("cli:outcome_eval")

var objectiveMappingGHAPIGetArray = ghAPIGetArray
var objectiveMappingGHAPIGraphQL = ghAPIGraphQL

// OutcomeResult classifies what happened to a safe output after execution.
type OutcomeResult string

const (
	OutcomeAccepted  OutcomeResult = "accepted"
	OutcomeRejected  OutcomeResult = "rejected"
	OutcomeIgnored   OutcomeResult = "ignored"
	OutcomePending   OutcomeResult = "pending"
	OutcomeUnknown   OutcomeResult = "unknown"
	OutcomeLifecycle OutcomeResult = "lifecycle"
	OutcomeError     OutcomeResult = "error"
)

// OutcomeReport is the result of evaluating one safe output item.
type OutcomeReport struct {
	OutcomeEvaluation
	Type               string        `json:"type" console:"header:Type"`
	ObjectURL          string        `json:"object_url,omitempty" console:"header:URL,omitempty"`
	ObjectNumber       int           `json:"object_number,omitempty" console:"header:#,omitempty"`
	TracedRootURL      string        `json:"traced_root_url,omitempty" console:"-"`
	Repo               string        `json:"repo,omitempty" console:"header:Repo,omitempty"`
	Result             OutcomeResult `json:"result" console:"header:Outcome"`
	Detail             string        `json:"detail,omitempty" console:"header:Detail,omitempty"`
	TimeToOutcomeHours float64       `json:"time_to_outcome_hours,omitempty" console:"header:Time,omitempty"`
	HumanComments      int           `json:"human_comments,omitempty" console:"header:Comments,omitempty"`
	HumanEdits         int           `json:"human_edits,omitempty" console:"header:Edits,omitempty"`
	HumanReviews       int           `json:"human_reviews,omitempty" console:"header:Reviews,omitempty"`
	ZeroTouch          bool          `json:"zero_touch,omitempty" console:"header:Zero-touch,omitempty"`
	ObjectiveValue     int           `json:"objective_value,omitempty" console:"header:Obj Value,omitempty"`
	ObjectiveLabels    []string      `json:"objective_labels,omitempty" console:"-"`
	CreatedAt          string        `json:"created_at" console:"-"`
	CheckedAt          string        `json:"checked_at" console:"-"`
	EvalError          string        `json:"eval_error,omitempty" console:"-"`
}

// OutcomeSummary aggregates outcomes across multiple safe output items.
type OutcomeSummary struct {
	Total                   int               `json:"total" console:"header:Total"`
	Accepted                int               `json:"accepted" console:"header:Accepted"`
	Rejected                int               `json:"rejected" console:"header:Rejected"`
	Ignored                 int               `json:"ignored" console:"header:Ignored"`
	Pending                 int               `json:"pending" console:"header:Pending"`
	AcceptedStrong          int               `json:"accepted_strong,omitempty"`
	AcceptedMedium          int               `json:"accepted_medium,omitempty"`
	AcceptedWeak            int               `json:"accepted_weak,omitempty"`
	FallbackExistsOnlyCount int               `json:"fallback_exists_only_count,omitempty"`
	Lifecycle               int               `json:"lifecycle" console:"header:Lifecycle"`
	Errors                  int               `json:"errors" console:"header:Errors"`
	ZeroTouch               int               `json:"zero_touch" console:"header:Zero-touch"`
	AcceptanceRate          float64           `json:"acceptance_rate" console:"header:Acceptance Rate"`
	WasteRate               float64           `json:"waste_rate" console:"header:Waste Rate"`
	ZeroTouchRate           float64           `json:"zero_touch_rate" console:"header:Zero-touch Rate"`
	MedianTimeToOutcome     float64           `json:"median_time_to_outcome_hours,omitempty"`
	CostPerAcceptedOutcome  float64           `json:"cost_per_accepted_outcome,omitempty"`
	TotalObjectiveValue     int               `json:"total_objective_value,omitempty" console:"header:Total Obj Value"`
	AcceptedObjectiveValue  int               `json:"accepted_objective_value,omitempty" console:"header:Accepted Obj Value"`
	ObjectiveEfficiency     float64           `json:"objective_efficiency,omitempty" console:"header:Obj Efficiency"`
	DomainBreakdowns        []DomainBreakdown `json:"domain_breakdowns,omitempty" console:"-"`
}

// outcomeEvaluator is a function that evaluates one safe output item.
type outcomeEvaluator func(item CreatedItemReport, repoOverride string) OutcomeReport

// outcomeEvaluators maps safe output types to their evaluator functions.
var outcomeEvaluators = map[string]outcomeEvaluator{
	"create_pull_request":                   evalCreatePullRequest,
	"create_issue":                          evalCreateIssue,
	"update_issue":                          evalUpdateIssue,
	"update_pull_request":                   evalUpdatePullRequest,
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
	"add_reviewer":                          evalAddReviewer,
	"submit_pull_request_review":            evalSubmitPullRequestReview,
}

// EvaluateOutcomes checks the current state of all safe output items from a run.
// The mapping parameter is required and defines how labels map to objective values.
func EvaluateOutcomes(items []CreatedItemReport, repoOverride string, mapping *github.ObjectiveMapping) []OutcomeReport {
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
		report.OutcomeEvaluation = normalizeOutcomeEvaluation(report)

		// Compute objective value from issue/PR labels
		enrichOutcomeWithObjectiveValue(&report, repo, mapping)

		reports = append(reports, report)
	}
	outcomeEvalLog.Printf("Outcome evaluation complete: reports=%d, skipped=%d", len(reports), skipped)
	return reports
}

// ComputeOutcomeSummary aggregates outcome reports into a summary.
// The mapping parameter is required and defines how labels map to objective values.
func ComputeOutcomeSummary(reports []OutcomeReport, mapping *github.ObjectiveMapping) OutcomeSummary {
	s := OutcomeSummary{Total: len(reports)}
	var times []float64
	for _, r := range reports {
		eval := normalizeOutcomeEvaluation(r)
		switch eval.OutcomeStatus {
		case OutcomeStatusAccepted:
			s.Accepted++
			switch eval.EvidenceStrength {
			case EvidenceStrong:
				s.AcceptedStrong++
			case EvidenceMedium:
				s.AcceptedMedium++
			case EvidenceWeak:
				s.AcceptedWeak++
			}
			if r.ZeroTouch {
				s.ZeroTouch++
			}
		case OutcomeStatusRejected:
			s.Rejected++
		case OutcomeStatusIgnored:
			s.Ignored++
		case OutcomeStatusPending:
			s.Pending++
		}
		if eval.Signal == "target_exists_only" {
			s.FallbackExistsOnlyCount++
		}
		switch r.Result {
		case OutcomeLifecycle:
			s.Lifecycle++
		case OutcomeError:
			s.Errors++
		}
		if r.TimeToOutcomeHours > 0 {
			times = append(times, r.TimeToOutcomeHours)
		}

		// Aggregate objective values
		s.TotalObjectiveValue += r.ObjectiveValue
		if eval.OutcomeStatus == OutcomeStatusAccepted {
			s.AcceptedObjectiveValue += r.ObjectiveValue
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
	}
	if len(times) > 0 {
		s.MedianTimeToOutcome = medianFloat(times)
	}

	// Compute objective efficiency
	if s.TotalObjectiveValue > 0 {
		s.ObjectiveEfficiency = float64(s.AcceptedObjectiveValue) / float64(s.TotalObjectiveValue)
	}

	// Compute domain breakdowns
	s.DomainBreakdowns = ComputeDomainBreakdowns(reports)

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

// ghAPIGraphQL calls the GitHub GraphQL API via gh cli and returns the parsed JSON.
func ghAPIGraphQL(query string, repo string) (map[string]any, error) {
	ownerRepo, host := normalizeRepoForAPI(repo)
	args := []string{"api", "graphql", "-f", "query=" + query}
	var output []byte
	var err error
	if host != "" {
		output, err = workflow.RunGHWithHost("Checking outcome...", host, args...)
	} else {
		output, err = workflow.RunGH("Checking outcome...", args...)
	}
	if err != nil {
		return nil, fmt.Errorf("gh api graphql: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parsing graphql response for %s: %w", ownerRepo, err)
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

// enrichOutcomeWithObjectiveValue computes the objective value for an outcome by fetching
// its associated issue/PR labels and applying the label-to-value mapping.
func enrichOutcomeWithObjectiveValue(report *OutcomeReport, repo string, mapping *github.ObjectiveMapping) {
	if report == nil || mapping == nil {
		return
	}

	// Only compute objective values for items that have a GitHub object number
	num := report.ObjectNumber
	if num == 0 || repo == "" {
		return
	}

	// Skip types that don't have associated labels on issues/PRs
	if report.Type == "noop" || report.Type == "missing_tool" || report.Type == "missing_data" || report.Type == "report_incomplete" {
		return
	}

	outcomeEvalLog.Printf("Computing objective value: type=%s, repo=%s, number=%d", report.Type, repo, num)

	root, err := traceOutcomeRoot(*report, repo)
	if err != nil {
		outcomeEvalLog.Printf("Could not trace root for objective value computation: %v", err)
		return
	}
	report.TracedRootURL = root.URL

	labelNames := root.Labels
	if len(labelNames) > 0 {
		outcomeEvalLog.Printf("Fetched root labels for %s#%d: root=%s labels=%v", repo, num, root.URL, labelNames)
	}

	// Compute objective value
	objectiveValue := mapping.ComputeObjectiveValue(labelNames)
	objectiveLabels := mapping.GetObjectiveLabels(labelNames)

	report.ObjectiveValue = objectiveValue
	report.ObjectiveLabels = objectiveLabels
	outcomeEvalLog.Printf("Computed objective value for %s#%d: value=%d, labels=%v", repo, num, objectiveValue, objectiveLabels)
}

type tracedOutcomeRoot struct {
	URL    string
	Number int
	Labels []string
}

func traceOutcomeRoot(report OutcomeReport, repo string) (tracedOutcomeRoot, error) {
	if isPullRequestOutcomeType(report.Type) {
		root, err := tracePullRequestRoot(report.ObjectNumber, repo)
		if err == nil && root.Number > 0 {
			return root, nil
		}
		if err != nil {
			outcomeEvalLog.Printf("Falling back to direct labels after PR root trace failure: %v", err)
		}
	}

	labels, err := objectiveMappingGHAPIGetArray(fmt.Sprintf("issues/%d/labels", report.ObjectNumber), repo)
	if err != nil {
		return tracedOutcomeRoot{}, err
	}
	return tracedOutcomeRoot{
		URL:    report.ObjectURL,
		Number: report.ObjectNumber,
		Labels: labelsToStringsFromMaps(labels),
	}, nil
}

func isPullRequestOutcomeType(outcomeType string) bool {
	switch outcomeType {
	case "create_pull_request", "update_pull_request", "create_pull_request_review_comment",
		"resolve_pull_request_review_thread", "mark_pull_request_as_ready_for_review",
		"push_to_pull_request_branch", "add_reviewer", "submit_pull_request_review":
		return true
	default:
		return false
	}
}

func tracePullRequestRoot(prNumber int, repo string) (tracedOutcomeRoot, error) {
	ownerRepo, _ := normalizeRepoForAPI(repo)
	owner, name, found := strings.Cut(ownerRepo, "/")
	if !found || owner == "" || name == "" {
		return tracedOutcomeRoot{}, fmt.Errorf("invalid repo for root tracing: %s", repo)
	}

	query := fmt.Sprintf(`query {
		repository(owner: "%s", name: "%s") {
			pullRequest(number: %d) {
				closingIssuesReferences(first: 10) {
					nodes {
						number
						url
						labels(first: 20) {
							nodes { name }
						}
					}
				}
			}
		}
	}`,
		escapeGraphQLString(owner),
		escapeGraphQLString(name),
		prNumber,
	)

	result, err := objectiveMappingGHAPIGraphQL(query, repo)
	if err != nil {
		return tracedOutcomeRoot{}, err
	}
	data, _ := result["data"].(map[string]any)
	repository, _ := data["repository"].(map[string]any)
	pullRequest, _ := repository["pullRequest"].(map[string]any)
	closingRefs, _ := pullRequest["closingIssuesReferences"].(map[string]any)
	nodes, _ := closingRefs["nodes"].([]any)
	if len(nodes) == 0 {
		return tracedOutcomeRoot{}, fmt.Errorf("no closing issues found for PR #%d", prNumber)
	}
	firstNode, _ := nodes[0].(map[string]any)
	root := tracedOutcomeRoot{}
	if url, ok := firstNode["url"].(string); ok {
		root.URL = url
	}
	if number, ok := firstNode["number"].(float64); ok {
		root.Number = int(number)
	}
	if labels, ok := firstNode["labels"].(map[string]any); ok {
		if labelNodes, ok := labels["nodes"].([]any); ok {
			root.Labels = labelsToStringsFromNodes(labelNodes)
		}
	}
	return root, nil
}

func labelsToStringsFromNodes(nodes []any) []string {
	if len(nodes) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		labelMap, _ := node.(map[string]any)
		if name, ok := labelMap["name"].(string); ok {
			result = append(result, name)
		}
	}
	return result
}

// labelsToStringsFromMaps converts GitHub API label map objects to string slice.
func labelsToStringsFromMaps(labels []map[string]any) []string {
	if len(labels) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(labels))
	for _, labelMap := range labels {
		if name, ok := labelMap["name"].(string); ok {
			result = append(result, name)
		}
	}
	return result
}
