//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRepositoryOnlyWorkflowSpec(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want bool
	}{
		{
			name: "repo only",
			spec: "owner/repo",
			want: true,
		},
		{
			name: "repo only with version",
			spec: "owner/repo@v1.0.0",
			want: true,
		},
		{
			name: "repo with workflow path",
			spec: "owner/repo/workflow",
			want: false,
		},
		{
			name: "repo with workflow path and version",
			spec: "owner/repo/workflow@main",
			want: false,
		},
		{
			name: "github url",
			spec: "https://github.com/owner/repo/blob/main/.github/workflows/test.md",
			want: false,
		},
		{
			name: "local path",
			spec: "./.github/workflows/test.md",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRepositoryOnlyWorkflowSpec(tt.spec)
			assert.Equal(t, tt.want, got, "isRepositoryOnlyWorkflowSpec(%q) should return %v", tt.spec, tt.want)
		})
	}
}
