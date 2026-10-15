//go:build !integration

package workflow

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputsMCPGatewayOptionalTargetRepo(t *testing.T) {
	tools := map[string]any{"safe-outputs": map[string]any{}}
	workflowData := &WorkflowData{
		Tools:       tools,
		SafeOutputs: &SafeOutputsConfig{AddComments: &AddCommentsConfig{}},
	}
	var yaml strings.Builder
	compiler := &Compiler{}
	require.NoError(t, compiler.generateMCPSetup(&yaml, tools, NewCopilotEngine(), workflowData))
	output := yaml.String()
	const export = `export GH_AW_TARGET_REPO_SLUG="${GH_AW_TARGET_REPO_SLUG:-}"`
	require.Contains(t, output, export)
	assert.Contains(t, output, `"GH_AW_TARGET_REPO_SLUG": "\${GH_AW_TARGET_REPO_SLUG}"`)
	assert.Contains(t, output, "-e GH_AW_TARGET_REPO_SLUG")
	assert.Less(t, strings.Index(output, export), strings.Index(output, "docker run"))
	assert.NotContains(t, collectMCPEnvironmentVariables(tools, []string{"safe-outputs"}, workflowData, false), "GH_AW_TARGET_REPO_SLUG")

	for _, tc := range []struct {
		name string
		env  []string
		want string
	}{
		{name: "unset"},
		{name: "empty", env: []string{"GH_AW_TARGET_REPO_SLUG="}},
		{name: "configured", env: []string{"GH_AW_TARGET_REPO_SLUG=owner/target-repo"}, want: "owner/target-repo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", "-eu", "-c", export+`; printf '%s' "${GH_AW_TARGET_REPO_SLUG?}"`)
			cmd.Env = append([]string{}, tc.env...)
			value, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", value)
			assert.Equal(t, tc.want, string(value))
		})
	}
}

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

func TestBuildToolsMetaRuntimeDataWithMultipleDistinctExpressions(t *testing.T) {
	input := `{"description_suffixes":{"create_issue":"${{ inputs.target_repo }}"},"repo_params":{"owner":"${{ github.repository_owner }}"},"dynamic_tools":[]}`

	sanitized, envKeys, envValues := buildToolsMetaRuntimeData(input)

	require.Len(t, envKeys, 2)
	assert.NotContains(t, sanitized, "${{ inputs.target_repo }}")
	assert.NotContains(t, sanitized, "${{ github.repository_owner }}")
	seen := map[string]bool{}
	for _, key := range envKeys {
		seen[envValues[key]] = true
		assert.Contains(t, sanitized, "${"+key+"}")
	}
	assert.True(t, seen["${{ inputs.target_repo }}"])
	assert.True(t, seen["${{ github.repository_owner }}"])
}

func TestBuildToolsMetaRuntimeDataDedupesRepeatedExpression(t *testing.T) {
	input := `{"description_suffixes":{"a":"${{ inputs.target_repo }}","b":"${{ inputs.target_repo }}"},"dynamic_tools":[]}`

	sanitized, envKeys, envValues := buildToolsMetaRuntimeData(input)

	require.Len(t, envKeys, 1)
	assert.Equal(t, "${{ inputs.target_repo }}", envValues[envKeys[0]])
	assert.NotContains(t, sanitized, "${{ inputs.target_repo }}")
	assert.Equal(t, 2, strings.Count(sanitized, "${"+envKeys[0]+"}"))
}

func TestBuildToolsMetaRuntimeDataDecodesHTMLEscapedExpression(t *testing.T) {
	rawJSON, err := json.Marshal(map[string]any{
		"description_suffixes": map[string]string{
			"create_issue": "${{ inputs.enabled && 'yes' || 'no' }}",
		},
		"dynamic_tools": []any{},
	})
	require.NoError(t, err)
	input := string(rawJSON)
	// Confirm encoding/json HTML-escaped the expression as expected by this test.
	require.Contains(t, input, `\u0026\u0026`)

	sanitized, envKeys, envValues := buildToolsMetaRuntimeData(input)

	require.Len(t, envKeys, 1)
	assert.Equal(t, "${{ inputs.enabled && 'yes' || 'no' }}", envValues[envKeys[0]])
	assert.NotContains(t, envValues[envKeys[0]], `\u0026`)
	assert.Contains(t, sanitized, "${"+envKeys[0]+"}")
}
