package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPullRequestTargetTrigger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		frontmatter map[string]any
		want        bool
	}{
		{
			name:        "no on key",
			frontmatter: map[string]any{},
			want:        false,
		},
		{
			name:        "on key nil map",
			frontmatter: map[string]any{"on": nil},
			want:        false,
		},
		{
			name:        "on map with pull_request_target",
			frontmatter: map[string]any{"on": map[string]any{"pull_request_target": nil}},
			want:        true,
		},
		{
			name:        "on map without pull_request_target",
			frontmatter: map[string]any{"on": map[string]any{"push": nil}},
			want:        false,
		},
		{
			name:        "on map empty",
			frontmatter: map[string]any{"on": map[string]any{}},
			want:        false,
		},
		{
			name:        "on []any with pull_request_target string",
			frontmatter: map[string]any{"on": []any{"push", "pull_request_target"}},
			want:        true,
		},
		{
			name:        "on []any with pull_request_target and whitespace",
			frontmatter: map[string]any{"on": []any{"  pull_request_target  "}},
			want:        true,
		},
		{
			name:        "on []any without pull_request_target",
			frontmatter: map[string]any{"on": []any{"push", "issues"}},
			want:        false,
		},
		{
			name:        "on []any with non-string entries",
			frontmatter: map[string]any{"on": []any{123, true, nil}},
			want:        false,
		},
		{
			name:        "on []any empty",
			frontmatter: map[string]any{"on": []any{}},
			want:        false,
		},
		{
			name:        "on []string with pull_request_target",
			frontmatter: map[string]any{"on": []string{"push", "pull_request_target"}},
			want:        true,
		},
		{
			name:        "on []string without pull_request_target",
			frontmatter: map[string]any{"on": []string{"push"}},
			want:        false,
		},
		{
			name:        "on []string with whitespace around match",
			frontmatter: map[string]any{"on": []string{" pull_request_target "}},
			want:        true,
		},
		{
			name:        "on string exact match",
			frontmatter: map[string]any{"on": "pull_request_target"},
			want:        true,
		},
		{
			name:        "on string with surrounding whitespace",
			frontmatter: map[string]any{"on": "  pull_request_target  "},
			want:        true,
		},
		{
			name:        "on string non-matching",
			frontmatter: map[string]any{"on": "push"},
			want:        false,
		},
		{
			name:        "on unsupported type (int)",
			frontmatter: map[string]any{"on": 42},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasPullRequestTargetTrigger(tt.frontmatter)
			assert.Equal(t, tt.want, got, "hasPullRequestTargetTrigger(%#v)", tt.frontmatter)
		})
	}
}
