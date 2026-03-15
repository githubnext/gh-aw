package workflow

import (
	"errors"
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var labelCommandLog = logger.New("workflow:label_command")

// labelCommandSupportedEvents defines the GitHub Actions events that support label-command triggers
var labelCommandSupportedEvents = []string{"issues", "pull_request", "discussion"}

// FilterLabelCommandEvents returns the label-command events to use based on the specified identifiers.
// If identifiers is nil or empty, returns all supported events.
func FilterLabelCommandEvents(identifiers []string) []string {
	if len(identifiers) == 0 {
		labelCommandLog.Print("No label-command event identifiers specified, returning all events")
		return labelCommandSupportedEvents
	}

	var result []string
	for _, id := range identifiers {
		if slices.Contains(labelCommandSupportedEvents, id) {
			result = append(result, id)
		}
	}

	labelCommandLog.Printf("Filtered label-command events: %v -> %v", identifiers, result)
	return result
}

// buildLabelCommandCondition creates a condition that checks whether the triggering label
// matches one of the configured label-command names. For non-label events (e.g.
// workflow_dispatch and any other events in LabelCommandOtherEvents), the condition
// passes unconditionally so that manual runs and other triggers still work.
func buildLabelCommandCondition(labelNames []string, labelCommandEvents []string, hasOtherEvents bool) (ConditionNode, error) {
	labelCommandLog.Printf("Building label-command condition: labels=%v, events=%v, has_other_events=%t",
		labelNames, labelCommandEvents, hasOtherEvents)

	if len(labelNames) == 0 {
		return nil, errors.New("no label names provided for label-command trigger")
	}

	filteredEvents := FilterLabelCommandEvents(labelCommandEvents)
	if len(filteredEvents) == 0 {
		return nil, errors.New("no valid events specified for label-command trigger")
	}

	// Build the label-name match condition: label1 == name OR label2 == name ...
	var labelNameChecks []ConditionNode
	for _, labelName := range labelNames {
		labelNameChecks = append(labelNameChecks, BuildEquals(
			BuildPropertyAccess("github.event.label.name"),
			BuildStringLiteral(labelName),
		))
	}
	var labelNameMatch ConditionNode
	if len(labelNameChecks) == 1 {
		labelNameMatch = labelNameChecks[0]
	} else {
		labelNameMatch = BuildDisjunction(false, labelNameChecks...)
	}

	// Build per-event checks: (event_name == 'issues' AND label matches) OR ...
	var eventChecks []ConditionNode
	for _, event := range filteredEvents {
		eventChecks = append(eventChecks, &AndNode{
			Left:  BuildEventTypeEquals(event),
			Right: labelNameMatch,
		})
	}
	labelCondition := BuildDisjunction(false, eventChecks...)

	if !hasOtherEvents {
		// No other events — the label condition is the entire condition.
		return labelCondition, nil
	}

	// When there are other events (e.g. workflow_dispatch from the expanded shorthand, or
	// user-supplied events), we allow non-label events through unconditionally and only
	// require the label-name check for label events.
	var labelEventChecks []ConditionNode
	for _, event := range filteredEvents {
		labelEventChecks = append(labelEventChecks, BuildEventTypeEquals(event))
	}
	isLabelEvent := BuildDisjunction(false, labelEventChecks...)
	isNotLabelEvent := &NotNode{Child: isLabelEvent}

	return &OrNode{
		Left:  &AndNode{Left: isLabelEvent, Right: labelNameMatch},
		Right: isNotLabelEvent,
	}, nil
}
