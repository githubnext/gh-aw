//go:build !integration

package linters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIEnforcedLinterSetsIgnoreShellComments(t *testing.T) {
	t.Parallel()
	workflow := []byte(`
jobs:
  lint-go-custom:
    steps:
      - run: |
          # GOOS=js GOARCH=wasm make golint-custom LINTER_FLAGS="-commentedout -test=false"
          echo skipped
      - run: |
          # GOOS=js GOARCH=wasm
          make golint-custom LINTER_FLAGS="-consolestderr -test=false"
      - env:
          GOOS: js
          GOARCH: wasm
        run: make golint-custom LINTER_FLAGS='-errstringmatch -test=false'
`)

	native, wasm := parseCIEnforcedLinterSets(t, workflow)
	assert.Equal(t, []string{"consolestderr"}, sortedKeys(native))
	assert.Equal(t, []string{"errstringmatch"}, sortedKeys(wasm))
}
