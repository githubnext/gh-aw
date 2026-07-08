package workflow

import (
	"math"
	"testing"
)

func TestParseBoundedIntField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   int
		wantOK bool
	}{
		{name: "missing field", input: nil, want: 0, wantOK: false},
		{name: "int", input: 7, want: 7, wantOK: true},
		{name: "zero", input: 0, want: 0, wantOK: false},
		{name: "negative", input: -1, want: 0, wantOK: false},
		{name: "uint64 clamp", input: uint64(math.MaxUint64), want: math.MaxInt, wantOK: true},
		{name: "float truncate", input: 12.75, want: 12, wantOK: true},
		{name: "float nan", input: math.NaN(), want: 0, wantOK: false},
		{name: "float inf", input: math.Inf(1), want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configMap := map[string]any{}
			if tt.name != "missing field" {
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
