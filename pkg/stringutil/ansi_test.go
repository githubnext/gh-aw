//go:build !integration

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "plain text unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "basic color code",
			input: "\x1b[31mred text\x1b[0m",
			want:  "red text",
		},
		{
			name:  "bold color code",
			input: "\x1b[1;32mBold Green\x1b[0m",
			want:  "Bold Green",
		},
		{
			name:  "reset code only",
			input: "text\x1b[mmore",
			want:  "textmore",
		},
		{
			name:  "multiple ANSI sequences",
			input: "\x1b[31mdoes important\x1b[0m things\x1b[m",
			want:  "does important things",
		},
		{
			name:  "description with embedded ANSI",
			input: "This workflow \x1b[31mdoes important\x1b[0m things\x1b[m",
			want:  "This workflow does important things",
		},
		{
			name:  "bold ANSI in description",
			input: "Workflow with \x1b[1mANSI\x1b[0m codes",
			want:  "Workflow with ANSI codes",
		},
		{
			name:  "file path with ANSI color",
			input: "path/to/\x1b[32mfile1.md\x1b[0m",
			want:  "path/to/file1.md",
		},
		{
			name:  "stop-time with ANSI codes",
			input: "2026-12-31\x1b[31mT23:59:59Z\x1b[0m",
			want:  "2026-12-31T23:59:59Z",
		},
		{
			name:  "environment name with ANSI bold",
			input: "production-\x1b[1menv\x1b[0m",
			want:  "production-env",
		},
		{
			name:  "OSC sequence with BEL terminator",
			input: "before\x1b]0;title\x07after",
			want:  "beforeafter",
		},
		{
			name:  "OSC sequence with ST terminator",
			input: "before\x1b]0;title\x1b\\after",
			want:  "beforeafter",
		},
		{
			name:  "G0 character set selection",
			input: "before\x1b(Bafter",
			want:  "beforeafter",
		},
		{
			name:  "G1 character set selection",
			input: "before\x1b)0after",
			want:  "beforeafter",
		},
		{
			name:  "application keypad mode",
			input: "before\x1b=after",
			want:  "beforeafter",
		},
		{
			name:  "normal keypad mode",
			input: "before\x1b>after",
			want:  "beforeafter",
		},
		{
			name:  "reset sequence",
			input: "before\x1bcafter",
			want:  "beforeafter",
		},
		{
			name:  "cursor save (two-char sequence)",
			input: "before\x1b7after",
			want:  "beforeafter",
		},
		{
			name:  "cursor restore (two-char sequence)",
			input: "before\x1b8after",
			want:  "beforeafter",
		},
		{
			name:  "ESC at end of string",
			input: "text\x1b",
			want:  "text",
		},
		{
			name:  "ESC with no following character stripped",
			input: "text\x1b[",
			want:  "text",
		},
		{
			name:  "multiline text with ANSI codes",
			input: "Line 1 with \x1b[32mgreen\x1b[0m text\nLine 2 with \x1b[31mred\x1b[0m text",
			want:  "Line 1 with green text\nLine 2 with red text",
		},
		{
			name:  "256-color foreground",
			input: "\x1b[38;5;196mred256\x1b[0m",
			want:  "red256",
		},
		{
			name:  "256-color background",
			input: "\x1b[48;5;21mblue256bg\x1b[0m",
			want:  "blue256bg",
		},
		{
			name:  "hyperlink OSC sequence",
			input: "\x1b]8;;https://example.com\x07click\x1b]8;;\x07",
			want:  "click",
		},
		{
			name:  "no ESC but brackets preserved",
			input: "array[0] and [key]",
			want:  "array[0] and [key]",
		},
		{
			name:  "source path with ANSI",
			input: "\x1b[33mowner/repo\x1b[0m@v1.2.3",
			want:  "owner/repo@v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
