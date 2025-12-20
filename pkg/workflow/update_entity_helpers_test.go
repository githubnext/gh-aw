package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/logger"
)

func TestParseUpdateEntityBoolField(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]any
		fieldName string
		mode      FieldParsingMode
		wantNil   bool
		wantValue bool // Only checked if wantNil is false
	}{
		// FieldParsingKeyExistence mode tests
		{
			name:      "key existence mode: key present with nil value",
			configMap: map[string]any{"title": nil},
			fieldName: "title",
			mode:      FieldParsingKeyExistence,
			wantNil:   false, // Should return non-nil pointer
			wantValue: false, // Default bool value
		},
		{
			name:      "key existence mode: key present with empty value",
			configMap: map[string]any{"title": ""},
			fieldName: "title",
			mode:      FieldParsingKeyExistence,
			wantNil:   false,
			wantValue: false,
		},
		{
			name:      "key existence mode: key not present",
			configMap: map[string]any{"other": true},
			fieldName: "title",
			mode:      FieldParsingKeyExistence,
			wantNil:   true,
		},
		{
			name:      "key existence mode: nil config map",
			configMap: nil,
			fieldName: "title",
			mode:      FieldParsingKeyExistence,
			wantNil:   true,
		},
		{
			name:      "key existence mode: empty config map",
			configMap: map[string]any{},
			fieldName: "title",
			mode:      FieldParsingKeyExistence,
			wantNil:   true,
		},

		// FieldParsingBoolValue mode tests
		{
			name:      "bool value mode: true value",
			configMap: map[string]any{"title": true},
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   false,
			wantValue: true,
		},
		{
			name:      "bool value mode: false value",
			configMap: map[string]any{"title": false},
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   false,
			wantValue: false,
		},
		{
			name:      "bool value mode: nil value (not a bool)",
			configMap: map[string]any{"title": nil},
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   true, // Non-bool values return nil
		},
		{
			name:      "bool value mode: string value (not a bool)",
			configMap: map[string]any{"title": "true"},
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   true,
		},
		{
			name:      "bool value mode: key not present",
			configMap: map[string]any{"other": true},
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   true,
		},
		{
			name:      "bool value mode: nil config map",
			configMap: nil,
			fieldName: "title",
			mode:      FieldParsingBoolValue,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUpdateEntityBoolField(tt.configMap, tt.fieldName, tt.mode)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil result, got nil")
				} else if *result != tt.wantValue {
					t.Errorf("Expected value %v, got %v", tt.wantValue, *result)
				}
			}
		})
	}
}

func TestParseUpdateEntityBoolFieldFieldNames(t *testing.T) {
	// Test that different field names work correctly
	configMap := map[string]any{
		"title":  nil,
		"body":   nil,
		"status": nil,
		"labels": nil,
	}

	fieldNames := []string{"title", "body", "status", "labels"}
	for _, fieldName := range fieldNames {
		t.Run(fieldName, func(t *testing.T) {
			result := parseUpdateEntityBoolField(configMap, fieldName, FieldParsingKeyExistence)
			if result == nil {
				t.Errorf("Expected non-nil result for field %s", fieldName)
			}
		})
	}
}

func TestParseUpdateEntityConfigWithFields(t *testing.T) {
	tests := []struct {
		name         string
		outputMap    map[string]any
		opts         UpdateEntityParseOptions
		wantNil      bool
		validateFunc func(*testing.T, *UpdateEntityConfig)
	}{
		{
			name: "basic config with no fields",
			outputMap: map[string]any{
				"update-test": map[string]any{
					"max": 2,
				},
			},
			opts: UpdateEntityParseOptions{
				EntityType: UpdateEntityIssue,
				ConfigKey:  "update-test",
				Logger:     logger.New("test"),
				Fields:     nil,
			},
			wantNil: false,
			validateFunc: func(t *testing.T, cfg *UpdateEntityConfig) {
				if cfg.Max != 2 {
					t.Errorf("Expected max=2, got %d", cfg.Max)
				}
			},
		},
		{
			name: "config with bool fields using key existence mode",
			outputMap: map[string]any{
				"update-test": map[string]any{
					"max":   3,
					"title": nil,
					"body":  nil,
				},
			},
			opts: func() UpdateEntityParseOptions {
				var title, body *bool
				return UpdateEntityParseOptions{
					EntityType: UpdateEntityIssue,
					ConfigKey:  "update-test",
					Logger:     logger.New("test"),
					Fields: []UpdateEntityFieldSpec{
						{Name: "title", Mode: FieldParsingKeyExistence, Dest: &title},
						{Name: "body", Mode: FieldParsingKeyExistence, Dest: &body},
					},
				}
			}(),
			wantNil: false,
			validateFunc: func(t *testing.T, cfg *UpdateEntityConfig) {
				if cfg.Max != 3 {
					t.Errorf("Expected max=3, got %d", cfg.Max)
				}
			},
		},
		{
			name: "config with custom parser",
			outputMap: map[string]any{
				"update-test": map[string]any{
					"max":    1,
					"custom": "value",
				},
			},
			opts: UpdateEntityParseOptions{
				EntityType: UpdateEntityIssue,
				ConfigKey:  "update-test",
				Logger:     logger.New("test"),
				Fields:     nil,
				CustomParser: func(cm map[string]any) {
					// Custom parser just demonstrates it runs
					_ = cm
				},
			},
			wantNil: false,
			validateFunc: func(t *testing.T, cfg *UpdateEntityConfig) {
				if cfg.Max != 1 {
					t.Errorf("Expected max=1, got %d", cfg.Max)
				}
			},
		},
		{
			name: "missing config key returns nil",
			outputMap: map[string]any{
				"other-key": map[string]any{},
			},
			opts: UpdateEntityParseOptions{
				EntityType: UpdateEntityIssue,
				ConfigKey:  "update-test",
				Logger:     logger.New("test"),
				Fields:     nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			result, _ := compiler.parseUpdateEntityConfigWithFields(tt.outputMap, tt.opts)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil result, got nil")
				} else if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}
