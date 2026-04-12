//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateArtifactSets(t *testing.T) {
	tests := []struct {
		name      string
		sets      []string
		expectErr bool
	}{
		{
			name:      "empty sets is valid",
			sets:      nil,
			expectErr: false,
		},
		{
			name:      "all is valid",
			sets:      []string{"all"},
			expectErr: false,
		},
		{
			name:      "activation is valid",
			sets:      []string{"activation"},
			expectErr: false,
		},
		{
			name:      "agent is valid",
			sets:      []string{"agent"},
			expectErr: false,
		},
		{
			name:      "mcp is valid",
			sets:      []string{"mcp"},
			expectErr: false,
		},
		{
			name:      "firewall is valid",
			sets:      []string{"firewall"},
			expectErr: false,
		},
		{
			name:      "detection is valid",
			sets:      []string{"detection"},
			expectErr: false,
		},
		{
			name:      "github-api is valid",
			sets:      []string{"github-api"},
			expectErr: false,
		},
		{
			name:      "multiple valid sets",
			sets:      []string{"agent", "mcp"},
			expectErr: false,
		},
		{
			name:      "unknown set returns error",
			sets:      []string{"unknown"},
			expectErr: true,
		},
		{
			name:      "mix of valid and unknown returns error",
			sets:      []string{"agent", "bad-set"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArtifactSets(tt.sets)
			if tt.expectErr {
				assert.Error(t, err, "Expected an error for sets: %v", tt.sets)
			} else {
				assert.NoError(t, err, "Expected no error for sets: %v", tt.sets)
			}
		})
	}
}

func TestResolveArtifactFilter(t *testing.T) {
	tests := []struct {
		name     string
		sets     []string
		expected []string // nil means "no filter" (download all)
	}{
		{
			name:     "nil sets returns nil filter",
			sets:     nil,
			expected: nil,
		},
		{
			name:     "empty sets returns nil filter",
			sets:     []string{},
			expected: nil,
		},
		{
			name:     "all returns nil filter",
			sets:     []string{"all"},
			expected: nil,
		},
		{
			name:     "all with other sets returns nil filter",
			sets:     []string{"agent", "all"},
			expected: nil,
		},
		{
			name:     "activation resolves to activation artifact",
			sets:     []string{"activation"},
			expected: []string{"activation"},
		},
		{
			name:     "agent resolves to agent artifact",
			sets:     []string{"agent"},
			expected: []string{"agent"},
		},
		{
			name:     "mcp resolves to agent artifact",
			sets:     []string{"mcp"},
			expected: []string{"agent"},
		},
		{
			name:     "firewall resolves to agent artifact",
			sets:     []string{"firewall"},
			expected: []string{"agent"},
		},
		{
			name:     "mcp and firewall both deduplicate to single agent",
			sets:     []string{"mcp", "firewall"},
			expected: []string{"agent"},
		},
		{
			name:     "detection resolves to detection artifact",
			sets:     []string{"detection"},
			expected: []string{"detection"},
		},
		{
			name:     "github-api resolves to activation and agent",
			sets:     []string{"github-api"},
			expected: []string{"activation", "agent"},
		},
		{
			name:     "multiple sets are merged and deduplicated",
			sets:     []string{"activation", "agent"},
			expected: []string{"activation", "agent"},
		},
		{
			name:     "github-api and agent deduplicates agent",
			sets:     []string{"github-api", "agent"},
			expected: []string{"activation", "agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveArtifactFilter(tt.sets)
			assert.Equal(t, tt.expected, result, "ResolveArtifactFilter(%v)", tt.sets)
		})
	}
}

func TestArtifactMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		artifact string
		filter   []string
		expected bool
	}{
		{
			name:     "nil filter matches everything",
			artifact: "agent",
			filter:   nil,
			expected: true,
		},
		{
			name:     "empty filter matches everything",
			artifact: "agent",
			filter:   []string{},
			expected: true,
		},
		{
			name:     "exact match",
			artifact: "agent",
			filter:   []string{"agent"},
			expected: true,
		},
		{
			name:     "no match",
			artifact: "detection",
			filter:   []string{"agent"},
			expected: false,
		},
		{
			name:     "prefixed match (workflow_call context)",
			artifact: "abc123-agent",
			filter:   []string{"agent"},
			expected: true,
		},
		{
			name:     "prefixed activation match",
			artifact: "deadbeef-activation",
			filter:   []string{"activation"},
			expected: true,
		},
		{
			name:     "prefix does not false-positive on partial names",
			artifact: "sub-agent-tools",
			filter:   []string{"agent"},
			expected: false,
		},
		{
			name:     "multi-filter any match succeeds",
			artifact: "firewall-audit-logs",
			filter:   []string{"agent", "firewall-audit-logs"},
			expected: true,
		},
		{
			name:     "firewall-audit-logs exact match",
			artifact: "firewall-audit-logs",
			filter:   []string{"firewall-audit-logs"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := artifactMatchesFilter(tt.artifact, tt.filter)
			assert.Equal(t, tt.expected, result, "artifactMatchesFilter(%q, %v)", tt.artifact, tt.filter)
		})
	}
}

func TestValidArtifactSetNames(t *testing.T) {
	names := ValidArtifactSetNames()
	require.NotEmpty(t, names, "ValidArtifactSetNames should return non-empty slice")

	expected := []string{"all", "activation", "agent", "detection", "firewall", "github-api", "mcp"}
	assert.ElementsMatch(t, expected, names, "ValidArtifactSetNames should contain all known sets")
}

func TestFindMissingFilterEntries(t *testing.T) {
	tests := []struct {
		name         string
		filter       []string
		existingDirs []string
		expected     []string
	}{
		{
			name:         "all entries present (exact match)",
			filter:       []string{"agent", "activation"},
			existingDirs: []string{"agent", "activation"},
			expected:     nil,
		},
		{
			name:         "all entries present (prefix match)",
			filter:       []string{"agent"},
			existingDirs: []string{"abc123-agent", "activation"},
			expected:     nil,
		},
		{
			name:         "one entry missing",
			filter:       []string{"agent", "firewall-audit-logs"},
			existingDirs: []string{"agent"},
			expected:     []string{"firewall-audit-logs"},
		},
		{
			name:         "all entries missing",
			filter:       []string{"agent", "firewall-audit-logs"},
			existingDirs: []string{},
			expected:     []string{"agent", "firewall-audit-logs"},
		},
		{
			name:         "prefix match does not false-positive on substring (suffix mismatch)",
			filter:       []string{"agent"},
			existingDirs: []string{"agent-output"},
			expected:     []string{"agent"},
		},
		{
			name:         "any-suffix directory matches filter entry (mirrors artifactMatchesFilter behavior)",
			filter:       []string{"agent"},
			existingDirs: []string{"super-agent"},
			// strings.HasSuffix("super-agent", "-agent") is true; intentional (consistent
			// with artifactMatchesFilter) — in practice only workflow_call hash-prefixed
			// directories appear in a run folder.
			expected: nil,
		},
		{
			name:         "firewall-audit-logs exact match found",
			filter:       []string{"firewall-audit-logs"},
			existingDirs: []string{"firewall-audit-logs"},
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, d := range tt.existingDirs {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, d), 0755), "failed to create test dir")
			}
			result := findMissingFilterEntries(tt.filter, dir)
			assert.Equal(t, tt.expected, result, "findMissingFilterEntries(%v, dir)", tt.filter)
		})
	}
}
