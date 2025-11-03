package workflow

import (
	"testing"
)

func TestParseTrigger_SimpleString(t *testing.T) {
	tests := []struct {
		name        string
		onValue     any
		wantSimple  string
		wantEvents  map[string]bool
		wantErr     bool
	}{
		{
			name:       "push trigger",
			onValue:    "push",
			wantSimple: "push",
			wantEvents: map[string]bool{"push": true},
			wantErr:    false,
		},
		{
			name:       "workflow_dispatch trigger",
			onValue:    "workflow_dispatch",
			wantSimple: "workflow_dispatch",
			wantEvents: map[string]bool{"workflow_dispatch": true},
			wantErr:    false,
		},
		{
			name:       "issues trigger",
			onValue:    "issues",
			wantSimple: "issues",
			wantEvents: map[string]bool{"issues": true},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseTrigger(tt.onValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if config.Simple != tt.wantSimple {
				t.Errorf("ParseTrigger() Simple = %v, want %v", config.Simple, tt.wantSimple)
			}

			for eventName, shouldExist := range tt.wantEvents {
				if config.HasEvent(eventName) != shouldExist {
					t.Errorf("ParseTrigger() HasEvent(%s) = %v, want %v", eventName, config.HasEvent(eventName), shouldExist)
				}
			}
		})
	}
}

func TestParseTrigger_ComplexEvents(t *testing.T) {
	tests := []struct {
		name       string
		onValue    any
		wantEvents map[string]bool
		checkTypes map[string][]string
		wantErr    bool
	}{
		{
			name: "workflow_dispatch with null config",
			onValue: map[string]any{
				"workflow_dispatch": nil,
			},
			wantEvents: map[string]bool{"workflow_dispatch": true},
			wantErr:    false,
		},
		{
			name: "pull_request with types",
			onValue: map[string]any{
				"pull_request": map[string]any{
					"types": []any{"opened", "synchronize"},
				},
			},
			wantEvents: map[string]bool{"pull_request": true},
			checkTypes: map[string][]string{
				"pull_request": {"opened", "synchronize"},
			},
			wantErr: false,
		},
		{
			name: "issues with types",
			onValue: map[string]any{
				"issues": map[string]any{
					"types": []any{"opened", "closed"},
				},
			},
			wantEvents: map[string]bool{"issues": true},
			checkTypes: map[string][]string{
				"issues": {"opened", "closed"},
			},
			wantErr: false,
		},
		{
			name: "push with branches",
			onValue: map[string]any{
				"push": map[string]any{
					"branches": []any{"main", "develop"},
				},
			},
			wantEvents: map[string]bool{"push": true},
			wantErr:    false,
		},
		{
			name: "multiple events",
			onValue: map[string]any{
				"push": map[string]any{
					"branches": []any{"main"},
				},
				"pull_request": map[string]any{
					"types": []any{"opened"},
				},
			},
			wantEvents: map[string]bool{"push": true, "pull_request": true},
			checkTypes: map[string][]string{
				"pull_request": {"opened"},
			},
			wantErr: false,
		},
		{
			name: "schedule event",
			onValue: map[string]any{
				"schedule": []any{
					map[string]any{"cron": "0 0 * * *"},
				},
			},
			wantEvents: map[string]bool{"schedule": true},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseTrigger(tt.onValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			for eventName, shouldExist := range tt.wantEvents {
				if config.HasEvent(eventName) != shouldExist {
					t.Errorf("ParseTrigger() HasEvent(%s) = %v, want %v", eventName, config.HasEvent(eventName), shouldExist)
				}
			}

			// Check types if specified
			for eventName, expectedTypes := range tt.checkTypes {
				eventConfig, exists := config.Events[eventName]
				if !exists {
					t.Errorf("ParseTrigger() event %s not found", eventName)
					continue
				}

				if len(eventConfig.Types) != len(expectedTypes) {
					t.Errorf("ParseTrigger() event %s types count = %d, want %d", eventName, len(eventConfig.Types), len(expectedTypes))
					continue
				}

				for i, expectedType := range expectedTypes {
					if i >= len(eventConfig.Types) || eventConfig.Types[i] != expectedType {
						t.Errorf("ParseTrigger() event %s types[%d] = %v, want %v", eventName, i, eventConfig.Types[i], expectedType)
					}
				}
			}
		})
	}
}

func TestParseTrigger_CommandTrigger(t *testing.T) {
	tests := []struct {
		name           string
		onValue        any
		wantCommand    bool
		wantCommandName string
		wantEvents     []string
		wantErr        bool
	}{
		{
			name: "simple command trigger",
			onValue: map[string]any{
				"command": map[string]any{
					"name": "bot",
				},
			},
			wantCommand:    true,
			wantCommandName: "bot",
			wantEvents:     nil,
			wantErr:        false,
		},
		{
			name: "command with events",
			onValue: map[string]any{
				"command": map[string]any{
					"name":   "helper",
					"events": []any{"issues", "issue_comment"},
				},
			},
			wantCommand:    true,
			wantCommandName: "helper",
			wantEvents:     []string{"issues", "issue_comment"},
			wantErr:        false,
		},
		{
			name: "command as string",
			onValue: map[string]any{
				"command": "mybot",
			},
			wantCommand:    true,
			wantCommandName: "mybot",
			wantEvents:     nil,
			wantErr:        false,
		},
		{
			name: "command with other events",
			onValue: map[string]any{
				"command": map[string]any{
					"name": "review",
				},
				"workflow_dispatch": nil,
			},
			wantCommand:    true,
			wantCommandName: "review",
			wantEvents:     nil,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseTrigger(tt.onValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if config.HasCommand() != tt.wantCommand {
				t.Errorf("ParseTrigger() HasCommand() = %v, want %v", config.HasCommand(), tt.wantCommand)
			}

			if config.GetCommandName() != tt.wantCommandName {
				t.Errorf("ParseTrigger() GetCommandName() = %v, want %v", config.GetCommandName(), tt.wantCommandName)
			}

			commandEvents := config.GetCommandEvents()
			if len(commandEvents) != len(tt.wantEvents) {
				t.Errorf("ParseTrigger() GetCommandEvents() length = %d, want %d", len(commandEvents), len(tt.wantEvents))
			} else {
				for i, ev := range tt.wantEvents {
					if commandEvents[i] != ev {
						t.Errorf("ParseTrigger() GetCommandEvents()[%d] = %v, want %v", i, commandEvents[i], ev)
					}
				}
			}
		})
	}
}

func TestParseTrigger_ReactionAndStopAfter(t *testing.T) {
	tests := []struct {
		name         string
		onValue      any
		wantReaction string
		wantStopAfter string
		wantErr      bool
	}{
		{
			name: "with reaction",
			onValue: map[string]any{
				"issues":   map[string]any{"types": []any{"opened"}},
				"reaction": "eyes",
			},
			wantReaction: "eyes",
			wantErr:      false,
		},
		{
			name: "with stop-after",
			onValue: map[string]any{
				"workflow_dispatch": nil,
				"stop-after":        "2024-12-31 23:59:59",
			},
			wantStopAfter: "2024-12-31 23:59:59",
			wantErr:       false,
		},
		{
			name: "with both reaction and stop-after",
			onValue: map[string]any{
				"issues":     map[string]any{"types": []any{"opened"}},
				"reaction":   "rocket",
				"stop-after": "+24h",
			},
			wantReaction:  "rocket",
			wantStopAfter: "+24h",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseTrigger(tt.onValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if config.Reaction != tt.wantReaction {
				t.Errorf("ParseTrigger() Reaction = %v, want %v", config.Reaction, tt.wantReaction)
			}

			if config.StopAfter != tt.wantStopAfter {
				t.Errorf("ParseTrigger() StopAfter = %v, want %v", config.StopAfter, tt.wantStopAfter)
			}
		})
	}
}

func TestParseTrigger_Errors(t *testing.T) {
	tests := []struct {
		name    string
		onValue any
		wantErr bool
	}{
		{
			name:    "nil value",
			onValue: nil,
			wantErr: true,
		},
		{
			name:    "invalid type - number",
			onValue: 123,
			wantErr: true,
		},
		{
			name:    "invalid type - boolean",
			onValue: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTrigger(tt.onValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTrigger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTriggerConfig_HasEvent(t *testing.T) {
	config := &TriggerConfig{
		Events: map[string]EventConfig{
			"push":              {},
			"pull_request":      {},
			"workflow_dispatch": {},
		},
	}

	tests := []struct {
		eventName string
		want      bool
	}{
		{"push", true},
		{"pull_request", true},
		{"workflow_dispatch", true},
		{"issues", false},
		{"schedule", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventName, func(t *testing.T) {
			if got := config.HasEvent(tt.eventName); got != tt.want {
				t.Errorf("HasEvent(%s) = %v, want %v", tt.eventName, got, tt.want)
			}
		})
	}

	// Test nil config
	var nilConfig *TriggerConfig
	if nilConfig.HasEvent("push") {
		t.Error("nil TriggerConfig should return false for HasEvent")
	}
}

func TestTriggerConfig_ToYAML(t *testing.T) {
	tests := []struct {
		name    string
		config  *TriggerConfig
		wantErr bool
	}{
		{
			name: "simple trigger",
			config: &TriggerConfig{
				Simple: "push",
				Events: map[string]EventConfig{
					"push": {},
				},
			},
			wantErr: false,
		},
		{
			name: "trigger with raw map",
			config: &TriggerConfig{
				Raw: map[string]any{
					"workflow_dispatch": nil,
				},
				Events: map[string]EventConfig{
					"workflow_dispatch": {},
				},
			},
			wantErr: false,
		},
		{
			name: "trigger with command",
			config: &TriggerConfig{
				Raw: map[string]any{
					"command": map[string]any{
						"name": "bot",
					},
				},
				Command: &CommandTriggerConfig{
					Name: "bot",
				},
				Events: map[string]EventConfig{},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := tt.config.ToYAML()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.config == nil && yaml != "" {
				t.Error("ToYAML() for nil config should return empty string")
			}
		})
	}
}

func TestParseEventConfig(t *testing.T) {
	tests := []struct {
		name       string
		eventValue any
		wantTypes  []string
		wantErr    bool
	}{
		{
			name:       "nil event",
			eventValue: nil,
			wantTypes:  nil,
			wantErr:    false,
		},
		{
			name: "event with types array",
			eventValue: map[string]any{
				"types": []any{"opened", "closed"},
			},
			wantTypes: []string{"opened", "closed"},
			wantErr:   false,
		},
		{
			name: "event with branches",
			eventValue: map[string]any{
				"branches": []any{"main", "develop"},
			},
			wantTypes: nil,
			wantErr:   false,
		},
		{
			name:       "invalid event type - string",
			eventValue: "invalid",
			wantTypes:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parseEventConfig(tt.eventValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEventConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if len(config.Types) != len(tt.wantTypes) {
				t.Errorf("parseEventConfig() Types length = %d, want %d", len(config.Types), len(tt.wantTypes))
			}
		})
	}
}

func TestParseStringArray(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{
			name:  "string array from []any",
			value: []any{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "string array from []string",
			value: []string{"x", "y", "z"},
			want:  []string{"x", "y", "z"},
		},
		{
			name:  "single string",
			value: "single",
			want:  []string{"single"},
		},
		{
			name:  "empty array",
			value: []any{},
			want:  []string{},
		},
		{
			name:  "nil value",
			value: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStringArray(tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("parseStringArray() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("parseStringArray()[%d] = %v, want %v", i, got[i], v)
				}
			}
		})
	}
}
