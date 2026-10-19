package aw

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionBlocksSafeOutputSteps(t *testing.T) {
	document, err := Parse(`---
safe-outputs:
  steps:
    - uses: actions/checkout@v6
  jobs:
    publish:
      steps:
        - run: echo public
---
`)
	require.NoError(t, err)
	blocks, err := ExecutionBlocks(context.Background(), document)
	require.NoError(t, err)
	require.Len(t, blocks, 3)
	assert.Empty(t, blocks[0].Path)
	assert.Equal(t, "safe-outputs", blocks[1].Path)
	require.Len(t, blocks[1].Steps, 1)
	assert.Equal(t, "safe-outputs.steps[0]", blocks[1].Steps[0].Path)
	assert.Equal(t, "safe-outputs.jobs.publish", blocks[2].Path)
	require.Len(t, blocks[2].Steps, 1)
	assert.Equal(t, "safe-outputs.jobs.publish.steps[0]", blocks[2].Steps[0].Path)
}

func TestExecutionBlocksMalformedSafeOutputSteps(t *testing.T) {
	document, err := Parse("---\nsafe-outputs: {steps: PRIVATE_VALUE}\n---\n")
	require.NoError(t, err)
	_, err = ExecutionBlocks(context.Background(), document)
	require.ErrorContains(t, err, "safe-outputs.steps")
	assert.NotContains(t, err.Error(), "PRIVATE_VALUE")
}
