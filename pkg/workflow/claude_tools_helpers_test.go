package workflow

import "testing"

func TestHasBashWildcard(t *testing.T) {
	tests := []struct {
		name     string
		commands []any
		want     bool
	}{
		{name: "no wildcard", commands: []any{"jq", "sed"}, want: false},
		{name: "star wildcard", commands: []any{"*"}, want: true},
		{name: "colon star wildcard", commands: []any{":*"}, want: true},
		{name: "non-string values", commands: []any{1, true}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBashWildcard(tt.commands); got != tt.want {
				t.Fatalf("hasBashWildcard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeSandboxWritablePattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantOkay bool
	}{
		{name: "absolute directory path", input: "/tmp/cache", want: "/tmp/cache/*", wantOkay: true},
		{name: "absolute glob path", input: "/tmp/cache/*", want: "/tmp/cache/*", wantOkay: true},
		{name: "trim whitespace", input: "  /tmp/cache  ", want: "/tmp/cache/*", wantOkay: true},
		{name: "relative path rejected", input: "tmp/cache", want: "", wantOkay: false},
		{name: "empty path rejected", input: "  ", want: "", wantOkay: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOkay := normalizeSandboxWritablePattern(tt.input)
			if got != tt.want || gotOkay != tt.wantOkay {
				t.Fatalf("normalizeSandboxWritablePattern() = (%q, %v), want (%q, %v)", got, gotOkay, tt.want, tt.wantOkay)
			}
		})
	}
}
