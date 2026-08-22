//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCreateIssuesConfigMapsCloseOlderConfig(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateIssuesConfig(map[string]any{
		"create-issue": map[string]any{
			"close-older-issues": true,
			"close-older-key":    "issue-key",
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.Enabled)
	assert.Equal(t, "true", *config.Enabled)
	assert.Equal(t, "issue-key", config.Key)
}

func TestParseCreateDiscussionsConfigMapsCloseOlderConfig(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateDiscussionsConfig(map[string]any{
		"create-discussion": map[string]any{
			"close-older-discussions": "${{ true }}",
			"close-older-key":         "discussion-key",
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.Enabled)
	assert.Equal(t, "${{ true }}", *config.Enabled)
	assert.Equal(t, "discussion-key", config.Key)
}

func TestParseCreatePullRequestsConfigMapsCloseOlderConfig(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreatePullRequestsConfig(map[string]any{
		"create-pull-request": map[string]any{
			"close-older-pull-requests": true,
			"close-older-key":           "pull-request-key",
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.Enabled)
	assert.Equal(t, "true", *config.Enabled)
	assert.Equal(t, "pull-request-key", config.Key)
}

func TestParseCreateIssuesConfigNoCloseOlderWhenAbsent(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateIssuesConfig(map[string]any{
		"create-issue": map[string]any{},
	})

	require.NotNil(t, config)
	assert.Nil(t, config.Enabled, "Enabled should be nil when not set")
	assert.Empty(t, config.Key)
}

func TestParseCreateIssuesConfigCloseOlderExplicitFalse(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateIssuesConfig(map[string]any{
		"create-issue": map[string]any{
			"close-older-issues": false,
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.Enabled, "Enabled should be set (not nil) when explicitly false")
	assert.Equal(t, "false", *config.Enabled)
}

func TestParseCreateDiscussionsConfigNoCloseOlderWhenAbsent(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateDiscussionsConfig(map[string]any{
		"create-discussion": map[string]any{},
	})

	require.NotNil(t, config)
	assert.Nil(t, config.Enabled, "Enabled should be nil when not set")
	assert.Empty(t, config.Key)
}

func TestParseCreatePullRequestsConfigNoCloseOlderWhenAbsent(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreatePullRequestsConfig(map[string]any{
		"create-pull-request": map[string]any{},
	})

	require.NotNil(t, config)
	assert.Nil(t, config.Enabled, "Enabled should be nil when not set")
	assert.Empty(t, config.Key)
}

// TestParseCreateIssuesConfigCloseOlderEnabledIsNotAPublicKey verifies that the internal
// canonical field name (matching the shared CloseOlderConfig.Enabled tag prior to this
// change) is not accepted directly as a workflow-authored YAML key, since Enabled is now
// tagged yaml:"-" and only populated via closeOlderEnabledFromConfigData.
func TestParseCreateIssuesConfigCloseOlderEnabledIsNotAPublicKey(t *testing.T) {
	compiler := NewCompiler(WithFailFast(true))
	config := compiler.parseCreateIssuesConfig(map[string]any{
		"create-issue": map[string]any{
			"close-older-enabled": true,
		},
	})

	require.NotNil(t, config)
	assert.Nil(t, config.Enabled, "close-older-enabled must not be a supported public YAML key")
}
