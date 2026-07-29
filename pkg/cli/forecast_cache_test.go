//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecastAICCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	const runID int64 = 12345

	// Miss when no cache file exists.
	if _, ok := loadForecastAICCache(dir, runID); ok {
		t.Fatalf("expected cache miss for empty dir")
	}

	// Save then load returns the same value.
	saveForecastAICCache(dir, runID, 42.5)
	got, ok := loadForecastAICCache(dir, runID)
	require.True(t, ok, "expected cache hit after save")
	assert.InDelta(t, 42.5, got, 1e-9)

	// The cache file records the current CLI version.
	data, err := os.ReadFile(filepath.Join(dir, forecastAICCacheFileName))
	require.NoError(t, err)
	var c forecastAICCache
	require.NoError(t, json.Unmarshal(data, &c))
	assert.Equal(t, GetVersion(), c.CLIVersion)
	assert.Equal(t, runID, c.RunID)
}

func TestForecastAICCache_NonPositiveNotWritten(t *testing.T) {
	dir := t.TempDir()
	saveForecastAICCache(dir, 1, 0)
	saveForecastAICCache(dir, 1, -5)
	if _, err := os.Stat(filepath.Join(dir, forecastAICCacheFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected no cache file for non-positive AIC")
	}
}

func TestForecastAICCache_InvalidatedOnVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	const runID int64 = 999
	c := forecastAICCache{CLIVersion: "some-old-version", RunID: runID, AIC: 10}
	data, err := json.MarshalIndent(&c, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, forecastAICCacheFileName), data, 0o644))

	if _, ok := loadForecastAICCache(dir, runID); ok {
		t.Fatalf("expected cache miss on CLI version mismatch")
	}
}

func TestForecastAICCache_MismatchedRunID(t *testing.T) {
	dir := t.TempDir()
	saveForecastAICCache(dir, 100, 7.0)
	if _, ok := loadForecastAICCache(dir, 200); ok {
		t.Fatalf("expected cache miss when run ID differs")
	}
}

func TestForecastAICCache_NegativeCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	const runID int64 = 555

	// A no-data marker is a cache hit that reports AIC 0, letting the caller skip the network.
	saveForecastNoDataCache(dir, runID)
	got, ok := loadForecastAICCache(dir, runID)
	require.True(t, ok, "expected negative-cache hit after saving no-data marker")
	assert.Equal(t, 0.0, got)

	// The marker records NoData=true and the current CLI version.
	data, err := os.ReadFile(filepath.Join(dir, forecastAICCacheFileName))
	require.NoError(t, err)
	var c forecastAICCache
	require.NoError(t, json.Unmarshal(data, &c))
	assert.True(t, c.NoData)
	assert.Equal(t, GetVersion(), c.CLIVersion)
	assert.Equal(t, runID, c.RunID)
}

func TestForecastAICCache_NegativeCacheInvalidatedOnVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	const runID int64 = 777
	c := forecastAICCache{CLIVersion: "old", RunID: runID, NoData: true}
	data, err := json.MarshalIndent(&c, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, forecastAICCacheFileName), data, 0o644))

	if _, ok := loadForecastAICCache(dir, runID); ok {
		t.Fatalf("expected miss for stale-version no-data marker so the run is re-checked")
	}
}
