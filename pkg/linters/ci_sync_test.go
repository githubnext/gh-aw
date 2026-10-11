//go:build !integration

package linters_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIEnforcedLinterTargetDrift(t *testing.T) {
	t.Parallel()
	workflowBytes, err := os.ReadFile("../../.github/workflows/cgo.yml")
	require.NoError(t, err)
	workflow := string(workflowBytes)
	nativeConsole := `run: make golint-custom LINTER_FLAGS="-consolestderr -test=false"`
	wasmConsole := `run: GOOS=js GOARCH=wasm make golint-custom LINTER_FLAGS="-consolestderr -test=false"`
	nativeBulk := `run: make golint-custom LINTER_FLAGS="-errstringmatch`
	wasmBulk := `run: GOOS=js GOARCH=wasm make golint-custom LINTER_FLAGS="-errstringmatch`
	require.Contains(t, workflow, nativeConsole)
	require.Contains(t, workflow, wasmConsole)
	require.Contains(t, workflow, nativeBulk)
	require.Contains(t, workflow, wasmBulk)

	tests := []struct {
		name           string
		workflow       string
		wantNativeOnly []string
		wantWasmOnly   []string
	}{
		{
			name:           "dedicated steps match",
			workflow:       workflow,
			wantNativeOnly: []string{"contextcancelnotdeferred"},
			wantWasmOnly:   []string{},
		},
		{
			name:           "missing dedicated wasm step reproduces original drift",
			workflow:       strings.Replace(workflow, wasmConsole, "run: echo skipped", 1),
			wantNativeOnly: []string{"consolestderr", "contextcancelnotdeferred"},
			wantWasmOnly:   []string{},
		},
		{
			name:           "missing dedicated native step",
			workflow:       strings.Replace(workflow, nativeConsole, "run: echo skipped", 1),
			wantNativeOnly: []string{"contextcancelnotdeferred"},
			wantWasmOnly:   []string{"consolestderr"},
		},
		{
			name:           "new native bulk flag cannot hide in registry union",
			workflow:       strings.Replace(workflow, nativeBulk, nativeBulk+" -futurelinter", 1),
			wantNativeOnly: []string{"contextcancelnotdeferred", "futurelinter"},
			wantWasmOnly:   []string{},
		},
		{
			name:           "new wasm bulk flag",
			workflow:       strings.Replace(workflow, wasmBulk, wasmBulk+" -futurelinter", 1),
			wantNativeOnly: []string{"contextcancelnotdeferred"},
			wantWasmOnly:   []string{"futurelinter"},
		},
		{
			name:           "restored parity makes native exception stale",
			workflow:       strings.Replace(workflow, wasmBulk, wasmBulk+" -contextcancelnotdeferred", 1),
			wantNativeOnly: []string{},
			wantWasmOnly:   []string{},
		},
		{
			name: "wasm target in step environment and single quoted flags",
			workflow: strings.Replace(workflow, wasmConsole,
				"env:\n        GOOS: js\n        GOARCH: wasm\n"+
					"      run: make golint-custom LINTER_FLAGS='-consolestderr -test=false'", 1),
			wantNativeOnly: []string{"contextcancelnotdeferred"},
			wantWasmOnly:   []string{},
		},
		{
			name:           "comments do not count as enforcement",
			workflow:       strings.Replace(workflow, wasmConsole, "# "+wasmConsole+"\n      run: echo skipped", 1),
			wantNativeOnly: []string{"consolestderr", "contextcancelnotdeferred"},
			wantWasmOnly:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			native, wasm := parseCIEnforcedLinterSets(t, []byte(tt.workflow))
			assert.Equal(t, tt.wantNativeOnly, linterSetDifference(native, wasm))
			assert.Equal(t, tt.wantWasmOnly, linterSetDifference(wasm, native))
			assert.NotContains(t, native, "test=false")
			assert.NotContains(t, wasm, "test=false")
		})
	}
}
