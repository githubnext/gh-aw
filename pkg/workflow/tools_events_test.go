package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildCommandTriggerEventsMap_SlashCommandOnly_NoLabeledEvent is a regression test for the
// bug where a slash_command-only workflow (no LabelCommand) would emit spurious "labeled" event
// types in the generated on: section.
func TestBuildCommandTriggerEventsMap_SlashCommandOnly_NoLabeledEvent(t *testing.T) {
	data := &WorkflowData{
		Command:            []string{"deploy"},
		CommandCentralized: false,
		// LabelCommand is intentionally empty — this is the regression path.
	}
	c := NewCompiler()
	eventsMap, err := c.buildCommandTriggerEventsMap(data)
	require.NoError(t, err)

	for eventName, v := range eventsMap {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		types, _ := m["types"].([]any)
		assert.NotContains(t, types, "labeled",
			"slash_command-only workflow must not emit spurious 'labeled' type in event %q", eventName)
	}
}

// TestBuildCommandTriggerEventsMap_WithLabelCommand_IncludesLabeledEvent verifies that when
// label_command is configured alongside slash_command, the "labeled" type IS included.
func TestBuildCommandTriggerEventsMap_WithLabelCommand_IncludesLabeledEvent(t *testing.T) {
	data := &WorkflowData{
		Command:            []string{"deploy"},
		CommandCentralized: false,
		LabelCommand:       []string{"bug"},
	}
	c := NewCompiler()
	eventsMap, err := c.buildCommandTriggerEventsMap(data)
	require.NoError(t, err)

	// At least one event entry must carry the "labeled" type.
	foundLabeled := false
	for _, v := range eventsMap {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		switch t2 := m["types"].(type) {
		case []any:
			for _, ty := range t2 {
				if ty == "labeled" {
					foundLabeled = true
				}
			}
		case []string:
			for _, ty := range t2 {
				if ty == "labeled" {
					foundLabeled = true
				}
			}
		}
	}
	assert.True(t, foundLabeled, "expected 'labeled' type when label_command is configured alongside slash_command")
}
