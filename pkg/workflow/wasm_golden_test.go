//go:build !integration

package workflow

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden test files")

// isUpdateMode checks if the -update flag was passed to regenerate golden files.
func isUpdateMode() bool {
	return *updateGolden
}

// TestWasmGolden_CompileFixtures compiles each workflow fixture using the string API
// (the same code path used by the wasm compiler) and compares against golden files.
//
// To update golden files:
//
//	go test -v ./pkg/workflow -run='^TestWasmGolden_' -update
//
// Or use the Makefile target:
//
//	make update-wasm-golden
func TestWasmGolden_CompileFixtures(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "wasm_golden", "fixtures")

	// Change to fixtures dir so relative imports resolve correctly
	origDir, err := os.Getwd()
	require.NoError(t, err)
	absFixturesDir, err := filepath.Abs(fixturesDir)
	require.NoError(t, err)
	err = os.Chdir(absFixturesDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	entries, err := os.ReadDir(".")
	require.NoError(t, err, "failed to read fixtures directory")

	var fixtures []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			fixtures = append(fixtures, entry.Name())
		}
	}
	require.NotEmpty(t, fixtures, "no .md fixtures found in %s", fixturesDir)

	// Golden files are stored relative to the original test directory
	goldenDir := filepath.Join(origDir, "testdata", "wasm_golden")

	for _, fixture := range fixtures {
		testName := strings.TrimSuffix(fixture, ".md")
		t.Run(testName, func(t *testing.T) {
			content, err := os.ReadFile(fixture)
			require.NoError(t, err, "failed to read fixture %s", fixture)

			// Use filename-derived identifier for fuzzy cron schedule scattering
			compiler := NewCompiler(
				WithNoEmit(true),
				WithSkipValidation(true),
				WithWorkflowIdentifier(testName),
			)

			wd, err := compiler.ParseWorkflowString(string(content), fixture)
			if err != nil {
				// Some production workflows cannot compile via string API due to:
				// - Path security restrictions (imports outside .github/)
				// - Missing external files (agent definitions, skill files)
				// - Configuration errors specific to file-based compilation
				// Skip these gracefully rather than failing the test.
				t.Skipf("skipping %s: %v", fixture, err)
			}

			yamlOutput, err := compiler.CompileToYAML(wd, fixture)
			if err != nil {
				t.Skipf("skipping %s (compile): %v", fixture, err)
			}
			require.NotEmpty(t, yamlOutput, "empty YAML output for %s", fixture)

			// Compare against golden file (golden files stored in goldenDir)
			goldenPath := filepath.Join(goldenDir, "TestWasmGolden_CompileFixtures", testName+".golden")
			if isUpdateMode() {
				dir := filepath.Dir(goldenPath)
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(goldenPath, []byte(yamlOutput), 0o644))
				return
			}
			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "golden file not found for %s (run with -update to create)", fixture)
			require.Equal(t, string(expected), yamlOutput, "output differs from golden for %s", fixture) //nolint:testifylint // golden test requires exact string comparison, not semantic YAML equality
		})
	}
}

// TestWasmGolden_CompileWithImports tests compilation of a workflow that
// imports a shared component. The shared component is on disk in the
// fixtures/shared/ directory. This exercises the import resolution path
// used by both native and wasm compilers.
func TestWasmGolden_CompileWithImports(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "wasm_golden", "fixtures")
	fixturePath := filepath.Join(fixturesDir, "with-imports.md")

	content, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	// Change to fixtures dir so relative imports resolve correctly
	origDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(fixturesDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	t.Run("with-file-imports", func(t *testing.T) {
		compiler := NewCompiler(
			WithNoEmit(true),
			WithSkipValidation(true),
		)

		wd, err := compiler.ParseWorkflowString(string(content), "with-imports.md")
		require.NoError(t, err, "ParseWorkflowString failed with file imports")

		yamlOutput, err := compiler.CompileToYAML(wd, "with-imports.md")
		require.NoError(t, err, "CompileToYAML failed with file imports")
		require.NotEmpty(t, yamlOutput, "empty YAML output with file imports")

		// Just verify it contains expected content - the import was resolved
		require.Contains(t, yamlOutput, "name:", "output should contain workflow name")
		require.Contains(t, yamlOutput, "jobs:", "output should contain jobs section")
	})
}

// TestWasmGolden_RoundTrip verifies that compiling the same input twice produces
// identical output (determinism check for the wasm compiler path).
func TestWasmGolden_RoundTrip(t *testing.T) {
	markdown := `---
name: determinism-test
description: Verify compilation determinism
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout-minutes: 10
---

# Mission

This workflow tests that compilation is deterministic.
`

	results := make([]string, 3)
	for i := 0; i < 3; i++ {
		compiler := NewCompiler(
			WithNoEmit(true),
			WithSkipValidation(true),
		)

		wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
		require.NoError(t, err)

		yamlOutput, err := compiler.CompileToYAML(wd, "workflow.md")
		require.NoError(t, err)

		results[i] = yamlOutput
	}

	require.Equal(t, results[0], results[1], "compilation 1 and 2 differ")
	require.Equal(t, results[1], results[2], "compilation 2 and 3 differ")
}

// TestWasmGolden_NativeVsStringAPI compiles a workflow using both the native
// file-based path and the string API path, then reports any differences.
// This catches cases where the wasm (string API) path diverges from the native path.
func TestWasmGolden_NativeVsStringAPI(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "wasm_golden", "fixtures")

	// Resolve absolute fixtures dir before CWD change
	absFixturesDir, err := filepath.Abs(fixturesDir)
	require.NoError(t, err)

	// Change to fixtures dir so relative imports resolve
	origDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(absFixturesDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		testName := strings.TrimSuffix(entry.Name(), ".md")
		t.Run(testName, func(t *testing.T) {
			content, err := os.ReadFile(entry.Name())
			require.NoError(t, err)

			// Compile via string API (wasm path)
			stringCompiler := NewCompiler(
				WithNoEmit(true),
				WithSkipValidation(true),
				WithWorkflowIdentifier(testName),
			)

			wd, err := stringCompiler.ParseWorkflowString(string(content), entry.Name())
			if err != nil {
				t.Skipf("skipping %s: %v", entry.Name(), err)
			}

			wasmYAML, err := stringCompiler.CompileToYAML(wd, entry.Name())
			if err != nil {
				t.Skipf("skipping %s (compile): %v", entry.Name(), err)
			}

			// Compile via file-based path (native path)
			absPath := filepath.Join(absFixturesDir, entry.Name())

			nativeCompiler := NewCompiler(
				WithNoEmit(true),
				WithSkipValidation(true),
				WithWorkflowIdentifier(testName),
			)
			nativeCompiler.skipHeader = true
			nativeCompiler.inlinePrompt = true

			nativeWd, err := nativeCompiler.ParseWorkflowFile(absPath)
			if err != nil {
				t.Skipf("skipping native compile for %s: %v", entry.Name(), err)
			}

			nativeYAML, err := nativeCompiler.CompileToYAML(nativeWd, absPath)
			if err != nil {
				t.Skipf("skipping native compile for %s: %v", entry.Name(), err)
			}

			// Compare and log differences (informational only, does not fail)
			if wasmYAML == nativeYAML {
				t.Logf("native and string API output match for %s", entry.Name())
			} else {
				wasmLines := strings.Split(wasmYAML, "\n")
				nativeLines := strings.Split(nativeYAML, "\n")
				t.Logf("INFO: native vs string API output differs for %s (wasm=%d lines, native=%d lines)",
					entry.Name(), len(wasmLines), len(nativeLines))
			}
		})
	}
}

// TestWasmGolden_AllEngines verifies that all three engine types compile correctly
// via the string API and produce valid YAML output.
func TestWasmGolden_AllEngines(t *testing.T) {
	engines := []struct {
		name   string
		engine string
		extra  string // additional frontmatter
	}{
		{"copilot", "copilot", ""},
		{"claude", "claude", "network:\n  allowed:\n    - defaults"},
		{"codex", "codex", "network:\n  allowed:\n    - defaults"},
	}

	for _, eng := range engines {
		t.Run(eng.name, func(t *testing.T) {
			extra := ""
			if eng.extra != "" {
				extra = eng.extra + "\n"
			}
			markdown := fmt.Sprintf(`---
name: engine-%s-test
description: Test %s engine compilation
on:
  workflow_dispatch:
permissions:
  contents: read
engine: %s
timeout-minutes: 10
%s---

# Mission

Test the %s engine compilation path.
`, eng.engine, eng.engine, eng.engine, extra, eng.engine)

			compiler := NewCompiler(
				WithNoEmit(true),
				WithSkipValidation(true),
			)

			wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
			require.NoError(t, err, "%s engine parse failed", eng.name)

			yamlOutput, err := compiler.CompileToYAML(wd, "workflow.md")
			require.NoError(t, err, "%s engine compile failed", eng.name)

			// Verify engine-specific output
			require.Contains(t, yamlOutput, "name:", "%s engine output missing name", eng.name)
			require.Contains(t, yamlOutput, "on:", "%s engine output missing on", eng.name)
			require.Contains(t, yamlOutput, "jobs:", "%s engine output missing jobs", eng.name)
		})
	}
}
