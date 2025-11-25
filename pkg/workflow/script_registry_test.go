package workflow

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptRegistry_Register(t *testing.T) {
	registry := NewScriptRegistry()

	registry.Register("test_script", "console.log('hello');")

	assert.True(t, registry.Has("test_script"))
	assert.False(t, registry.Has("nonexistent"))
}

func TestScriptRegistry_Get_NotFound(t *testing.T) {
	registry := NewScriptRegistry()

	result := registry.Get("nonexistent")

	assert.Empty(t, result)
}

func TestScriptRegistry_Get_BundlesOnce(t *testing.T) {
	registry := NewScriptRegistry()

	// Register a simple script that doesn't require bundling
	source := "console.log('hello');"
	registry.Register("simple", source)

	// Get should bundle and return result
	result1 := registry.Get("simple")
	result2 := registry.Get("simple")

	// Both calls should return the same result (cached)
	assert.Equal(t, result1, result2)
	assert.NotEmpty(t, result1)
}

func TestScriptRegistry_GetSource(t *testing.T) {
	registry := NewScriptRegistry()

	source := "const x = 1;"
	registry.Register("test", source)

	// GetSource should return original source
	assert.Equal(t, source, registry.GetSource("test"))
}

func TestScriptRegistry_GetSource_NotFound(t *testing.T) {
	registry := NewScriptRegistry()

	result := registry.GetSource("nonexistent")

	assert.Empty(t, result)
}

func TestScriptRegistry_Names(t *testing.T) {
	registry := NewScriptRegistry()

	registry.Register("script_a", "a")
	registry.Register("script_b", "b")
	registry.Register("script_c", "c")

	names := registry.Names()

	assert.Len(t, names, 3)
	assert.Contains(t, names, "script_a")
	assert.Contains(t, names, "script_b")
	assert.Contains(t, names, "script_c")
}

func TestScriptRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewScriptRegistry()
	source := "console.log('concurrent test');"
	registry.Register("concurrent", source)

	// Test concurrent Get calls
	var wg sync.WaitGroup
	results := make([]string, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = registry.Get("concurrent")
		}(i)
	}

	wg.Wait()

	// All results should be the same (due to Once semantics)
	for i := 1; i < 10; i++ {
		assert.Equal(t, results[0], results[i], "concurrent access should return consistent results")
	}
}

func TestScriptRegistry_Overwrite(t *testing.T) {
	registry := NewScriptRegistry()

	registry.Register("test", "original")
	assert.Equal(t, "original", registry.GetSource("test"))

	registry.Register("test", "updated")
	assert.Equal(t, "updated", registry.GetSource("test"))
}

func TestDefaultScriptRegistry_GetScript(t *testing.T) {
	// Create a fresh registry for this test to avoid interference
	oldRegistry := DefaultScriptRegistry
	DefaultScriptRegistry = NewScriptRegistry()
	defer func() { DefaultScriptRegistry = oldRegistry }()

	// Register a test script
	DefaultScriptRegistry.Register("test_global", "global test")

	// GetScript should use DefaultScriptRegistry
	result := GetScript("test_global")
	require.NotEmpty(t, result)
}

func TestScriptRegistry_Has(t *testing.T) {
	registry := NewScriptRegistry()

	assert.False(t, registry.Has("missing"))

	registry.Register("present", "code")

	assert.True(t, registry.Has("present"))
	assert.False(t, registry.Has("still_missing"))
}
