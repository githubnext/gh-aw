//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestDedentTrailingOnCommentBlock(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "no trailing comment – last line is real content",
			input: []string{"on:", "  issues:", "    types: [labeled]"},
			want:  []string{"on:", "  issues:", "    types: [labeled]"},
		},
		{
			name: "single indented trailing comment is dedented to column 0",
			input: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"  # roles: # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "multiple indented trailing comment lines are all dedented",
			input: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"  # roles: # Roles processed as role check in pre-activation job",
				"  # - admin # Roles processed as role check in pre-activation job",
				"  # - write # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
				"# - admin # Roles processed as role check in pre-activation job",
				"# - write # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "trailing blank lines after comment block are skipped; comment block is still dedented",
			input: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"  # roles: # Roles processed as role check in pre-activation job",
				"",
				"",
			},
			want: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
				"",
				"",
			},
		},
		{
			name: "trailing comment already at column 0 is unchanged",
			input: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"on:",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "middle comment block with real content after it is left untouched",
			input: []string{
				"on:",
				"  # forks: # Fork check applied elsewhere",
				"  issues:",
				"    types: [labeled]",
				"  # roles: # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"on:",
				"  # forks: # Fork check applied elsewhere",
				"  issues:",
				"    types: [labeled]",
				"# roles: # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "file consisting entirely of comments is not modified",
			input: []string{
				"# roles: # Roles processed as role check in pre-activation job",
				"# - admin # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"# roles: # Roles processed as role check in pre-activation job",
				"# - admin # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "deeply nested trailing block whose parent has no real children is dedented",
			input: []string{
				"on:",
				"  deployment_status:",
				"    # state: # State filtering compiled into if condition",
				"    # - error # State filtering compiled into if condition",
				"    # - failure # State filtering compiled into if condition",
			},
			want: []string{
				"on:",
				"  deployment_status:",
				"# state: # State filtering compiled into if condition",
				"# - error # State filtering compiled into if condition",
				"# - failure # State filtering compiled into if condition",
			},
		},
		{
			name: "deeply nested trailing block with a real sibling at its indent is left untouched",
			input: []string{
				"on:",
				"  pull_request:",
				"    types: [opened]",
				"    # forks: # Fork filtering applied via job conditions",
				"    # - octocat # Fork filtering applied via job conditions",
			},
			want: []string{
				"on:",
				"  pull_request:",
				"    types: [opened]",
				"    # forks: # Fork filtering applied via job conditions",
				"    # - octocat # Fork filtering applied via job conditions",
			},
		},
		{
			name: "tab-indented trailing comment is dedented",
			input: []string{
				"on:",
				"  issues:",
				"\t# roles: # Roles processed as role check in pre-activation job",
			},
			want: []string{
				"on:",
				"  issues:",
				"# roles: # Roles processed as role check in pre-activation job",
			},
		},
		{
			name: "mixed indentation (spaces then tabs) in trailing block is all stripped",
			input: []string{
				"on:",
				"  issues:",
				"  # roles: # comment",
				"\t# - admin # comment",
			},
			want: []string{
				"on:",
				"  issues:",
				"# roles: # comment",
				"# - admin # comment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a defensive copy so the function's in-place modification of the
			// input slice doesn't corrupt subsequent sub-test runs.
			input := make([]string, len(tt.input))
			copy(input, tt.input)

			got := dedentTrailingOnCommentBlock(input)

			if len(got) != len(tt.want) {
				t.Fatalf("dedentTrailingOnCommentBlock() returned %d lines, want %d\ngot:  %q\nwant: %q",
					len(got), len(tt.want),
					strings.Join(got, "\n"),
					strings.Join(tt.want, "\n"))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("line %d mismatch:\n  got:  %q\n  want: %q\nfull output:\n%s",
						i, got[i], tt.want[i], strings.Join(got, "\n"))
				}
			}
		})
	}
}
