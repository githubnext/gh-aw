//go:build !integration

package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunningAsMCPServer(t *testing.T) {
	origMCP := os.Getenv("GH_AW_MCP_SERVER")
	defer func() {
		os.Setenv("GH_AW_MCP_SERVER", origMCP)
	}()

	tests := []struct {
		name     string
		mcpEnv   string
		expected bool
	}{
		{
			name:     "not in MCP server mode",
			mcpEnv:   "",
			expected: false,
		},
		{
			name:     "in MCP server mode",
			mcpEnv:   "true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GH_AW_MCP_SERVER", tt.mcpEnv)
			result := isRunningAsMCPServer()
			if result != tt.expected {
				t.Errorf("isRunningAsMCPServer() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindLatestPublishedReleaseTag(t *testing.T) {
	tests := []struct {
		name     string
		releases []Release
		want     string
	}{
		{
			name: "returns first non-draft release tag",
			releases: []Release{
				{TagName: "v1.2.0-beta.1", Draft: false, Prerelease: true},
				{TagName: "v1.1.0", Draft: false, Prerelease: false},
			},
			want: "v1.2.0-beta.1",
		},
		{
			name: "skips draft releases",
			releases: []Release{
				{TagName: "v1.3.0", Draft: true},
				{TagName: "v1.2.0", Draft: false},
			},
			want: "v1.2.0",
		},
		{
			name: "skips empty tags",
			releases: []Release{
				{TagName: "", Draft: false},
				{TagName: "v1.0.0", Draft: false},
			},
			want: "v1.0.0",
		},
		{
			name: "returns empty when no published releases",
			releases: []Release{
				{TagName: "", Draft: true},
				{TagName: "", Draft: false},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLatestPublishedReleaseTag(tt.releases)
			assert.Equal(t, tt.want, got, "unexpected latest published release tag")
		})
	}
}
