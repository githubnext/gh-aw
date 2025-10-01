package workflow

import "fmt"

// buildEventAwareCommandCondition creates a condition that only applies command checks to comment-related events
func buildEventAwareCommandCondition(commandName string, hasOtherEvents bool) ConditionNode {
	// Define the command condition using proper expression nodes
	commandText := fmt.Sprintf("/%s", commandName)

	// Build command checks for different content sources using expression nodes
	issueBodyCheck := BuildContains(
		BuildPropertyAccess("github.event.issue.body"),
		BuildStringLiteral(commandText),
	)
	commentBodyCheck := BuildContains(
		BuildPropertyAccess("github.event.comment.body"),
		BuildStringLiteral(commandText),
	)
	prBodyCheck := BuildContains(
		BuildPropertyAccess("github.event.pull_request.body"),
		BuildStringLiteral(commandText),
	)

	// Combine all command checks with OR
	commandCondition := &OrNode{
		Left: &OrNode{
			Left:  issueBodyCheck,
			Right: commentBodyCheck,
		},
		Right: prBodyCheck,
	}

	if !hasOtherEvents {
		// If there are no other events, just use the simple command condition
		return commandCondition
	}

	// Define which events should be checked for command using expression nodes
	commentEventChecks := &DisjunctionNode{
		Terms: []ConditionNode{
			BuildEventTypeEquals("issues"),
			BuildEventTypeEquals("issue_comment"),
			BuildEventTypeEquals("pull_request"),
			BuildEventTypeEquals("pull_request_review_comment"),
		},
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
