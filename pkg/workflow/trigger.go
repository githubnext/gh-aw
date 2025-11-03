package workflow

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// TriggerConfig represents the parsed and validated trigger section from workflow frontmatter
type TriggerConfig struct {
	// Raw holds the original trigger data from frontmatter for backward compatibility
	Raw map[string]any

	// Simple is set when the trigger is a simple string like "push" or "workflow_dispatch"
	Simple string

	// Events maps event names to their configurations
	// For simple events without config: map["push"] = nil
	// For complex events: map["pull_request"] = {types: [opened], branches: [main]}
	Events map[string]EventConfig

	// Command holds command trigger configuration if present
	Command *CommandTriggerConfig

	// Reaction holds reaction configuration if present in the on section
	Reaction string

	// StopAfter holds the stop-after deadline if present in the on section
	StopAfter string

	// ManualApproval holds the environment name for manual approval if present
	ManualApproval string
}

// EventConfig represents the configuration for a specific event type
type EventConfig struct {
	// Raw holds the raw configuration for this event
	Raw any

	// Types specifies the activity types that trigger the event (e.g., [opened, closed])
	Types []string

	// Branches specifies branch filters
	Branches []string

	// Tags specifies tag filters
	Tags []string

	// Paths specifies path filters
	Paths []string

	// WorkflowRuns specifies workflow run filters for workflow_run events
	WorkflowRuns []string

	// Additional fields can be stored in Raw for events with custom configurations
}

// CommandTriggerConfig represents the command trigger configuration
type CommandTriggerConfig struct {
	// Name is the command name (e.g., "bot-name" for /bot-name)
	Name string

	// Events lists the events where the command should be active
	// nil or empty means all comment-related events
	Events []string
}

// ParseTrigger parses the "on" section from frontmatter into a TriggerConfig
func ParseTrigger(onValue any) (*TriggerConfig, error) {
	if onValue == nil {
		return nil, fmt.Errorf("on field is required but not provided")
	}

	config := &TriggerConfig{
		Events: make(map[string]EventConfig),
	}

	// Handle simple string trigger (e.g., on: "push")
	if simpleStr, ok := onValue.(string); ok {
		config.Simple = simpleStr
		config.Events[simpleStr] = EventConfig{Raw: nil}
		return config, nil
	}

	// Handle complex trigger configuration (map)
	onMap, ok := onValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("on field must be a string or object, got %T", onValue)
	}

	// Store the raw map for backward compatibility
	config.Raw = onMap

	// Parse command trigger if present
	if commandValue, hasCommand := onMap["command"]; hasCommand {
		commandConfig, err := parseCommandTrigger(commandValue)
		if err != nil {
			return nil, fmt.Errorf("invalid command trigger: %w", err)
		}
		config.Command = commandConfig
	}

	// Parse reaction if present
	if reactionValue, hasReaction := onMap["reaction"]; hasReaction {
		if reactionStr, ok := reactionValue.(string); ok {
			config.Reaction = reactionStr
		}
	}

	// Parse stop-after if present
	if stopAfterValue, hasStopAfter := onMap["stop-after"]; hasStopAfter {
		if stopAfterStr, ok := stopAfterValue.(string); ok {
			config.StopAfter = stopAfterStr
		}
	}

	// Parse manual-approval if present
	if manualApprovalValue, hasManualApproval := onMap["manual-approval"]; hasManualApproval {
		if manualApprovalStr, ok := manualApprovalValue.(string); ok {
			config.ManualApproval = manualApprovalStr
		}
	}

	// Parse all event configurations
	for eventName, eventValue := range onMap {
		// Skip special fields that are not event types
		if eventName == "command" || eventName == "reaction" || eventName == "stop-after" || eventName == "manual-approval" {
			continue
		}

		eventConfig, err := parseEventConfig(eventValue)
		if err != nil {
			return nil, fmt.Errorf("invalid configuration for event '%s': %w", eventName, err)
		}
		config.Events[eventName] = eventConfig
	}

	return config, nil
}

// parseCommandTrigger parses command trigger configuration
func parseCommandTrigger(commandValue any) (*CommandTriggerConfig, error) {
	config := &CommandTriggerConfig{}

	// Handle simple string command name
	if commandStr, ok := commandValue.(string); ok {
		config.Name = commandStr
		return config, nil
	}

	// Handle complex command configuration
	commandMap, ok := commandValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("command must be a string or object, got %T", commandValue)
	}

	// Extract command name
	if nameValue, hasName := commandMap["name"]; hasName {
		if nameStr, ok := nameValue.(string); ok {
			config.Name = nameStr
		} else {
			return nil, fmt.Errorf("command.name must be a string, got %T", nameValue)
		}
	}

	// Extract events list
	if eventsValue, hasEvents := commandMap["events"]; hasEvents {
		switch v := eventsValue.(type) {
		case []any:
			for _, ev := range v {
				if evStr, ok := ev.(string); ok {
					config.Events = append(config.Events, evStr)
				}
			}
		case []string:
			config.Events = v
		case string:
			// Single event as string
			config.Events = []string{v}
		}
	}

	return config, nil
}

// parseEventConfig parses the configuration for a specific event
func parseEventConfig(eventValue any) (EventConfig, error) {
	config := EventConfig{
		Raw: eventValue,
	}

	// Handle nil event (e.g., workflow_dispatch: null)
	if eventValue == nil {
		return config, nil
	}

	// Handle array values (e.g., schedule: [{cron: ...}])
	// For arrays, we just store the raw value without parsing
	if _, ok := eventValue.([]any); ok {
		return config, nil
	}

	// Handle event configuration as map
	eventMap, ok := eventValue.(map[string]any)
	if !ok {
		// If it's not nil, not an array, and not a map, it's invalid
		return config, fmt.Errorf("event configuration must be null, an array, or an object, got %T", eventValue)
	}

	// Parse types
	if typesValue, hasTypes := eventMap["types"]; hasTypes {
		config.Types = parseStringArray(typesValue)
	}

	// Parse branches
	if branchesValue, hasBranches := eventMap["branches"]; hasBranches {
		config.Branches = parseStringArray(branchesValue)
	}

	// Parse tags
	if tagsValue, hasTags := eventMap["tags"]; hasTags {
		config.Tags = parseStringArray(tagsValue)
	}

	// Parse paths
	if pathsValue, hasPaths := eventMap["paths"]; hasPaths {
		config.Paths = parseStringArray(pathsValue)
	}

	// Parse workflows (for workflow_run events)
	if workflowsValue, hasWorkflows := eventMap["workflows"]; hasWorkflows {
		config.WorkflowRuns = parseStringArray(workflowsValue)
	}

	return config, nil
}

// parseStringArray converts various array representations to []string
func parseStringArray(value any) []string {
	var result []string

	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	case []string:
		result = v
	case string:
		// Single string value
		result = []string{v}
	}

	return result
}

// HasEvent returns true if the trigger includes the specified event
func (t *TriggerConfig) HasEvent(eventName string) bool {
	if t == nil {
		return false
	}
	_, exists := t.Events[eventName]
	return exists
}

// HasCommand returns true if the trigger includes a command
func (t *TriggerConfig) HasCommand() bool {
	return t != nil && t.Command != nil
}

// GetCommandName returns the command name if present, empty string otherwise
func (t *TriggerConfig) GetCommandName() string {
	if t == nil || t.Command == nil {
		return ""
	}
	return t.Command.Name
}

// GetCommandEvents returns the list of events where the command is active
func (t *TriggerConfig) GetCommandEvents() []string {
	if t == nil || t.Command == nil {
		return nil
	}
	return t.Command.Events
}

// ToYAML converts the TriggerConfig back to YAML format for compilation
func (t *TriggerConfig) ToYAML() (string, error) {
	if t == nil {
		return "", nil
	}

	// If we have the raw map, use it for backward compatibility
	if t.Raw != nil {
		data := map[string]any{"on": t.Raw}
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(yamlBytes), nil
	}

	// If it's a simple trigger, return the simple form
	if t.Simple != "" {
		data := map[string]any{"on": t.Simple}
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(yamlBytes), nil
	}

	// Build the trigger map from events
	onMap := make(map[string]any)
	for eventName, eventConfig := range t.Events {
		onMap[eventName] = eventConfig.Raw
	}

	// Add special fields
	if t.Command != nil {
		if t.Command.Name != "" && len(t.Command.Events) == 0 {
			onMap["command"] = map[string]any{"name": t.Command.Name}
		} else if t.Command.Name != "" {
			onMap["command"] = map[string]any{
				"name":   t.Command.Name,
				"events": t.Command.Events,
			}
		}
	}

	if t.Reaction != "" {
		onMap["reaction"] = t.Reaction
	}

	if t.StopAfter != "" {
		onMap["stop-after"] = t.StopAfter
	}

	if t.ManualApproval != "" {
		onMap["manual-approval"] = t.ManualApproval
	}

	data := map[string]any{"on": onMap}
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}
