//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSafeGitRefName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "simple branch name is safe", ref: "main", want: true},
		{name: "slash separated ref is safe", ref: "experiments/my-run", want: true},
		{name: "empty string is unsafe", ref: "", want: false},
		{name: "leading slash is unsafe", ref: "/main", want: false},
		{name: "trailing slash is unsafe", ref: "main/", want: false},
		{name: "trailing dot is unsafe", ref: "main.", want: false},
		{name: "double slash is unsafe", ref: "foo//bar", want: false},
		{name: "double dot sequence is unsafe", ref: "foo..bar", want: false},
		{name: "at-brace sequence is unsafe", ref: "foo@{bar}", want: false},
		{name: "backslash is unsafe", ref: `foo\bar`, want: false},
		{name: "empty path component is unsafe", ref: "foo//", want: false},
		{name: "component starting with dot is unsafe", ref: "foo/.bar", want: false},
		{name: "component ending with .lock is unsafe", ref: "foo/bar.lock", want: false},
		{name: "component with space is unsafe", ref: "foo bar", want: false},
		{name: "component with control char is unsafe", ref: "foo\tbar", want: false},
		{name: "component with tilde is unsafe", ref: "foo~1", want: false},
		{name: "component with caret is unsafe", ref: "foo^1", want: false},
		{name: "component with colon is unsafe", ref: "foo:bar", want: false},
		{name: "component with question mark is unsafe", ref: "foo?bar", want: false},
		{name: "component with asterisk is unsafe", ref: "foo*bar", want: false},
		{name: "component with open bracket is unsafe", ref: "foo[bar", want: false},
		{name: "component with DEL char is unsafe", ref: "foo\x7fbar", want: false},
		{name: "multi-level valid ref is safe", ref: "refs/heads/experiments/foo-bar_1", want: true},
		{name: "single dot component is unsafe", ref: "foo/./bar", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isSafeGitRefName(tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}

func FuzzIsSafeGitRefName(f *testing.F) {
	seeds := []string{
		"",
		"main",
		"experiments/foo",
		"/main",
		"main/",
		"main.",
		"foo//bar",
		"foo..bar",
		"foo@{bar}",
		`foo\bar`,
		"foo/.bar",
		"foo/bar.lock",
		"foo bar",
		"foo~1",
		"foo^1",
		"foo:bar",
		"foo?bar",
		"foo*bar",
		"foo[bar",
		"foo\x7fbar",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, ref string) {
		// isSafeGitRefName must never panic and must be deterministic (a pure
		// function of its input).
		got := isSafeGitRefName(ref)
		again := isSafeGitRefName(ref)
		assert.Equal(t, got, again)
	})
}
