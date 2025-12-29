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

	assert.True(t, registry.Has("test_script"), "registry should have test_script after registration")
	assert.False(t, registry.Has("nonexistent"), "registry should not have nonexistent script")
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

func TestScriptRegistry_Overwrite_AfterGet(t *testing.T) {
	registry := NewScriptRegistry()

	// Register initial script
	registry.Register("test", "console.log('original');")

	// Trigger bundling by calling Get()
	firstResult := registry.Get("test")
	assert.NotEmpty(t, firstResult)
	assert.Contains(t, firstResult, "original")

	// Overwrite with new source
	registry.Register("test", "console.log('updated');")

	// Verify GetSource returns new source
	assert.Equal(t, "console.log('updated');", registry.GetSource("test"))

	// Verify Get() returns bundled version of new source
	secondResult := registry.Get("test")
	assert.NotEmpty(t, secondResult)
	assert.Contains(t, secondResult, "updated")
	assert.NotContains(t, secondResult, "original")
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

	assert.False(t, registry.Has("missing"), "registry should not have missing script")

	registry.Register("present", "code")

	assert.True(t, registry.Has("present"), "registry should have present script after registration")
	assert.False(t, registry.Has("still_missing"), "registry should not have still_missing script")
}

func TestScriptRegistry_RegisterWithMode(t *testing.T) {
	// Create a custom registry for testing to avoid side effects
	registry := NewScriptRegistry()

	// Test that bundling respects runtime mode
	// In GitHub Script mode: module.exports should be removed
	// In Node.js mode: module.exports should be preserved

	scriptWithExports := `function test() {
  return 42;
}

module.exports = { test };
`

	// Register with GitHub Script mode (default)
	registry.Register("github_mode", scriptWithExports)
	githubResult := registry.Get("github_mode")

	// Should not contain module.exports in GitHub Script mode
	assert.NotContains(t, githubResult, "module.exports",
		"GitHub Script mode should remove module.exports")
	assert.Contains(t, githubResult, "function test()",
		"Should still contain the function")

	// Register with Node.js mode
	registry.RegisterWithMode("nodejs_mode", scriptWithExports, RuntimeModeNodeJS)
	nodejsResult := registry.Get("nodejs_mode")

	// Should contain module.exports in Node.js mode
	assert.Contains(t, nodejsResult, "module.exports",
		"Node.js mode should preserve module.exports")
	assert.Contains(t, nodejsResult, "function test()",
		"Should still contain the function")
}

func TestScriptRegistry_RegisterWithMode_PreservesDifference(t *testing.T) {
	registry := NewScriptRegistry()

	source := `function helper() { 
  return "value"; 
}

module.exports = { helper };`

	// Register same source with different modes
	registry.RegisterWithMode("github_mode", source, RuntimeModeGitHubScript)
	registry.RegisterWithMode("nodejs_mode", source, RuntimeModeNodeJS)

	githubResult := registry.Get("github_mode")
	nodejsResult := registry.Get("nodejs_mode")

	// GitHub Script mode should remove module.exports
	assert.NotContains(t, githubResult, "module.exports",
		"GitHub Script mode should remove module.exports")
	assert.Contains(t, githubResult, "function helper()",
		"Should contain the function in GitHub mode")

	// Node.js mode should preserve module.exports
	assert.Contains(t, nodejsResult, "module.exports",
		"Node.js mode should preserve module.exports")
	assert.Contains(t, nodejsResult, "function helper()",
		"Should contain the function in Node.js mode")
}

func TestScriptRegistry_GetWithMode(t *testing.T) {
	registry := NewScriptRegistry()

	source := `function helper() { 
  return "value"; 
}

module.exports = { helper };`

	// Register with GitHub Script mode
	registry.RegisterWithMode("test_script", source, RuntimeModeGitHubScript)

	// Test GetWithMode with matching mode - should work without warning
	result := registry.GetWithMode("test_script", RuntimeModeGitHubScript)
	assert.NotEmpty(t, result, "Should return bundled script")
	assert.NotContains(t, result, "module.exports", "GitHub Script mode should remove module.exports")

	// Test GetWithMode with mismatched mode - should log warning but still work
	result2 := registry.GetWithMode("test_script", RuntimeModeNodeJS)
	assert.NotEmpty(t, result2, "Should return bundled script even with mode mismatch")
	// The script was bundled with GitHub Script mode, so module.exports should still be removed
	assert.NotContains(t, result2, "module.exports", "Script was bundled with GitHub Script mode")
}

func TestScriptRegistry_GetWithMode_ModeMismatch(t *testing.T) {
	registry := NewScriptRegistry()

	source := `function test() { return 42; }
module.exports = { test };`

	// Register with Node.js mode
	registry.RegisterWithMode("nodejs_script", source, RuntimeModeNodeJS)

	// Request with GitHub Script mode - should log warning
	result := registry.GetWithMode("nodejs_script", RuntimeModeGitHubScript)

	// Script was bundled with Node.js mode, so module.exports should be preserved
	assert.Contains(t, result, "module.exports", "Node.js mode should preserve module.exports")
}

func TestGetScriptWithMode(t *testing.T) {
	// Create a fresh registry for this test
	oldRegistry := DefaultScriptRegistry
	DefaultScriptRegistry = NewScriptRegistry()
	defer func() { DefaultScriptRegistry = oldRegistry }()

	// Register a test script
	DefaultScriptRegistry.RegisterWithMode("test_helper", "function test() { return 1; }", RuntimeModeGitHubScript)

	// Test GetScriptWithMode
	result := GetScriptWithMode("test_helper", RuntimeModeGitHubScript)
	require.NotEmpty(t, result)
	assert.Contains(t, result, "function test()")
}
