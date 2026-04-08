//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRepoConfig_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "missing aw.json should return default config without error")
	assert.False(t, cfg.MaintenanceDisabled, "maintenance should be enabled by default")
	assert.Nil(t, cfg.Maintenance, "maintenance config should be nil when file is absent")
}

func TestLoadRepoConfig_MaintenanceFalse(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"maintenance": false}`)

	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "valid aw.json should load without error")
	assert.True(t, cfg.MaintenanceDisabled, "maintenance should be disabled")
	assert.Nil(t, cfg.Maintenance, "maintenance config should be nil when disabled")
}

func TestLoadRepoConfig_MaintenanceWithStringRunsOn(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"maintenance": {"runs_on": "custom-runner"}}`)

	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "valid aw.json should load without error")
	assert.False(t, cfg.MaintenanceDisabled, "maintenance should not be disabled")
	require.NotNil(t, cfg.Maintenance, "maintenance config should be set")
	assert.Equal(t, RunsOnValue{"custom-runner"}, cfg.Maintenance.RunsOn, "string runs_on should be normalised to a single-element RunsOnValue")
}

func TestLoadRepoConfig_MaintenanceWithArrayRunsOn(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"maintenance": {"runs_on": ["self-hosted", "linux"]}}`)

	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "valid aw.json should load without error")
	assert.False(t, cfg.MaintenanceDisabled, "maintenance should not be disabled")
	require.NotNil(t, cfg.Maintenance, "maintenance config should be set")
	assert.Equal(t, RunsOnValue{"self-hosted", "linux"}, cfg.Maintenance.RunsOn, "array runs_on should be deserialised as RunsOnValue")
}

func TestLoadRepoConfig_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{}`)

	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "empty aw.json should load without error")
	assert.False(t, cfg.MaintenanceDisabled, "maintenance should be enabled by default")
	assert.Nil(t, cfg.Maintenance, "maintenance config should be nil when not specified")
}

func TestLoadRepoConfig_MaintenanceEmptyObject(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"maintenance": {}}`)

	cfg, err := LoadRepoConfig(dir)
	require.NoError(t, err, "aw.json with empty maintenance object should load without error")
	assert.False(t, cfg.MaintenanceDisabled, "maintenance should not be disabled")
	require.NotNil(t, cfg.Maintenance, "maintenance config should be set")
	assert.Empty(t, cfg.Maintenance.RunsOn, "runs_on should be empty when not specified")
}

func TestLoadRepoConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeAWJSONRaw(t, dir, `not-json`)

	_, err := LoadRepoConfig(dir)
	assert.Error(t, err, "invalid JSON should return an error")
}

func TestLoadRepoConfig_SchemaViolation(t *testing.T) {
	dir := t.TempDir()
	// "maintenance: true" is not allowed by the schema (only false or object)
	writeAWJSON(t, dir, `{"maintenance": true}`)

	_, err := LoadRepoConfig(dir)
	assert.Error(t, err, "schema violation should return an error")
}

func TestLoadRepoConfig_UnknownProperty(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"unknown_property": "value"}`)

	_, err := LoadRepoConfig(dir)
	assert.Error(t, err, "unknown property should fail schema validation (additionalProperties: false)")
}

// TestFormatRunsOn tests the YAML serialisation of runs-on values.
func TestFormatRunsOn(t *testing.T) {
	const def = "ubuntu-slim"

	tests := []struct {
		name     string
		runsOn   RunsOnValue
		expected string
	}{
		{"nil uses default", nil, def},
		{"empty slice uses default", RunsOnValue{}, def},
		{"empty string element uses default", RunsOnValue{""}, def},
		{"single label", RunsOnValue{"custom-runner"}, "custom-runner"},
		{"single self-hosted label", RunsOnValue{"self-hosted"}, "self-hosted"},
		{"multi-label array", RunsOnValue{"self-hosted", "linux"}, `["self-hosted","linux"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRunsOn(tt.runsOn, def)
			assert.Equal(t, tt.expected, got, "FormatRunsOn should return expected YAML value")
		})
	}
}

// writeAWJSON creates .github/workflows/aw.json with the given JSON content.
func writeAWJSON(t *testing.T, gitRoot, content string) {
	t.Helper()
	writeAWJSONRaw(t, gitRoot, content)
}

// writeAWJSONRaw creates .github/workflows/aw.json with raw (possibly invalid) content.
func writeAWJSONRaw(t *testing.T, gitRoot, content string) {
	t.Helper()
	dir := filepath.Join(gitRoot, ".github", "workflows")
	require.NoError(t, os.MkdirAll(dir, 0o755), "failed to create workflows dir")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "aw.json"), []byte(content), 0o600), "failed to write aw.json")
}
