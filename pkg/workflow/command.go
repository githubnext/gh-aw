package workflow

import (
	"fmt"
	"slices"
)

// buildEventAwareCommandCondition creates a condition that only applies command checks to comment-related events
// commandEvents: list of event identifiers where command should be active (nil = all events)
func buildEventAwareCommandCondition(commandName string, commandEvents []string, hasOtherEvents bool) ConditionNode {
	// Define the command condition using proper expression nodes
	commandText := fmt.Sprintf("/%s", commandName)

	// Get the filtered events where command should be active
	filteredEvents := FilterCommentEvents(commandEvents)
	eventNames := GetCommentEventNames(filteredEvents)

	// Build command checks for different content sources based on filtered events
	var commandChecks []ConditionNode

	// Check which events are enabled and build appropriate checks
	hasIssues := slices.Contains(eventNames, "issues")
	hasIssueComment := slices.Contains(eventNames, "issue_comment")
	hasPR := slices.Contains(eventNames, "pull_request")
	hasPRReview := slices.Contains(eventNames, "pull_request_review_comment")

	if hasIssues {
		issueBodyCheck := BuildContains(
			BuildPropertyAccess("github.event.issue.body"),
			BuildStringLiteral(commandText),
		)
		commandChecks = append(commandChecks, issueBodyCheck)
	}

	if hasIssueComment || hasPRReview {
		// issue_comment and pull_request_review_comment both use github.event.comment.body
		commentBodyCheck := BuildContains(
			BuildPropertyAccess("github.event.comment.body"),
			BuildStringLiteral(commandText),
		)
		commandChecks = append(commandChecks, commentBodyCheck)
	}

	if hasPR {
		prBodyCheck := BuildContains(
			BuildPropertyAccess("github.event.pull_request.body"),
			BuildStringLiteral(commandText),
		)
		commandChecks = append(commandChecks, prBodyCheck)
	}

	// Combine all command checks with OR using BuildDisjunction helper
	var commandCondition ConditionNode
	if len(commandChecks) == 0 {
		// No events enabled - this indicates a configuration error
		panic(fmt.Sprintf("No valid comment events specified for command '%s'. At least one event must be enabled.", commandName))
	} else {
		// BuildDisjunction handles arrays of size 1 or more correctly
		commandCondition = BuildDisjunction(commandChecks...)
	}

	if !hasOtherEvents {
		// If there are no other events, just use the simple command condition
		return commandCondition
	}

	// Define which events should be checked for command using expression nodes
	// Only include the filtered events
	var commentEventTerms []ConditionNode
	for _, eventName := range eventNames {
		commentEventTerms = append(commentEventTerms, BuildEventTypeEquals(eventName))
	}

	commentEventChecks := BuildDisjunction(commentEventTerms...)

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
	}
}
