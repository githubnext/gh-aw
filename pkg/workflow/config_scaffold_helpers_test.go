//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testScaffoldConfig is a minimal config type used only in scaffold helper tests.
type testScaffoldConfig struct {
	Name    string   `yaml:"name,omitempty"`
	Allowed []string `yaml:"allowed,omitempty"`
}

var testScaffoldLog = logger.New("workflow:test_scaffold")

func TestParseConfigScaffold_KeyAbsent(t *testing.T) {
	outputMap := map[string]any{
		"other-key": map[string]any{"name": "value"},
	}

	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			t.Error("onError should not be called when key is absent")
			return nil
		})

	assert.Nil(t, result, "should return nil when key is absent")
}

func TestParseConfigScaffold_KeyPresentNullValue(t *testing.T) {
	// Null YAML value (nil) should unmarshal into a zero-value struct successfully
	outputMap := map[string]any{
		"my-key": nil,
	}

	onErrorCalled := false
	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			onErrorCalled = true
			return nil
		})

	assert.False(t, onErrorCalled, "onError should not be called for nil (null) config value")
	require.NotNil(t, result, "should return non-nil zero-value config for null config")
	assert.Empty(t, result.Name, "zero-value Name should be empty string")
}

func TestParseConfigScaffold_ValidConfig(t *testing.T) {
	outputMap := map[string]any{
		"my-key": map[string]any{
			"name":    "test-name",
			"allowed": []any{"foo", "bar"},
		},
	}

	onErrorCalled := false
	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			onErrorCalled = true
			return nil
		})

	assert.False(t, onErrorCalled, "onError should not be called for valid config")
	require.NotNil(t, result, "should return parsed config")
	assert.Equal(t, "test-name", result.Name, "should parse name field")
	assert.Equal(t, []string{"foo", "bar"}, result.Allowed, "should parse allowed array")
}

func TestParseConfigScaffold_EmptyConfigFallback(t *testing.T) {
	// A non-map value causes unmarshal to fail → onError must return empty fallback
	outputMap := map[string]any{
		"my-key": "not-a-map",
	}

	fallback := &testScaffoldConfig{Name: "fallback"}
	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			assert.Error(t, err, "onError should receive the unmarshal error")
			return fallback
		})

	assert.Equal(t, fallback, result, "should return the fallback from onError")
}

func TestParseConfigScaffold_DisableOnError(t *testing.T) {
	// A non-map value causes unmarshal to fail → onError returns nil (disable handler)
	outputMap := map[string]any{
		"my-key": "not-a-map",
	}

	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			return nil
		})

	assert.Nil(t, result, "should return nil when onError disables the handler")
}

func TestParseConfigScaffold_OnErrorReceivesError(t *testing.T) {
	outputMap := map[string]any{
		"my-key": "not-a-map",
	}

	var capturedErr error
	parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			capturedErr = err
			return nil
		})

	require.Error(t, capturedErr, "onError should receive a non-nil error")
}

func TestParseConfigScaffold_EmptyMap(t *testing.T) {
	// An empty map should unmarshal into a zero-value struct successfully
	outputMap := map[string]any{
		"my-key": map[string]any{},
	}

	onErrorCalled := false
	result := parseConfigScaffold(outputMap, "my-key", testScaffoldLog,
		func(err error) *testScaffoldConfig {
			onErrorCalled = true
			return nil
		})

	assert.False(t, onErrorCalled, "onError should not be called for empty map config")
	require.NotNil(t, result, "should return non-nil zero-value config for empty map")
	assert.Empty(t, result.Name, "zero-value Name should be empty string")
	assert.Nil(t, result.Allowed, "zero-value Allowed should be nil")
}
