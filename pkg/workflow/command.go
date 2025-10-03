package workflow

import "fmt"

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
	hasIssues := containsEventName(eventNames, "issues")
	hasIssueComment := containsEventName(eventNames, "issue_comment")
	hasPR := containsEventName(eventNames, "pull_request")
	hasPRReview := containsEventName(eventNames, "pull_request_review_comment")

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

	// Combine all command checks with OR
	var commandCondition ConditionNode
	if len(commandChecks) == 0 {
		// No events enabled, this should not happen but handle gracefully
		// Return a false condition
		commandCondition = BuildEquals(
			BuildStringLiteral("true"),
			BuildStringLiteral("false"),
		)
	} else if len(commandChecks) == 1 {
		commandCondition = commandChecks[0]
	} else {
		// Build OR chain for multiple checks
		commandCondition = commandChecks[0]
		for i := 1; i < len(commandChecks); i++ {
			commandCondition = &OrNode{
				Left:  commandCondition,
				Right: commandChecks[i],
			}
		}
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

	commentEventChecks := &DisjunctionNode{
		Terms: commentEventTerms,
	}

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

// containsEventName checks if a slice of event names contains a specific event name
func containsEventName(eventNames []string, eventName string) bool {
	for _, name := range eventNames {
		if name == eventName {
			return true
		}
	}
	return false
}
