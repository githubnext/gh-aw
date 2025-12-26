package workflow

import (
	"strings"
	"sync"
	"testing"
)

// TestBundlerIntegration tests the integration of bundler with embedded scripts
func TestBundlerIntegration(t *testing.T) {
	t.Run("getCollectJSONLOutputScript bundles correctly", func(t *testing.T) {
		script := getCollectJSONLOutputScript()

		// Should not be empty
		if script == "" {
			t.Fatal("bundled script is empty")
		}

		// Should contain inlined sanitizeContent function
		if !strings.Contains(script, "function sanitizeContent") {
			t.Error("bundled script does not contain inlined sanitizeContent function")
		}

		// Should contain the inlining comment
		if !strings.Contains(script, "Inlined from") {
			t.Error("bundled script does not contain inlining comment")
		}

		// Should not contain the require statement
		if strings.Contains(script, `require("./sanitize_content.cjs")`) {
			t.Error("bundled script still contains require statement")
		}

		// Should contain original script content
		if !strings.Contains(script, "async function main") {
			t.Error("bundled script does not contain main function")
		}
	})

	t.Run("getComputeTextScript bundles correctly", func(t *testing.T) {
		script := getComputeTextScript()

		// Should not be empty
		if script == "" {
			t.Fatal("bundled script is empty")
		}

		// Should contain inlined sanitizeContent function
		if !strings.Contains(script, "function sanitizeContent") {
			t.Error("bundled script does not contain inlined sanitizeContent function")
		}

		// Should contain the inlining comment
		if !strings.Contains(script, "Inlined from") {
			t.Error("bundled script does not contain inlining comment")
		}

		// Should not contain the require statement
		if strings.Contains(script, `require("./sanitize_content.cjs")`) {
			t.Error("bundled script still contains require statement")
		}

		// Should contain original script content
		if !strings.Contains(script, "async function main") {
			t.Error("bundled script does not contain main function")
		}
	})

	t.Run("getSanitizeOutputScript bundles correctly", func(t *testing.T) {
		script := getSanitizeOutputScript()

		// Should not be empty
		if script == "" {
			t.Fatal("bundled script is empty")
		}

		// Should contain inlined sanitizeContent function
		if !strings.Contains(script, "function sanitizeContent") {
			t.Error("bundled script does not contain inlined sanitizeContent function")
		}

		// Should contain the inlining comment
		if !strings.Contains(script, "Inlined from") {
			t.Error("bundled script does not contain inlining comment")
		}

		// Should not contain the require statement
		if strings.Contains(script, `require("./sanitize_content.cjs")`) {
			t.Error("bundled script still contains require statement")
		}

		// Should contain original script content
		if !strings.Contains(script, "async function main") {
			t.Error("bundled script does not contain main function")
		}
	})
}

// TestBundlerCaching tests that bundling is cached and only happens once
func TestBundlerCaching(t *testing.T) {
	// Reset the sync.Once for testing
	// Note: In production, this would only run once per process

	t.Run("multiple calls return same result", func(t *testing.T) {
		script1 := getCollectJSONLOutputScript()
		script2 := getCollectJSONLOutputScript()

		if script1 != script2 {
			t.Error("multiple calls to getCollectJSONLOutputScript returned different results")
		}

		script3 := getComputeTextScript()
		script4 := getComputeTextScript()

		if script3 != script4 {
			t.Error("multiple calls to getComputeTextScript returned different results")
		}

		script5 := getSanitizeOutputScript()
		script6 := getSanitizeOutputScript()

		if script5 != script6 {
			t.Error("multiple calls to getSanitizeOutputScript returned different results")
		}
	})
}

// TestBundlerConcurrency tests that bundler works correctly with concurrent access
func TestBundlerConcurrency(t *testing.T) {
	const goroutines = 10

	t.Run("concurrent access to getCollectJSONLOutputScript", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]string, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = getCollectJSONLOutputScript()
			}(i)
		}

		wg.Wait()

		// All results should be identical
		first := results[0]
		for i, result := range results {
			if result != first {
				t.Errorf("result %d differs from result 0", i)
			}
		}
	})

	t.Run("concurrent access to getComputeTextScript", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]string, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = getComputeTextScript()
			}(i)
		}

		wg.Wait()

		// All results should be identical
		first := results[0]
		for i, result := range results {
			if result != first {
				t.Errorf("result %d differs from result 0", i)
			}
		}
	})

	t.Run("concurrent access to getSanitizeOutputScript", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]string, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = getSanitizeOutputScript()
			}(i)
		}

		wg.Wait()

		// All results should be identical
		first := results[0]
		for i, result := range results {
			if result != first {
				t.Errorf("result %d differs from result 0", i)
			}
		}
	})
}

// TestBundledScriptsContainHelperFunctions tests that bundled scripts contain expected helper functions
func TestBundledScriptsContainHelperFunctions(t *testing.T) {
	// Different scripts have different helper functions based on what they use
	scriptsWithHelpers := map[string]struct {
		getScript       func() string
		helperFunctions []string
	}{
		"collectJSONLOutput": {
			getScript: getCollectJSONLOutputScript,
			helperFunctions: []string{
				"function sanitizeUrlDomains",
				"function sanitizeUrlProtocols",
				"function neutralizeMentions",
				"function removeXmlComments",
				"function neutralizeBotTriggers",
			},
		},
		"sanitizeOutput": {
			getScript: getSanitizeOutputScript,
			helperFunctions: []string{
				"function sanitizeUrlDomains",
				"function sanitizeUrlProtocols",
				"function neutralizeMentions",
				"function removeXmlComments",
				"function neutralizeBotTriggers",
			},
		},
		"computeText": {
			getScript: getComputeTextScript,
			// compute_text uses sanitizeIncomingText, not sanitizeContent, so it doesn't have neutralizeMentions
			helperFunctions: []string{
				"function sanitizeUrlDomains",
				"function sanitizeUrlProtocols",
				"function removeXmlComments",
			},
		},
	}

	for scriptName, config := range scriptsWithHelpers {
		t.Run(scriptName, func(t *testing.T) {
			script := config.getScript()

			for _, helperFunc := range config.helperFunctions {
				if !strings.Contains(script, helperFunc) {
					t.Errorf("bundled script %s does not contain %s", scriptName, helperFunc)
				}
			}
		})
	}
}

// TestBundledScriptsDoNotContainExports tests that exports are removed
func TestBundledScriptsDoNotContainExports(t *testing.T) {
	scripts := map[string]func() string{
		"collectJSONLOutput": getCollectJSONLOutputScript,
		"computeText":        getComputeTextScript,
		"sanitizeOutput":     getSanitizeOutputScript,
	}

	for scriptName, getScript := range scripts {
		t.Run(scriptName, func(t *testing.T) {
			script := getScript()

			// Should not contain module.exports
			if strings.Contains(script, "module.exports") {
				t.Errorf("bundled script %s still contains module.exports", scriptName)
			}

			// Should not contain exports.
			if strings.Contains(script, "exports.") {
				t.Errorf("bundled script %s still contains exports.", scriptName)
			}
		})
	}
}

// TestBundledScriptsHaveCorrectStructure tests the overall structure
func TestBundledScriptsHaveCorrectStructure(t *testing.T) {
	scripts := map[string]func() string{
		"collectJSONLOutput": getCollectJSONLOutputScript,
		"computeText":        getComputeTextScript,
		"sanitizeOutput":     getSanitizeOutputScript,
	}

	for scriptName, getScript := range scripts {
		t.Run(scriptName, func(t *testing.T) {
			script := getScript()

			// Should have TypeScript check comment
			if !strings.HasPrefix(script, "// @ts-check") {
				t.Errorf("bundled script %s does not start with TypeScript check comment", scriptName)
			}

			// Should contain the boundary markers
			if !strings.Contains(script, "// === Inlined from") {
				t.Errorf("bundled script %s does not contain start boundary marker", scriptName)
			}

			if !strings.Contains(script, "// === End of") {
				t.Errorf("bundled script %s does not contain end boundary marker", scriptName)
			}

			// Should have async function main
			if !strings.Contains(script, "async function main") {
				t.Errorf("bundled script %s does not contain async function main", scriptName)
			}
		})
	}
}

// TestSourceFilesAreSmaller tests that source files are smaller than bundled versions
// SKIPPED: Scripts are now loaded from external files at runtime using require() pattern
func TestSourceFilesAreSmaller(t *testing.T) {
	t.Skip("Bundler tests skipped - scripts now use require() pattern to load external files")
}

// TestGetJavaScriptSources tests that the sources map is correctly populated
// SKIPPED: GetJavaScriptSources now returns empty map as scripts are loaded from external files
func TestGetJavaScriptSources(t *testing.T) {
	t.Skip("GetJavaScriptSources test skipped - scripts now use require() pattern to load external files")
}
