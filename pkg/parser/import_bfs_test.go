//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImportSpecsFromArray_RejectsIfField(t *testing.T) {
	_, err := parseImportSpecsFromArray([]any{
		map[string]any{
			"uses": "shared/workflow.md",
			"if":   "experiments.variant == 'a'",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import 'if' is no longer supported")
}
