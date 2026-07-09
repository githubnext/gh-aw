//go:build !integration

package workflow

import (
	"math"
	"testing"
)

func TestParseBoundedIntField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		omitFromMap bool
		want        int
		wantOK      bool
	}{
		{name: "missing field", omitFromMap: true, want: 0, wantOK: false},
		{name: "int", input: 7, want: 7, wantOK: true},
		{name: "zero", input: 0, want: 0, wantOK: false},
		{name: "negative", input: -1, want: 0, wantOK: false},
		{name: "int64", input: int64(42), want: 42, wantOK: true},
		{name: "int64 clamp", input: int64(math.MaxInt64), want: math.MaxInt, wantOK: true},
		{name: "int64 non-positive", input: int64(-1), want: 0, wantOK: false},
		{name: "uint64 clamp", input: uint64(math.MaxUint64), want: math.MaxInt, wantOK: true},
		{name: "float truncate", input: 12.75, want: 12, wantOK: true},
		{name: "float nan", input: math.NaN(), want: 0, wantOK: false},
		{name: "float inf", input: math.Inf(1), want: 0, wantOK: false},
		{name: "string input", input: "1024", want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configMap := map[string]any{}
			if !tt.omitFromMap {
				configMap["field"] = tt.input
			}

			got, ok := parseBoundedIntField(configMap, "field", safeOutputsConfigLog)
			if ok != tt.wantOK {
				t.Fatalf("parseBoundedIntField() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("parseBoundedIntField() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseBoundedIntFieldOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       map[string]any
		defaultValue int
		want         int
	}{
		{
			name:         "present valid value overrides default",
			config:       map[string]any{"k": 10},
			defaultValue: 99,
			want:         10,
		},
		{
			name:         "absent key returns default",
			config:       map[string]any{},
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "invalid value returns default",
			config:       map[string]any{"k": 0},
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "zero default remains zero when absent",
			config:       map[string]any{},
			defaultValue: 0,
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseBoundedIntFieldOrDefault(tt.config, "k", tt.defaultValue, safeOutputsConfigLog)
			if got != tt.want {
				t.Fatalf("parseBoundedIntFieldOrDefault() = %d, want %d", got, tt.want)
			}
		})
	}
}
