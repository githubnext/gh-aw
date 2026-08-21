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
	require.NotNil(t, config.CloseOlderConfig.Enabled)
	assert.Equal(t, "true", *config.CloseOlderConfig.Enabled)
	assert.Equal(t, "issue-key", config.CloseOlderConfig.Key)
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
	require.NotNil(t, config.CloseOlderConfig.Enabled)
	assert.Equal(t, "${{ true }}", *config.CloseOlderConfig.Enabled)
	assert.Equal(t, "discussion-key", config.CloseOlderConfig.Key)
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
	require.NotNil(t, config.CloseOlderConfig.Enabled)
	assert.Equal(t, "true", *config.CloseOlderConfig.Enabled)
	assert.Equal(t, "pull-request-key", config.CloseOlderConfig.Key)
}
