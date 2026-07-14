package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownRenamedIncludedFieldHints(t *testing.T) {
	hints := knownRenamedIncludedFieldHints([]string{"safe-inputs", "other"})
	assert.Len(t, hints, 1)
	assert.Contains(t, hints[0], "safe-inputs")
	assert.Contains(t, hints[0], "mcp-scripts")
	assert.Contains(t, hints[0], "gh aw fix --write")
}
