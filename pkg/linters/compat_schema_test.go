//go:build !integration

package linters_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestCompatJSONConformsToSchema(t *testing.T) {
	schemaJSON, err := os.ReadFile("../../.github/aw/compat.schema.json")
	require.NoError(t, err)

	schema, err := parser.CompileSchema(string(schemaJSON), "https://github.com/github/gh-aw/.github/aw/compat.schema.json")
	require.NoError(t, err)

	configJSON, err := os.ReadFile("../../.github/aw/compat.json")
	require.NoError(t, err)

	var config any
	require.NoError(t, json.Unmarshal(configJSON, &config))
	require.NoError(t, schema.Validate(config))
}

// TestCompatBlockedVersionsBoundary locks in the boundaries of the
// GHSA-8h78-hpm7-29gg blocked range so that an accidental edit to
// .github/aw/compat.json is caught by CI instead of silently reintroducing
// an affected version or over-blocking a safe one.
func TestCompatBlockedVersionsBoundary(t *testing.T) {
	configJSON, err := os.ReadFile("../../.github/aw/compat.json")
	require.NoError(t, err)

	var config struct {
		BlockedVersions []string `json:"blockedVersions"`
	}
	require.NoError(t, json.Unmarshal(configJSON, &config))

	require.Contains(t, config.BlockedVersions, "v0.82.8", "first affected version must be blocked")
	require.Contains(t, config.BlockedVersions, "v0.85.3", "last affected version must be blocked")
	require.NotContains(t, config.BlockedVersions, "v0.82.7", "version before the affected range must not be blocked")
	require.NotContains(t, config.BlockedVersions, "v0.85.4", "first fixed version must not be blocked")
}
