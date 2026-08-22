//go:build !integration

package workflow

import (
	"reflect"
	"testing"
)

func TestParseEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		wantTypes []string
		wantOK    bool
	}{
		{"nil value", nil, nil, false},
		{"string slice", []string{"opened", "closed"}, []string{"opened", "closed"}, true},
		{"empty string slice", []string{}, []string{}, true},
		{"any slice of strings", []any{"opened", "reopened"}, []string{"opened", "reopened"}, true},
		{"empty any slice", []any{}, []string{}, true},
		{"any slice with non-string entry", []any{"opened", 42}, nil, false},
		{"any slice with nil entry", []any{"opened", nil}, nil, false},
		{"plain string not a slice", "opened", nil, false},
		{"integer value", 42, nil, false},
		{"map value", map[string]any{"a": "b"}, nil, false},
		{"bool value", true, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTypes, gotOK := parseEventTypes(tt.value)
			if gotOK != tt.wantOK {
				t.Fatalf("parseEventTypes(%#v) ok = %v, want %v", tt.value, gotOK, tt.wantOK)
			}
			if !reflect.DeepEqual(gotTypes, tt.wantTypes) {
				t.Errorf("parseEventTypes(%#v) types = %#v, want %#v", tt.value, gotTypes, tt.wantTypes)
			}
		})
	}
}
