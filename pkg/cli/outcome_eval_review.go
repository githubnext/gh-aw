package cli

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeReviewLog = logger.New("cli:outcome_eval_review")

var outcomeReviewGHAPIGet = ghAPIGet
var outcomeReviewGHAPIGetArray = ghAPIGetArray

func evalAddReviewer(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	outcomeReviewLog.Printf("Evaluating add-reviewer outcome: repo=%s, pr=%d", repo, num)
	if num == 0 || repo == "" {
		report.Result = OutcomeError
		report.EvalError = "missing PR number or repo"
		return report
	}

	requestedReviewers := metadataStringSlice(item.Metadata, "requested_reviewers")
	requestedTeams := metadataStringSlice(item.Metadata, "requested_team_reviewers")

	reviews, err := outcomeReviewGHAPIGetArray(fmt.Sprintf("pulls/%d/reviews", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	requested, err := outcomeReviewGHAPIGet(fmt.Sprintf("pulls/%d/requested_reviewers", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	outcomeReviewLog.Printf("Fetched %d reviews for PR #%d (requested reviewers=%d, teams=%d)", len(reviews), num, len(requestedReviewers), len(requestedTeams))
	latestByReviewer := evalAddReviewerLatestByReviewer(requestedReviewers, reviews, item.Timestamp)
	if reviewerReport, ok := evalAddReviewerFromRequestedReviews(report, latestByReviewer); ok {
		return reviewerReport
	}

	// We cannot cheaply verify team membership for each reviewer from this endpoint,
	// so any submitted post-request review counts as medium-evidence team activity.
	if len(requestedTeams) > 0 && hasReviewAfterTimestamp(reviews, item.Timestamp) {
		return evalAddReviewerTeamReviewSubmitted(report)
	}

	currentUsers := extractLogins(requested["users"])
	currentTeams := extractTeamSlugs(requested["teams"])
	stillPending := intersectsFold(requestedReviewers, currentUsers) || intersectsFold(requestedTeams, currentTeams)
	if stillPending {
		return evalAddReviewerStillPending(report)
	}

	return evalAddReviewerNoSubmittedReview(report, requestedReviewers, requestedTeams)
}

func evalAddReviewerLatestByReviewer(requestedReviewers []string, reviews []map[string]any, timestamp string) map[string]map[string]any {
	requestedReviewerSet := make(map[string]struct{}, len(requestedReviewers))
	for _, reviewer := range requestedReviewers {
		requestedReviewerSet[strings.ToLower(reviewer)] = struct{}{}
	}

	latestByReviewer := make(map[string]map[string]any, len(requestedReviewerSet))
	for _, review := range reviews {
		login := strings.ToLower(outcomeNestedString(review["user"], "login"))
		if _, ok := requestedReviewerSet[login]; !ok {
			continue
		}
		state := strings.ToUpper(outcomeString(review["state"]))
		submittedAt := outcomeString(review["submitted_at"])
		if state == "" || state == "PENDING" || submittedAt == "" { //nolint:tolowerequalfold
			continue
		}
		if !timestampOnOrAfter(submittedAt, timestamp) {
			continue
		}
		prev, ok := latestByReviewer[login]
		if !ok || timestampOnOrAfter(submittedAt, outcomeString(prev["submitted_at"])) {
			latestByReviewer[login] = review
		}
	}
	return latestByReviewer
}

func evalAddReviewerFromRequestedReviews(report OutcomeReport, latestByReviewer map[string]map[string]any) (OutcomeReport, bool) {
	var approvedReviewer string
	var submittedReviewer string
	for login, review := range latestByReviewer {
		if strings.EqualFold(outcomeString(review["state"]), "APPROVED") {
			approvedReviewer = login
			break
		}
		if submittedReviewer == "" {
			submittedReviewer = login
		}
	}
	switch {
	case approvedReviewer != "":
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("requested reviewer %s approved", approvedReviewer)
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusAccepted,
			EvidenceStrength: EvidenceStrong,
			Signal:           "review_approved",
		}
		return report, true
	case submittedReviewer != "":
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("requested reviewer %s submitted a review", submittedReviewer)
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusAccepted,
			EvidenceStrength: EvidenceMedium,
			Signal:           "review_submitted",
		}
		return report, true
	}
	return report, false
}

func evalAddReviewerTeamReviewSubmitted(report OutcomeReport) OutcomeReport {
	report.Result = OutcomeAccepted
	report.Detail = "team review request received a review"
	report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusAccepted,
		EvidenceStrength: EvidenceMedium,
		Signal:           "review_submitted",
	}
	return report
}

func evalAddReviewerStillPending(report OutcomeReport) OutcomeReport {
	report.Result = OutcomePending
	report.Detail = "review request still pending"
	report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusPending,
		EvidenceStrength: EvidenceMedium,
		Signal:           "awaiting_review",
	}
	return report
}

func evalAddReviewerNoSubmittedReview(report OutcomeReport, requestedReviewers []string, requestedTeams []string) OutcomeReport {
	if len(requestedReviewers) > 0 || len(requestedTeams) > 0 {
		report.Result = OutcomeRejected
		report.Detail = "review request removed without submitted review"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusRejected,
			EvidenceStrength: EvidenceStrong,
			Signal:           "review_request_removed",
		}
		return report
	}

	report.Result = OutcomeUnknown
	report.Detail = "no persisted reviewer request metadata"
	report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusUnknown,
		EvidenceStrength: EvidenceWeak,
		Signal:           "unknown",
	}
	return report
}

func evalSubmitPullRequestReview(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	outcomeReviewLog.Printf("Evaluating submit-review outcome: repo=%s, pr=%d", repo, num)
	if num == 0 || repo == "" {
		report.Result = OutcomeError
		report.EvalError = "missing PR number or repo"
		return report
	}

	pr, err := outcomeReviewGHAPIGet(fmt.Sprintf("pulls/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}
	reviews, err := outcomeReviewGHAPIGetArray(fmt.Sprintf("pulls/%d/reviews", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	reviewID := metadataInt(item.Metadata, "review_id")
	review := findReviewByID(reviews, reviewID)
	if review == nil {
		review = latestReviewAfterTimestamp(reviews, item.Timestamp)
	}
	if review == nil {
		outcomeReviewLog.Printf("Submitted review not found for PR #%d (reviewID=%d, reviews=%d)", num, reviewID, len(reviews))
		report.Result = OutcomeUnknown
		report.Detail = "submitted review not found"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusUnknown,
			EvidenceStrength: EvidenceWeak,
			Signal:           "review_missing",
		}
		return report
	}

	reviewState := strings.ToUpper(outcomeString(review["state"]))
	reviewSubmittedAt := outcomeString(review["submitted_at"])
	prMerged, _ := pr["merged"].(bool)
	prState, _ := pr["state"].(string)
	return evalSubmitPullRequestReviewClassify(evalSubmitPullRequestReviewClassifyParams{Report: report, Item: item, Repo: repo, Num: num, PR: pr, Reviews: reviews, Review: review, ReviewState: reviewState, ReviewSubmittedAt: reviewSubmittedAt, PRMerged: prMerged, PRState: prState})
}

type evalSubmitPullRequestReviewClassifyParams struct {
	Report            OutcomeReport
	Item              CreatedItemReport
	Repo              string
	Num               int
	PR                map[string]any
	Reviews           []map[string]any
	Review            map[string]any
	ReviewState       string
	ReviewSubmittedAt string
	PRMerged          bool
	PRState           string
}

func evalSubmitPullRequestReviewClassify(p evalSubmitPullRequestReviewClassifyParams) OutcomeReport {
	switch {
	case p.ReviewState == "DISMISSED": //nolint:tolowerequalfold
		return evalSubmitPullRequestReviewRejected(p.Report, "review dismissed by repo admin", "review_dismissed", EvidenceStrong)
	case p.PRMerged && p.ReviewState == "APPROVED": //nolint:tolowerequalfold
		p.Report.Result = OutcomeAccepted
		p.Report.Detail = "approved review followed by merge"
		p.Report.TimeToOutcomeHours = timeBetween(p.Item.Timestamp, outcomeString(p.PR["merged_at"]))
		p.Report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusAccepted,
			EvidenceStrength: EvidenceStrong,
			Signal:           "review_approved",
		}
		return p.Report
	case p.PRMerged && p.ReviewState == "CHANGES_REQUESTED": //nolint:tolowerequalfold
		if updatedReport, ok := evalSubmitPullRequestReviewChangesAddressed(p.Report, p.Item, p.Repo, p.Num, p.PR, p.ReviewSubmittedAt); ok {
			return updatedReport
		}
	case p.PRState == "closed" && !p.PRMerged:
		p.Report = evalSubmitPullRequestReviewRejected(p.Report, "PR closed without merge after review submission", "closed_without_merge_after_review", EvidenceMedium)
		p.Report.TimeToOutcomeHours = timeBetween(p.Item.Timestamp, outcomeString(p.PR["closed_at"]))
		return p.Report
	case p.PRState == "open" && isLatestReview(p.Reviews, p.Review):
		p.Report.Result = OutcomePending
		p.Report.Detail = "review is latest review on open PR"
		p.Report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusPending,
			EvidenceStrength: EvidenceMedium,
			Signal:           "latest_review_pending",
		}
		return p.Report
	}

	p.Report.Result = OutcomeUnknown
	p.Report.Detail = "review outcome could not be determined"
	p.Report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusUnknown,
		EvidenceStrength: EvidenceWeak,
		Signal:           "unknown",
	}
	return p.Report
}

func evalSubmitPullRequestReviewChangesAddressed(report OutcomeReport, item CreatedItemReport, repo string, num int, pr map[string]any, reviewSubmittedAt string) (OutcomeReport, bool) {
	commits, err := outcomeReviewGHAPIGetArray(fmt.Sprintf("pulls/%d/commits", num), repo)
	if err == nil && hasCommitAfterTimestamp(commits, reviewSubmittedAt) {
		report.Result = OutcomeAccepted
		report.Detail = "changes requested, updated, and merged"
		report.TimeToOutcomeHours = timeBetween(item.Timestamp, outcomeString(pr["merged_at"]))
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusAccepted,
			EvidenceStrength: EvidenceMedium,
			Signal:           "changes_requested_addressed",
		}
		return report, true
	}
	return report, false
}

func evalSubmitPullRequestReviewRejected(report OutcomeReport, detail string, signal string, strength EvidenceStrength) OutcomeReport {
	report.Result = OutcomeRejected
	report.Detail = detail
	report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusRejected,
		EvidenceStrength: strength,
		Signal:           signal,
	}
	return report
}

func metadataStringSlice(metadata map[string]any, key string) []string {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			s := strings.TrimSpace(fmt.Sprint(value))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func metadataInt(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	switch value := metadata[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		var parsed int
		_, _ = fmt.Sscanf(value, "%d", &parsed)
		return parsed
	}
	return 0
}

func outcomeString(raw any) string {
	s, _ := raw.(string)
	return s
}

func outcomeNestedString(raw any, nestedKey string) string {
	obj, _ := raw.(map[string]any)
	if obj == nil {
		return ""
	}
	value, _ := obj[nestedKey].(string)
	return value
}

func extractLogins(raw any) []string {
	items, _ := raw.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if login := outcomeNestedString(item, "login"); login != "" {
			out = append(out, login)
		}
	}
	return out
}

func extractTeamSlugs(raw any) []string {
	items, _ := raw.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		slug := outcomeNestedString(item, "slug")
		if slug == "" {
			slug = outcomeNestedString(item, "name")
		}
		if slug != "" {
			out = append(out, slug)
		}
	}
	return out
}

func intersectsFold(a []string, b []string) bool {
	for _, left := range a {
		if slices.ContainsFunc(b, func(right string) bool {
			return strings.EqualFold(left, right)
		}) {
			return true
		}
	}
	return false
}

func timestampOnOrAfter(candidate string, threshold string) bool {
	if candidate == "" {
		return false
	}
	if threshold == "" {
		return true
	}
	candidateTime, err := time.Parse(time.RFC3339, candidate)
	if err != nil {
		return false
	}
	thresholdTime, err := time.Parse(time.RFC3339, threshold)
	if err != nil {
		return false
	}
	return !candidateTime.Before(thresholdTime)
}

func hasReviewAfterTimestamp(reviews []map[string]any, threshold string) bool {
	for _, review := range reviews {
		state := strings.ToUpper(outcomeString(review["state"]))
		if state == "" || state == "PENDING" { //nolint:tolowerequalfold
			continue
		}
		if timestampOnOrAfter(outcomeString(review["submitted_at"]), threshold) {
			return true
		}
	}
	return false
}

func findReviewByID(reviews []map[string]any, reviewID int) map[string]any {
	if reviewID == 0 {
		return nil
	}
	for _, review := range reviews {
		if metadataInt(review, "id") == reviewID {
			return review
		}
	}
	return nil
}

func latestReviewAfterTimestamp(reviews []map[string]any, threshold string) map[string]any {
	var latest map[string]any
	for _, review := range reviews {
		state := strings.ToUpper(outcomeString(review["state"]))
		submittedAt := outcomeString(review["submitted_at"])
		if state == "" || state == "PENDING" || submittedAt == "" { //nolint:tolowerequalfold
			continue
		}
		if !timestampOnOrAfter(submittedAt, threshold) {
			continue
		}
		if latest == nil || timestampOnOrAfter(submittedAt, outcomeString(latest["submitted_at"])) {
			latest = review
		}
	}
	return latest
}

func isLatestReview(reviews []map[string]any, review map[string]any) bool {
	latest := latestReviewAfterTimestamp(reviews, "")
	return latest != nil && metadataInt(latest, "id") == metadataInt(review, "id")
}

func hasCommitAfterTimestamp(commits []map[string]any, threshold string) bool {
	for _, commit := range commits {
		commitObj, _ := commit["commit"].(map[string]any)
		if timestampOnOrAfter(outcomeNestedString(commitObj["committer"], "date"), threshold) || timestampOnOrAfter(outcomeNestedString(commitObj["author"], "date"), threshold) {
			return true
		}
	}
	return false
}
