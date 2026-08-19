//go:build !integration

package cli

import (
	"reflect"
	"testing"
)

// TestExtractGitHubToolsets exercises all branches of extractGitHubToolsets:
// missing/malformed tools, github, and toolsets keys, plus each supported
// toolsets value type ([]string, []any, string) with trimming/filtering.
func TestExtractGitHubToolsets(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		wantToolsets []string
		wantOK       bool
	}{
		{
			name:         "no tools key",
			frontmatter:  map[string]any{},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name:         "tools is not a map",
			frontmatter:  map[string]any{"tools": "not-a-map"},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name:         "tools present but no github key",
			frontmatter:  map[string]any{"tools": map[string]any{}},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name:         "github value is not a map",
			frontmatter:  map[string]any{"tools": map[string]any{"github": "not-a-map"}},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name: "github present but no toolsets key",
			frontmatter: map[string]any{
				"tools": map[string]any{"github": map[string]any{}},
			},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name: "toolsets as []string",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": []string{"repos", " issues ", "", "  "},
					},
				},
			},
			wantToolsets: []string{"repos", "issues"},
			wantOK:       true,
		},
		{
			name: "toolsets as []any with strings",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": []any{"repos", " issues ", "", 42, nil},
					},
				},
			},
			wantToolsets: []string{"repos", "issues"},
			wantOK:       true,
		},
		{
			name: "toolsets as comma-separated string",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": "repos, issues,, ,pull_requests",
					},
				},
			},
			wantToolsets: []string{"repos", "issues", "pull_requests"},
			wantOK:       true,
		},
		{
			name: "toolsets as unsupported type",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": 123,
					},
				},
			},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name: "toolsets as empty []string yields no toolsets",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": []string{},
					},
				},
			},
			wantToolsets: nil,
			wantOK:       false,
		},
		{
			name: "toolsets as all-whitespace string yields no toolsets",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"toolsets": "  ,  , ",
					},
				},
			},
			wantToolsets: nil,
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractGitHubToolsets(tt.frontmatter)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !reflect.DeepEqual(got, tt.wantToolsets) {
				t.Fatalf("toolsets = %#v, want %#v", got, tt.wantToolsets)
			}
		})
	}
}
