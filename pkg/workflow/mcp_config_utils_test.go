//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

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
