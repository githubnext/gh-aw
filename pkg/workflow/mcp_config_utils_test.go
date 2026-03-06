//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestGetTypeString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "nil value",
			value: nil,
			want:  "null",
		},
		{
			name:  "int value",
			value: 42,
			want:  "number",
		},
		{
			name:  "int64 value",
			value: int64(100),
			want:  "number",
		},
		{
			name:  "float64 value",
			value: 3.14,
			want:  "number",
		},
		{
			name:  "float32 value",
			value: float32(2.71),
			want:  "number",
		},
		{
			name:  "boolean true",
			value: true,
			want:  "boolean",
		},
		{
			name:  "boolean false",
			value: false,
			want:  "boolean",
		},
		{
			name:  "string value",
			value: "hello world",
			want:  "string",
		},
		{
			name:  "empty string",
			value: "",
			want:  "string",
		},
		{
			name: "object (map[string]any)",
			value: map[string]any{
				"key": "value",
			},
			want: "object",
		},
		{
			name:  "empty object",
			value: map[string]any{},
			want:  "object",
		},
		{
			name:  "array of strings",
			value: []string{"a", "b", "c"},
			want:  "array",
		},
		{
			name:  "array of ints",
			value: []int{1, 2, 3},
			want:  "array",
		},
		{
			name:  "array of any",
			value: []any{"mixed", 123, true},
			want:  "array",
		},
		{
			name:  "empty array",
			value: []string{},
			want:  "array",
		},
		{
			name:  "array of objects",
			value: []map[string]any{{"key": "value"}},
			want:  "array",
		},
		{
			name: "unknown type (struct)",
			value: struct {
				Name string
			}{Name: "test"},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTypeString(tt.value)
			if got != tt.want {
				t.Errorf("getTypeString(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestWriteArgsToYAMLInline(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no args",
			args: []string{},
			want: "",
		},
		{
			name: "single simple arg",
			args: []string{"--verbose"},
			want: `, "--verbose"`,
		},
		{
			name: "multiple simple args",
			args: []string{"--verbose", "--debug"},
			want: `, "--verbose", "--debug"`,
		},
		{
			name: "args with spaces",
			args: []string{"--message", "hello world"},
			want: `, "--message", "hello world"`,
		},
		{
			name: "args with quotes",
			args: []string{"--text", `say "hello"`},
			want: `, "--text", "say \"hello\""`,
		},
		{
			name: "args with special characters",
			args: []string{"--path", "/tmp/test\n\t"},
			want: `, "--path", "/tmp/test\n\t"`,
		},
		{
			name: "args with backslashes",
			args: []string{"--path", `C:\Windows\System32`},
			want: `, "--path", "C:\\Windows\\System32"`,
		},
		{
			name: "empty string arg",
			args: []string{""},
			want: `, ""`,
		},
		{
			name: "unicode args",
			args: []string{"--text", "Hello 世界 🌍"},
			want: `, "--text", "Hello 世界 🌍"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			writeArgsToYAMLInline(&builder, tt.args)
			got := builder.String()
			if got != tt.want {
				t.Errorf("writeArgsToYAMLInline() = %q, want %q", got, tt.want)
			}
		})
	}
}
