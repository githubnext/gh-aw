//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildToolsMetaRuntimeDataExtractsExpressions(t *testing.T) {
	input := `{"dynamic_tools":[{"inputSchema":{"properties":{"repo":{"default":"${{ inputs.target_repo }}"}}}}]}`

	sanitized, envKeys, envValues := buildToolsMetaRuntimeData(input)

	require.Len(t, envKeys, 1)
	assert.Equal(t, "${{ inputs.target_repo }}", envValues[envKeys[0]])
	assert.Contains(t, sanitized, `"default":"${`+envKeys[0]+`}"`)
	assert.NotContains(t, sanitized, "${{ inputs.target_repo }}")
}

func TestBuildToolsMetaRuntimeDataWithoutExpressions(t *testing.T) {
	input := `{"dynamic_tools":[]}`

	sanitized, envKeys, envValues := buildToolsMetaRuntimeData(input)

	assert.Equal(t, input, sanitized)
	assert.Nil(t, envKeys)
	assert.Nil(t, envValues)
}
