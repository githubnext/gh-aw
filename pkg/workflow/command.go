package workflow

import (
	"fmt"
	"slices"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var commandLog = logger.New("workflow:command")

// buildEventAwareCommandCondition creates a condition that only applies command checks to comment-related events
// commandEvents: list of event identifiers where command should be active (nil = all events)
func buildEventAwareCommandCondition(commandName string, commandEvents []string, hasOtherEvents bool) (ConditionNode, error) {
	commandLog.Printf("Building event-aware command condition: command=%s, event_count=%d, has_other_events=%t",
		commandName, len(commandEvents), hasOtherEvents)

	// Define the command condition using proper expression nodes
	commandText := fmt.Sprintf("/%s", commandName)

	// Get the filtered events where command should be active
	filteredEvents := FilterCommentEvents(commandEvents)
	eventNames := GetCommentEventNames(filteredEvents)
	commandLog.Printf("Filtered command events: command=%s, filtered_count=%d", commandName, len(eventNames))

	// Build command checks for different content sources based on filtered events
	var commandChecks []ConditionNode

	// Check which events are enabled and build appropriate checks
	hasIssues := slices.Contains(eventNames, "issues")
	hasIssueComment := slices.Contains(eventNames, "issue_comment")
	hasPRComment := slices.Contains(eventNames, "pull_request_comment")
	hasPR := slices.Contains(eventNames, "pull_request")
	hasPRReview := slices.Contains(eventNames, "pull_request_review_comment")
	hasDiscussion := slices.Contains(eventNames, "discussion")
	hasDiscussionComment := slices.Contains(eventNames, "discussion_comment")

	if hasIssues {
		// issues event - check github.event.issue.body only when event is 'issues'
		issueBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("issues"),
			Right: BuildContains(
				BuildPropertyAccess("github.event.issue.body"),
				BuildStringLiteral(commandText),
			),
		}
		commandChecks = append(commandChecks, issueBodyCheck)
	}

	if hasIssueComment {
		// issue_comment event only on issues (not PRs) - check github.event.comment.body only when event is 'issue_comment'
		// and github.event.issue.pull_request is null
		commentBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("issue_comment"),
			Right: &AndNode{
				Left: BuildContains(
					BuildPropertyAccess("github.event.comment.body"),
					BuildStringLiteral(commandText),
				),
				Right: BuildEquals(
					BuildPropertyAccess("github.event.issue.pull_request"),
					BuildNullLiteral(),
				),
			},
		}
		commandChecks = append(commandChecks, commentBodyCheck)
	}

	if hasPRComment {
		// pull_request_comment event only on PRs - check github.event.comment.body only when event is 'issue_comment'
		// and github.event.issue.pull_request is not null
		prCommentBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("issue_comment"),
			Right: &AndNode{
				Left: BuildContains(
					BuildPropertyAccess("github.event.comment.body"),
					BuildStringLiteral(commandText),
				),
				Right: BuildNotEquals(
					BuildPropertyAccess("github.event.issue.pull_request"),
					BuildNullLiteral(),
				),
			},
		}
		commandChecks = append(commandChecks, prCommentBodyCheck)
	}

	if hasPRReview {
		// pull_request_review_comment uses github.event.comment.body only when event is 'pull_request_review_comment'
		reviewCommentBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("pull_request_review_comment"),
			Right: BuildContains(
				BuildPropertyAccess("github.event.comment.body"),
				BuildStringLiteral(commandText),
			),
		}
		commandChecks = append(commandChecks, reviewCommentBodyCheck)
	}

	if hasPR {
		// pull_request event - check github.event.pull_request.body only when event is 'pull_request'
		prBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("pull_request"),
			Right: BuildContains(
				BuildPropertyAccess("github.event.pull_request.body"),
				BuildStringLiteral(commandText),
			),
		}
		commandChecks = append(commandChecks, prBodyCheck)
	}

	if hasDiscussion {
		// discussion event - check github.event.discussion.body only when event is 'discussion'
		discussionBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("discussion"),
			Right: BuildContains(
				BuildPropertyAccess("github.event.discussion.body"),
				BuildStringLiteral(commandText),
			),
		}
		commandChecks = append(commandChecks, discussionBodyCheck)
	}

	if hasDiscussionComment {
		// discussion_comment event - check github.event.comment.body only when event is 'discussion_comment'
		discussionCommentBodyCheck := &AndNode{
			Left: BuildEventTypeEquals("discussion_comment"),
			Right: BuildContains(
				BuildPropertyAccess("github.event.comment.body"),
				BuildStringLiteral(commandText),
			),
		}
		commandChecks = append(commandChecks, discussionCommentBodyCheck)
	}

	// Combine all command checks with OR using BuildDisjunction helper
	var commandCondition ConditionNode
	if len(commandChecks) == 0 {
		// No events enabled - this indicates a configuration error
		return nil, fmt.Errorf("no valid comment events specified for command '%s' - at least one event must be enabled", commandName)
	}
	// BuildDisjunction handles arrays of size 1 or more correctly
	commandCondition = BuildDisjunction(false, commandChecks...)

	if !hasOtherEvents {
		// If there are no other events, just use the simple command condition
		return commandCondition, nil
	}

	// Define which events should be checked for command using expression nodes
	// Map logical event names to actual GitHub event names
	var commentEventTerms []ConditionNode
	actualEventNames := make(map[string]bool) // Use map to deduplicate
	for _, eventName := range eventNames {
		actualName := GetActualGitHubEventName(eventName)
		if !actualEventNames[actualName] {
			actualEventNames[actualName] = true
			commentEventTerms = append(commentEventTerms, BuildEventTypeEquals(actualName))
		}
	}

	commentEventChecks := BuildDisjunction(false, commentEventTerms...)

	// For comment events: check command; for other events: allow unconditionally
	commentEventCheck := &AndNode{
		Left:  commentEventChecks,
		Right: commandCondition,
	}

	// Allow all non-comment events to run
	nonCommentEvents := &NotNode{Child: commentEventChecks}

	// Combine: (comment events && command check) || (non-comment events)
	return &OrNode{
		Left:  commentEventCheck,
		Right: nonCommentEvents,
	}, nil
}
