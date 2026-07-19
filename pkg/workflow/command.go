package workflow

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var commandLog = logger.New("workflow:command")

func isWildcardCommandName(commandName string) bool {
	return len(commandName) > 1 && strings.HasSuffix(commandName, "*")
}

// buildEventAwareCommandCondition creates a condition that only applies command checks to comment-related events
// commandNames: list of command names that can trigger this workflow
// commandEvents: list of event identifiers where command should be active (nil = all events)
func buildEventAwareCommandCondition(commandNames []string, commandEvents []string, hasOtherEvents bool) (ConditionNode, error) {
	commandLog.Printf("Building event-aware command condition: commands=%v, event_count=%d, has_other_events=%t",
		commandNames, len(commandEvents), hasOtherEvents)

	if len(commandNames) == 0 {
		return nil, errors.New("no command names provided")
	}

	// Get the filtered events where command should be active
	filteredEvents := FilterCommentEvents(commandEvents)
	eventNames := GetCommentEventNames(filteredEvents)
	commandLog.Printf("Filtered command events: commands=%v, filtered_count=%d", commandNames, len(eventNames))

	commandChecks := buildCommandEventChecks(commandNames, eventNames)

	if len(commandChecks) == 0 {
		return nil, fmt.Errorf("no valid comment events specified for commands %v - at least one event must be enabled", commandNames)
	}
	commandLog.Printf("Built %d command check(s) for commands: %v", len(commandChecks), commandNames)
	commandCondition := BuildDisjunction(false, commandChecks...)

	if !hasOtherEvents {
		commandLog.Print("Using simple command condition (no other events)")
		return commandCondition, nil
	}
	commandLog.Print("Using event-aware condition (mixed command and non-command events)")

	return buildMixedEventCommandCondition(commandCondition, eventNames), nil
}

func buildMultiCommandCheck(commandNames []string, bodyAccessor string) ConditionNode {
	var commandOrChecks []ConditionNode
	for _, commandName := range commandNames {
		commandOrChecks = append(commandOrChecks, buildSingleCommandCheck(commandName, bodyAccessor))
	}
	if len(commandOrChecks) == 1 {
		return commandOrChecks[0]
	}
	return BuildDisjunction(false, commandOrChecks...)
}

func buildSingleCommandCheck(commandName string, bodyAccessor string) ConditionNode {
	if isWildcardCommandName(commandName) {
		commandPrefix := "/" + strings.TrimSuffix(commandName, "*")
		return BuildFunctionCall("startsWith", BuildPropertyAccess(bodyAccessor), BuildStringLiteral(commandPrefix))
	}

	commandText := "/" + commandName
	startsWithMatch := BuildFunctionCall("startsWith", BuildPropertyAccess(bodyAccessor), BuildStringLiteral(fmt.Sprintf("/%s ", commandName)))
	startsWithNewlineMatch := BuildFunctionCall("startsWith", BuildPropertyAccess(bodyAccessor), BuildStringLiteral(fmt.Sprintf("/%s\n", commandName)))
	exactMatch := BuildEquals(BuildPropertyAccess(bodyAccessor), BuildStringLiteral(commandText))
	return &OrNode{Left: &OrNode{Left: startsWithMatch, Right: startsWithNewlineMatch}, Right: exactMatch}
}

func buildCommandEventChecks(commandNames []string, eventNames []string) []ConditionNode {
	var commandChecks []ConditionNode
	addEventCheck := func(event, accessor string) {
		commandChecks = append(commandChecks, &AndNode{Left: BuildEventTypeEquals(event), Right: buildMultiCommandCheck(commandNames, accessor)})
	}
	if slices.Contains(eventNames, "issues") {
		addEventCheck("issues", "github.event.issue.body")
	}
	if slices.Contains(eventNames, "issue_comment") {
		commandChecks = append(commandChecks, buildIssueCommentCommandCheck(commandNames, false))
	}
	if slices.Contains(eventNames, "pull_request_comment") {
		commandChecks = append(commandChecks, buildIssueCommentCommandCheck(commandNames, true))
	}
	if slices.Contains(eventNames, "pull_request_review_comment") {
		addEventCheck("pull_request_review_comment", "github.event.comment.body")
	}
	if slices.Contains(eventNames, "pull_request") {
		addEventCheck("pull_request", "github.event.pull_request.body")
	}
	if slices.Contains(eventNames, "discussion") {
		addEventCheck("discussion", "github.event.discussion.body")
	}
	if slices.Contains(eventNames, "discussion_comment") {
		addEventCheck("discussion_comment", "github.event.comment.body")
	}
	return commandChecks
}

func buildIssueCommentCommandCheck(commandNames []string, isPR bool) ConditionNode {
	issuePRAccessor := BuildPropertyAccess("github.event.issue.pull_request")
	var prCheck ConditionNode = BuildEquals(issuePRAccessor, BuildNullLiteral())
	if isPR {
		prCheck = BuildNotEquals(issuePRAccessor, BuildNullLiteral())
	}
	return &AndNode{
		Left: BuildEventTypeEquals("issue_comment"),
		Right: &AndNode{
			Left:  buildMultiCommandCheck(commandNames, "github.event.comment.body"),
			Right: prCheck,
		},
	}
}

func buildMixedEventCommandCondition(commandCondition ConditionNode, eventNames []string) ConditionNode {
	var commentEventTerms []ConditionNode
	actualEventNames := make(map[string]struct { // Use map to deduplicate
	})
	for _, eventName := range eventNames {
		actualName := GetActualGitHubEventName(eventName)
		if !setutil.Contains(actualEventNames, actualName) {
			actualEventNames[actualName] = struct {
			}{}
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

	return &OrNode{
		Left:  commentEventCheck,
		Right: nonCommentEvents,
	}
}
