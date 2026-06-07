package importinpututil

import "testing"

func TestResolvePathValue(t *testing.T) {
	inputs := map[string]any{
		"name": "alice",
		"config": map[string]any{
			"token": "abc",
		},
		"bad": "not-map",
	}

	tests := []struct {
		name  string
		path  string
		want  any
		found bool
	}{
		{name: "top level", path: "name", want: "alice", found: true},
		{name: "dotted path", path: "config.token", want: "abc", found: true},
		{name: "missing top level", path: "missing", found: false},
		{name: "missing dotted key", path: "config.missing", found: false},
		{name: "dotted non map", path: "bad.token", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolvePathValue(inputs, tt.path)
			if ok != tt.found {
				t.Fatalf("ResolvePathValue(%q) found = %v, want %v", tt.path, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Fatalf("ResolvePathValue(%q) = %#v, want %#v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatResolvedValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
		ok    bool
	}{
		{name: "scalar string", value: "hello", want: "hello", ok: true},
		{name: "scalar int", value: 42, want: "42", ok: true},
		{name: "slice any", value: []any{"a", 1}, want: `["a",1]`, ok: true},
		{name: "typed slice", value: []string{"x", "y"}, want: `["x","y"]`, ok: true},
		{name: "map any", value: map[string]any{"k": "v"}, want: `{"k":"v"}`, ok: true},
		{name: "typed map", value: map[string]string{"k": "v"}, want: `{"k":"v"}`, ok: true},
		{name: "nil value", value: nil, want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FormatResolvedValue(tt.value)
			if ok != tt.ok {
				t.Fatalf("FormatResolvedValue(%#v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("FormatResolvedValue(%#v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
