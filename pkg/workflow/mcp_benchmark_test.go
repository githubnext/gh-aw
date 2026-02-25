//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// BenchmarkRenderPlaywrightMCPConfig benchmarks Playwright MCP config generation
func BenchmarkRenderPlaywrightMCPConfig(b *testing.B) {
	playwrightTool := map[string]any{
		"container": "mcr.microsoft.com/playwright:v1.41.0",
		"args":      []any{"--debug"},
	}
	playwrightConfig := parsePlaywrightTool(playwrightTool)

	for b.Loop() {
		var yaml strings.Builder
		renderPlaywrightMCPConfig(&yaml, playwrightConfig, true)
	}
}

// BenchmarkGeneratePlaywrightDockerArgs benchmarks Playwright args generation
func BenchmarkGeneratePlaywrightDockerArgs(b *testing.B) {
	playwrightTool := map[string]any{
		"container": "mcr.microsoft.com/playwright:v1.41.0",
	}
	playwrightConfig := parsePlaywrightTool(playwrightTool)

	for b.Loop() {
		_ = generatePlaywrightDockerArgs(playwrightConfig)
	}
}

// BenchmarkRenderPlaywrightMCPConfig_Complex benchmarks complex Playwright config
func BenchmarkRenderPlaywrightMCPConfig_Complex(b *testing.B) {
	playwrightTool := map[string]any{
		"container": "mcr.microsoft.com/playwright:v1.41.0",
		"args":      []any{"--debug", "--timeout", "30000"},
	}
	playwrightConfig := parsePlaywrightTool(playwrightTool)

	for b.Loop() {
		var yaml strings.Builder
		renderPlaywrightMCPConfig(&yaml, playwrightConfig, true)
	}
}

// BenchmarkExtractExpressionsFromPlaywrightArgs benchmarks expression extraction
func BenchmarkExtractExpressionsFromPlaywrightArgs(b *testing.B) {
	customArgs := []string{"--debug", "--timeout", "${{ github.event.inputs.timeout }}"}

	for b.Loop() {
		_ = extractExpressionsFromPlaywrightArgs(customArgs)
	}
}
