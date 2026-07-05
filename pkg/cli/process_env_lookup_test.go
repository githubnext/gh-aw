//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetProcessEnvLookup(t *testing.T) {
	t.Cleanup(func() { SetProcessEnvLookup(nil) })

	SetProcessEnvLookup(func(key string) (string, bool) {
		env := map[string]string{
			"PRESENT": "hello",
			"EMPTY":   "",
		}
		v, ok := env[key]
		return v, ok
	})

	t.Run("lookupEnv returns value for present key", func(t *testing.T) {
		assert.Equal(t, "hello", lookupEnv("PRESENT"))
	})

	t.Run("lookupEnv returns empty string for missing key", func(t *testing.T) {
		assert.Empty(t, lookupEnv("MISSING"))
	})

	t.Run("lookupEnv returns empty string for explicitly empty key", func(t *testing.T) {
		assert.Empty(t, lookupEnv("EMPTY"))
	})

	t.Run("lookupEnvOk returns value and true for present key", func(t *testing.T) {
		val, ok := lookupEnvOk("PRESENT")
		assert.Equal(t, "hello", val)
		assert.True(t, ok)
	})

	t.Run("lookupEnvOk returns empty string and false for missing key", func(t *testing.T) {
		val, ok := lookupEnvOk("MISSING")
		assert.Empty(t, val)
		assert.False(t, ok)
	})

	t.Run("lookupEnvOk returns empty string and true for explicitly empty key", func(t *testing.T) {
		val, ok := lookupEnvOk("EMPTY")
		assert.Empty(t, val)
		assert.True(t, ok)
	})
}

func TestSetProcessEnvLookupNilRestoresDefault(t *testing.T) {
	t.Cleanup(func() { SetProcessEnvLookup(nil) })

	SetProcessEnvLookup(func(key string) (string, bool) {
		return "overridden", true
	})

	SetProcessEnvLookup(nil)

	// After restoring the default, missing keys should return "" / false.
	val, ok := lookupEnvOk("__GH_AW_TEST_KEY_THAT_MUST_NOT_EXIST__")
	assert.Empty(t, val)
	assert.False(t, ok)
}
