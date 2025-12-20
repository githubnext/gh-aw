package workflow

import (
	"testing"
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
