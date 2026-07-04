//go:build !integration

package workflow

import (
	"context"
	"strings"
	"testing"

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
			ActionRepo:    "github/gh-aw-actions/setup-cli",
			ActionVersion: "v0.72.1",
		},
	})

	ghAwRaw, ok := result["gh-aw"]
	require.True(t, ok)
	ghAw, ok := ghAwRaw.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "v1.2.3", ghAw["version"])
	assert.Equal(t, "github.event_name == 'push'", ghAw["if"])
	assert.Equal(t, "github/gh-aw-actions/setup-cli", ghAw["action-repo"])
	assert.Equal(t, "v0.72.1", ghAw["action-version"])
}

func TestDetectRuntimeFromCommand_GhAw(t *testing.T) {
	originalVersion := GetVersion()
	originalRelease := IsRelease()
	t.Cleanup(func() {
		SetVersion(originalVersion)
		SetIsRelease(originalRelease)
	})

	t.Run("release build uses compiler version", func(t *testing.T) {
		SetVersion("v9.9.9")
		SetIsRelease(true)

		requirements := make(map[string]*RuntimeRequirement)
		detectRuntimeFromCommand("gh aw add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md", requirements)

		req, ok := requirements["gh-aw"]
		require.True(t, ok)
		assert.Equal(t, "v9.9.9", req.Version)
	})

	t.Run("dev build uses current build version", func(t *testing.T) {
		SetVersion("dev-build-sha")
		SetIsRelease(false)

		requirements := make(map[string]*RuntimeRequirement)
		detectRuntimeFromCommand("gh aw add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md", requirements)

		req, ok := requirements["gh-aw"]
		require.True(t, ok)
		assert.Equal(t, "dev-build-sha", req.Version)
	})
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

func TestGenerateRuntimeSetupSteps_GhAw_DevBuildsFromSource(t *testing.T) {
	originalVersion := GetVersion()
	originalRelease := IsRelease()
	t.Cleanup(func() {
		SetVersion(originalVersion)
		SetIsRelease(originalRelease)
	})

	SetVersion("dev-build-sha")
	SetIsRelease(false)

	ghAwRuntime := findRuntimeByID("gh-aw")
	require.NotNil(t, ghAwRuntime)

	steps := GenerateRuntimeSetupSteps([]RuntimeRequirement{{
		Runtime: ghAwRuntime,
		Version: "",
	}}, nil)
	require.NotEmpty(t, steps)

	content := strings.Join(steps[0], "\n")
	assert.Contains(t, content, "Build and install gh-aw CLI from source")
	assert.Contains(t, content, "gh extension remove aw || true")
	assert.Contains(t, content, "make build")
	assert.Contains(t, content, "gh extension install .")
	assert.Contains(t, content, "gh aw version")
	assert.Contains(t, content, "GH_TOKEN: ${{ github.token }}")
	assert.NotContains(t, content, "setup-cli@")
}

func TestGenerateRuntimeSetupSteps_GhAw_ReleaseUsesSetupCLI(t *testing.T) {
	originalVersion := GetVersion()
	originalRelease := IsRelease()
	t.Cleanup(func() {
		SetVersion(originalVersion)
		SetIsRelease(originalRelease)
	})

	SetVersion("v0.72.1")
	SetIsRelease(true)

	ghAwRuntime := findRuntimeByID("gh-aw")
	require.NotNil(t, ghAwRuntime)

	steps := GenerateRuntimeSetupSteps([]RuntimeRequirement{{
		Runtime: ghAwRuntime,
		Version: "",
	}}, nil)
	require.NotEmpty(t, steps)

	content := strings.Join(steps[0], "\n")
	assert.Contains(t, content, "uses: github/gh-aw-actions/setup-cli@")
	assert.Contains(t, content, "version: 'v0.72.1'")
}

func TestGenerateRuntimeSetupSteps_GhAw_ReleaseMergesAndSortsExtraFields(t *testing.T) {
	originalVersion := GetVersion()
	originalRelease := IsRelease()
	t.Cleanup(func() {
		SetVersion(originalVersion)
		SetIsRelease(originalRelease)
	})

	SetVersion("v0.72.1")
	SetIsRelease(true)

	ghAwRuntime := *findRuntimeByID("gh-aw")
	ghAwRuntime.ExtraWithFields = map[string]string{
		"alpha": "'runtime'",
		"zeta":  "false",
	}

	steps := GenerateRuntimeSetupSteps([]RuntimeRequirement{{
		Runtime: &ghAwRuntime,
		ExtraFields: map[string]any{
			"alpha": "user",
			"beta":  true,
		},
	}}, nil)
	require.NotEmpty(t, steps)

	content := strings.Join(steps[0], "\n")
	assert.Contains(t, content, "alpha: 'user'")
	assert.NotContains(t, content, "alpha: 'runtime'")
	assert.Contains(t, content, "beta: true")
	assert.Contains(t, content, "zeta: false")
	assert.Less(t, strings.Index(content, "alpha: 'user'"), strings.Index(content, "beta: true"))
	assert.Less(t, strings.Index(content, "beta: true"), strings.Index(content, "zeta: false"))
}

func TestGenerateRuntimeSetupSteps_GhAw_ReleaseUsesWorkflowDataPin(t *testing.T) {
	originalVersion := GetVersion()
	originalRelease := IsRelease()
	t.Cleanup(func() {
		SetVersion(originalVersion)
		SetIsRelease(originalRelease)
	})

	SetVersion("v0.72.1")
	SetIsRelease(true)

	ghAwRuntime := findRuntimeByID("gh-aw")
	require.NotNil(t, ghAwRuntime)

	cache := NewActionCache(t.TempDir())
	sha := "0123456789abcdef0123456789abcdef01234567"
	cache.Set("github/gh-aw-actions/setup-cli", "v0.72.1", sha)

	data := &WorkflowData{
		Ctx:            context.Background(),
		ActionResolver: NewActionResolver(cache),
	}

	steps := GenerateRuntimeSetupSteps([]RuntimeRequirement{{
		Runtime: ghAwRuntime,
		Version: "",
	}}, data)
	require.NotEmpty(t, steps)

	content := strings.Join(steps[0], "\n")
	assert.Contains(t, content, "uses: github/gh-aw-actions/setup-cli@"+sha)
	assert.Contains(t, content, "version: 'v0.72.1'")
}
