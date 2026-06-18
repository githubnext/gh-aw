//go:build !integration

package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/console"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInteractiveConfig_resolveEngineApiKeyCredential(t *testing.T) {
	tests := []struct {
		name            string
		engineOverride  string
		existingSecrets map[string]struct{}
		envVars         map[string]string
		wantName        string
		wantValueEmpty  bool
		wantErr         bool
	}{
		{
			name:           "copilot with token in env",
			engineOverride: "copilot",
			envVars: map[string]string{
				"COPILOT_GITHUB_TOKEN": "test-token-123",
			},
			wantName:       "COPILOT_GITHUB_TOKEN",
			wantValueEmpty: false,
			wantErr:        false,
		},
		{
			name:           "copilot secret already exists",
			engineOverride: "copilot",
			existingSecrets: map[string]struct{}{
				"COPILOT_GITHUB_TOKEN": {},
			},
			wantName:       "COPILOT_GITHUB_TOKEN",
			wantValueEmpty: true,
			wantErr:        false,
		},
		{
			name:           "claude with token in env",
			engineOverride: "claude",
			envVars: map[string]string{
				"ANTHROPIC_API_KEY": "test-api-key-456",
			},
			wantName:       "ANTHROPIC_API_KEY",
			wantValueEmpty: false,
			wantErr:        false,
		},
		{
			name:           "unknown engine",
			engineOverride: "unknown-engine",
			wantErr:        true,
		},
		{
			name:           "copilot with no token",
			engineOverride: "copilot",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, val := range tt.envVars {
				os.Setenv(key, val)
				defer os.Unsetenv(key)
			}

			config := &AddInteractiveConfig{
				EngineOverride:  tt.engineOverride,
				existingSecrets: tt.existingSecrets,
			}

			if config.existingSecrets == nil {
				config.existingSecrets = make(map[string]struct{})
			}

			name, value, err := config.resolveEngineApiKeyCredential()

			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
				assert.Equal(t, tt.wantName, name, "Secret name should match")
				if tt.wantValueEmpty {
					assert.Empty(t, value, "Value should be empty when secret exists")
				} else {
					assert.NotEmpty(t, value, "Value should not be empty")
				}
			}
		})
	}
}

func TestAddInteractiveConfig_configureEngineAPISecret_noWriteAccess(t *testing.T) {
	tests := []struct {
		name   string
		engine string
	}{
		{
			name:   "copilot engine - skips secret setup",
			engine: "copilot",
		},
		{
			name:   "claude engine - skips secret setup",
			engine: "claude",
		},
		{
			name:   "unknown engine - skips without error",
			engine: "unknown-engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AddInteractiveConfig{
				EngineOverride:  tt.engine,
				RepoOverride:    "owner/repo",
				hasWriteAccess:  false,
				existingSecrets: make(map[string]struct{}),
			}

			// When the user doesn't have write access, configureEngineAPISecret should
			// return nil without prompting or uploading any secrets.
			err := config.configureEngineAPISecret(tt.engine)
			require.NoError(t, err, "configureEngineAPISecret should succeed without write access")
		})
	}
}

func TestAddInteractiveConfig_configureEngineAPISecret_skipSecret(t *testing.T) {
	tests := []struct {
		name   string
		engine string
	}{
		{
			name:   "copilot engine - skips secret setup",
			engine: "copilot",
		},
		{
			name:   "claude engine - skips secret setup",
			engine: "claude",
		},
		{
			name:   "unknown engine - skips without error",
			engine: "unknown-engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AddInteractiveConfig{
				EngineOverride:  tt.engine,
				RepoOverride:    "owner/repo",
				hasWriteAccess:  true,
				SkipSecret:      true,
				existingSecrets: make(map[string]struct{}),
			}

			// When SkipSecret is true, configureEngineAPISecret should return nil without
			// prompting or uploading any secrets, even with write access.
			err := config.configureEngineAPISecret(tt.engine)
			require.NoError(t, err, "configureEngineAPISecret should succeed when SkipSecret is true")
		})
	}
}

func TestAddInteractiveConfig_selectAIEngineAndKey_engineOverrideFormatsInfoMessage(t *testing.T) {
	config := &AddInteractiveConfig{
		Ctx:            context.Background(),
		EngineOverride: "copilot",
		SkipSecret:     true,
		RepoOverride:   "owner/repo",
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err, "Failed to create stderr pipe")
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	err = config.selectAIEngineAndKey()

	w.Close()

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr, "Failed to read stderr output")
	require.NoError(t, err, "selectAIEngineAndKey should succeed with an explicit engine override")

	assert.Contains(
		t,
		buf.String(),
		console.FormatInfoMessage("Using coding agent: copilot"),
		"Expected engine override path to use formatted info output",
	)
}

func TestParseSecretNames(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{
			name:     "single secret name",
			input:    []byte("MY_SECRET\n"),
			expected: []string{"MY_SECRET"},
		},
		{
			name:     "multiple secret names",
			input:    []byte("SECRET_A\nSECRET_B\nSECRET_C"),
			expected: []string{"SECRET_A", "SECRET_B", "SECRET_C"},
		},
		{
			name:     "empty output",
			input:    []byte(""),
			expected: nil,
		},
		{
			name:     "output with only whitespace",
			input:    []byte("   \n  \n"),
			expected: nil,
		},
		{
			name:     "names with surrounding whitespace",
			input:    []byte("  MY_SECRET  \n  ANOTHER_SECRET  "),
			expected: []string{"MY_SECRET", "ANOTHER_SECRET"},
		},
		{
			name:     "output with blank lines interspersed",
			input:    []byte("FIRST_SECRET\n\nSECOND_SECRET\n\n"),
			expected: []string{"FIRST_SECRET", "SECOND_SECRET"},
		},
		{
			name:     "trailing newline is handled correctly",
			input:    []byte("SECRET_ONE\nSECRET_TWO\n"),
			expected: []string{"SECRET_ONE", "SECRET_TWO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSecretNames(tt.input)
			assert.Equal(t, tt.expected, result, "parseSecretNames output should match expected")
		})
	}
}

func TestAddInteractiveConfig_checkExistingSecrets(t *testing.T) {
	config := &AddInteractiveConfig{
		RepoOverride: "test-owner/test-repo",
	}

	// This test requires GitHub CLI access, so we just verify it doesn't panic
	// and initializes the existingSecrets map
	require.NotPanics(t, func() {
		_ = config.checkExistingSecrets()
	}, "checkExistingSecrets should not panic")

	assert.NotNil(t, config.existingSecrets, "existingSecrets map should be initialized")
}
