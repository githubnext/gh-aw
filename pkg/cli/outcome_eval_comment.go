package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalCommentLog = logger.New("cli:outcome_eval_comment")

// evalAddComment checks whether a comment received replies, reactions, or was deleted/hidden.
func evalAddComment(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	outcomeEvalCommentLog.Printf("Evaluating add_comment: repo=%s, url=%s", repo, item.URL)
	report := OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      repo,
	}

	// Extract comment ID from URL: .../issues/123#issuecomment-456789 or .../comments/456789
	commentID := extractCommentID(item.URL)
	if commentID == "" {
		outcomeEvalCommentLog.Printf("Unable to extract comment ID from URL: %s", item.URL)
		report.Result = OutcomeError
		report.EvalError = "cannot extract comment ID from URL"
		return report
	}

	data, err := ghAPIGet("issues/comments/"+commentID, repo)
	if err != nil {
		// 404 means deleted
		if errorutil.IsNotFoundError(err) {
			outcomeEvalCommentLog.Printf("Comment %s deleted (404)", commentID)
			report.Result = OutcomeRejected
			report.Detail = "deleted"
			return report
		}
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	// Check reactions
	reactions, _ := data["reactions"].(map[string]any)
	totalReactions := 0
	if reactions != nil {
		if tc, ok := reactions["total_count"].(float64); ok {
			totalReactions = int(tc)
		}
	}

	// Check if the comment is minimized (hidden)
	// The REST API field is "performed_via_github_app" but minimized state
	// is not directly in REST. We approximate: if the comment body is empty
	// or the node_id can be checked via GraphQL. For now, use reactions+replies.

	// To check replies, we need the issue number and look for comments posted after this one
	issueNumber := parseNumberFromURL(item.URL)
	replyCount := 0
	if issueNumber > 0 {
		commentList, cerr := ghAPIGetArray(fmt.Sprintf("issues/%d/comments", issueNumber), repo)
		if cerr == nil {
			createdAt, _ := data["created_at"].(string)
			for _, c := range commentList {
				cCreatedAt, _ := c["created_at"].(string)
				if cCreatedAt > createdAt {
					user, _ := c["user"].(map[string]any)
					login, _ := user["login"].(string)
					if !isBotUser(login) {
						replyCount++
					}
				}
			}
		}
	}

	report.HumanComments = replyCount

	switch {
	case totalReactions > 0 || replyCount > 0:
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("%d reactions, %d replies", totalReactions, replyCount)
	default:
		report.Result = OutcomeIgnored
		report.Detail = "no engagement"
	}

	return report
}

// extractCommentID extracts the numeric comment ID from a GitHub comment URL.
// Handles formats like:
//
//	https://github.com/owner/repo/issues/123#issuecomment-456789
//	https://github.com/owner/repo/pull/123#issuecomment-456789
func extractCommentID(url string) string {
	if _, after, found := strings.Cut(url, "#issuecomment-"); found {
		return after
	}
	// Fallback: look for /comments/ID pattern
	const commentsPrefix = "/comments/"
	if idx := strings.LastIndex(url, commentsPrefix); idx >= 0 {
		rest := url[idx+len(commentsPrefix):]
		// Take only digits
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end > 0 {
			return rest[:end]
		}
	}
	return ""
}
