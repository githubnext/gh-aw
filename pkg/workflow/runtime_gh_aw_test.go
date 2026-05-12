//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRuntimesConfig_GhAw(t *testing.T) {
	config, err := parseRuntimesConfig(map[string]any{
		"gh-aw": map[string]any{
			"version": "v9.9.9",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.GhAw)
	assert.Equal(t, "v9.9.9", config.GhAw.Version)
}

func TestRuntimesConfigToMap_GhAw(t *testing.T) {
	result := runtimesConfigToMap(&RuntimesConfig{
		GhAw: &RuntimeConfig{
			Version:       "v1.2.3",
			If:            "github.event_name == 'push'",
			ActionRepo:    "github/gh-aw/actions/setup-cli",
			ActionVersion: "main",
		},
	})

	ghAwRaw, ok := result["gh-aw"]
	require.True(t, ok)
	ghAw, ok := ghAwRaw.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "v1.2.3", ghAw["version"])
	assert.Equal(t, "github.event_name == 'push'", ghAw["if"])
	assert.Equal(t, "github/gh-aw/actions/setup-cli", ghAw["action-repo"])
	assert.Equal(t, "main", ghAw["action-version"])
}

func TestDetectRuntimeFromCommand_GhAw(t *testing.T) {
	requirements := make(map[string]*RuntimeRequirement)
	detectRuntimeFromCommand("gh aw add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md", requirements)

	req, ok := requirements["gh-aw"]
	require.True(t, ok)
	assert.Equal(t, string(constants.DefaultGhAWVersion), req.Version)
}

func TestGetDomainsFromRuntimes_GhAw(t *testing.T) {
	domains := getDomainsFromRuntimes(map[string]any{
		"gh-aw": map[string]any{
			"version": "v0.72.1",
		},
	})

	assert.Contains(t, domains, "github.com")
	assert.Contains(t, domains, "github.github.com")
	assert.Contains(t, domains, "raw.githubusercontent.com")
}
