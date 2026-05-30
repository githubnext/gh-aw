//go:build !integration

package workflow

import (
	"testing"
)

// BenchmarkExtractExpressionsFromPlaywrightArgs benchmarks expression extraction
func BenchmarkExtractExpressionsFromPlaywrightArgs(b *testing.B) {
	customArgs := []string{"--debug", "--timeout", "${{ github.event.inputs.timeout }}"}

	for b.Loop() {
		_ = extractExpressionsFromPlaywrightArgs(customArgs)
	}
}
