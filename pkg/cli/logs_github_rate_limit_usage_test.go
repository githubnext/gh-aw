//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGitHubRateLimitsFileBasic verifies that a well-formed JSONL file is parsed
// correctly and produces the expected aggregated statistics.
func TestParseGitHubRateLimitsFileBasic(t *testing.T) {
	content := `{"timestamp":"2026-04-05T08:00:00.000Z","source":"rate_limit_api","operation":"check_rate_limit_start","resource":"core","limit":5000,"remaining":4900,"used":100,"reset":"2026-04-05T09:00:00.000Z"}
{"timestamp":"2026-04-05T08:01:00.000Z","source":"response_headers","operation":"issues.listComments","resource":"core","limit":5000,"remaining":4890,"used":110,"reset":"2026-04-05T09:00:00.000Z"}
{"timestamp":"2026-04-05T08:02:00.000Z","source":"response_headers","operation":"issues.createComment","resource":"core","limit":5000,"remaining":4880,"used":120,"reset":"2026-04-05T09:00:00.000Z"}
{"timestamp":"2026-04-05T08:03:00.000Z","source":"response_headers","operation":"search.repos","resource":"search","limit":30,"remaining":28,"used":2,"reset":"2026-04-05T08:04:00.000Z"}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600), "should write test JSONL file")

	usage, err := parseGitHubRateLimitsFile(path)
	require.NoError(t, err, "parseGitHubRateLimitsFile should not return an error")
	require.NotNil(t, usage, "usage should not be nil")

	assert.Equal(t, 3, usage.TotalRequestsMade, "should count 3 response_headers entries")

	// Core resource: 2 calls, consumed = lastUsed(120) - firstUsed(110) = 10
	assert.Equal(t, 10, usage.CoreConsumed, "core quota consumed should be 10")
	assert.Equal(t, 4880, usage.CoreRemaining, "core remaining should match last entry")
	assert.Equal(t, 5000, usage.CoreLimit, "core limit should be 5000")

	// Should have entries for both resources
	require.Len(t, usage.Resources, 2, "should have resource entries for core and search")
}

// TestParseGitHubRateLimitsFileEmpty verifies that an empty file returns nil without error.
func TestParseGitHubRateLimitsFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600), "should create empty file")

	usage, err := parseGitHubRateLimitsFile(path)
	require.NoError(t, err, "empty file should not return an error")
	assert.Nil(t, usage, "empty file should return nil usage")
}

// TestParseGitHubRateLimitsFileMissingResource verifies that entries without a resource
// field are bucketed under "core".
func TestParseGitHubRateLimitsFileMissingResource(t *testing.T) {
	content := `{"timestamp":"2026-04-05T08:01:00.000Z","source":"response_headers","operation":"fetch","limit":5000,"remaining":4999,"used":1}
{"timestamp":"2026-04-05T08:02:00.000Z","source":"response_headers","operation":"fetch","limit":5000,"remaining":4998,"used":2}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600), "should write test JSONL file")

	usage, err := parseGitHubRateLimitsFile(path)
	require.NoError(t, err, "should not return an error")
	require.NotNil(t, usage, "usage should not be nil")

	assert.Equal(t, 2, usage.TotalRequestsMade, "should count 2 requests")
	assert.Equal(t, 1, usage.CoreConsumed, "core consumed should be 1 (lastUsed - firstUsed)")
}

// TestParseGitHubRateLimitsFileWindowReset verifies that a rate-limit window reset during
// a run is handled gracefully by falling back to lastUsed as the consumption estimate.
func TestParseGitHubRateLimitsFileWindowReset(t *testing.T) {
	// Simulate a window reset: used goes from 4900 down to 5 (new window)
	content := `{"timestamp":"2026-04-05T08:01:00.000Z","source":"response_headers","operation":"issues.get","resource":"core","limit":5000,"remaining":100,"used":4900,"reset":"2026-04-05T09:00:00.000Z"}
{"timestamp":"2026-04-05T09:00:05.000Z","source":"response_headers","operation":"issues.get","resource":"core","limit":5000,"remaining":4995,"used":5,"reset":"2026-04-05T10:00:00.000Z"}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600), "should write test JSONL file")

	usage, err := parseGitHubRateLimitsFile(path)
	require.NoError(t, err, "should not return an error")
	require.NotNil(t, usage, "usage should not be nil")

	assert.Equal(t, 2, usage.TotalRequestsMade, "should count 2 requests")
	// Window reset: lastUsed(5) < firstUsed(4900), falls back to lastUsed
	assert.Equal(t, 5, usage.CoreConsumed, "should fall back to lastUsed on window reset")
}

// TestParseGitHubRateLimitsFileOnlyAPISnapshots verifies that a file containing only
// rate_limit_api snapshot entries (no response_headers) returns zero requests made but
// still captures remaining/limit for context.
func TestParseGitHubRateLimitsFileOnlyAPISnapshots(t *testing.T) {
	content := `{"timestamp":"2026-04-05T08:00:00.000Z","source":"rate_limit_api","operation":"startup","resource":"core","limit":5000,"remaining":4850,"used":150,"reset":"2026-04-05T09:00:00.000Z"}
{"timestamp":"2026-04-05T08:30:00.000Z","source":"rate_limit_api","operation":"shutdown","resource":"core","limit":5000,"remaining":4840,"used":160,"reset":"2026-04-05T09:00:00.000Z"}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600), "should write test JSONL file")

	usage, err := parseGitHubRateLimitsFile(path)
	require.NoError(t, err, "should not return an error")
	require.NotNil(t, usage, "usage should not be nil")

	assert.Equal(t, 0, usage.TotalRequestsMade, "should count 0 API calls from response_headers")
}

// TestFindGitHubRateLimitsFileAbsent verifies that findGitHubRateLimitsFile returns
// an empty string when the file does not exist.
func TestFindGitHubRateLimitsFileAbsent(t *testing.T) {
	dir := t.TempDir()
	result := findGitHubRateLimitsFile(dir)
	assert.Empty(t, result, "should return empty string when file is absent")
}

// TestFindGitHubRateLimitsFileRoot verifies that the file is found when placed at the
// root of the run directory (the standard location after artifact flattening).
func TestFindGitHubRateLimitsFileRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600), "should create file")

	result := findGitHubRateLimitsFile(dir)
	assert.Equal(t, path, result, "should find file at root of run directory")
}

// TestFindGitHubRateLimitsFileNested verifies that the file is found via walk when
// placed in a subdirectory.
func TestFindGitHubRateLimitsFileNested(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "activation")
	require.NoError(t, os.MkdirAll(subDir, 0750), "should create subdirectory")
	path := filepath.Join(subDir, "github_rate_limits.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600), "should create nested file")

	result := findGitHubRateLimitsFile(dir)
	assert.Equal(t, path, result, "should find file via directory walk")
}

// TestGitHubRateLimitUsageResourceRows verifies that ResourceRows returns rows sorted
// by requests made in descending order.
func TestGitHubRateLimitUsageResourceRows(t *testing.T) {
	usage := &GitHubRateLimitUsage{
		TotalRequestsMade: 15,
		Resources: []*GitHubRateLimitResourceUsage{
			{Resource: "search", RequestsMade: 3},
			{Resource: "core", RequestsMade: 10},
			{Resource: "graphql", RequestsMade: 2},
		},
	}

	rows := usage.ResourceRows()
	require.Len(t, rows, 3, "should return all resource rows")
	assert.Equal(t, "core", rows[0].Resource, "highest request count should be first")
	assert.Equal(t, "search", rows[1].Resource, "second highest should be second")
	assert.Equal(t, "graphql", rows[2].Resource, "lowest should be last")
}
