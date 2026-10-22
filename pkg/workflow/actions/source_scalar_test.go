package actions

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditSourceMultilineScalars(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		"steps:\n  - run: | # script\n      echo old\n      echo before\n    shell: bash # shell\n",
		"steps:\n  - name: test\n    run: |- # script\n      echo old\n    shell: bash # shell\n",
		"steps:\n    - name: test\n      run: | # script\n          echo old\n      shell: bash # shell\n",
	} {
		for _, script := range []string{
			"echo \"$TOKEN\"\necho done",
			"echo \"$TOKEN\"\n# shell comment\necho done\n",
			"echo \"$TOKEN\"\necho done\n\n",
		} {
			t.Run(content+script, func(t *testing.T) {
				t.Parallel()
				var before, after map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(content), &before))
				require.NoError(t, yaml.Unmarshal([]byte(content), &after))
				step := after["steps"].([]any)[0].(map[string]any)
				step["run"] = script
				step["env"] = map[string]any{"TOKEN": "${{ secrets.TOKEN }}"}
				result, err := EditSource(content, before, after)
				require.NoError(t, err)
				var actual map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(result), &actual))
				assert.Equal(t, after, actual)
				assert.Contains(t, result, "run: |")
				assert.Contains(t, result, "echo \"$TOKEN\"")
				assert.Equal(t, sourceComments(t, content), sourceComments(t, result))
			})
		}
	}
}

func TestEditSourcePreservesLeadingIndentation(t *testing.T) {
	t.Parallel()
	script := "  echo \"$TOKEN\"\n  echo done\n"
	content := "steps:\n  - run: | # script\n      echo old\n"
	var before, after map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &before))
	require.NoError(t, yaml.Unmarshal([]byte(content), &after))
	after["steps"].([]any)[0].(map[string]any)["run"] = script
	result, err := EditSource(content, before, after)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(result), &actual))
	assert.Equal(t, after, actual)
	assert.Equal(t, sourceComments(t, content), sourceComments(t, result))
}

func TestEditSourceFlowScalars(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		value any
	}{
		{"expression", "${{ secrets.TOKEN }}"},
		{"comma", "mixed,category"},
		{"brackets", "brackets [value]"},
		{"braces", "braces {value}"},
		{"colon", "colon: value"},
		{"comment", "# comment text"},
		{"multiline", "first\nsecond\n"},
		{"nested", map[string]any{"unsafe,key": []any{"${{ inputs.Category }}", "line\nbreak\n"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "env: {KEEP: 'Keep,Quoted'} # env\n"
			var before, after map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(content), &before))
			require.NoError(t, yaml.Unmarshal([]byte(content), &after))
			after["env"].(map[string]any)["NEW"] = tt.value
			result, err := EditSource(content, before, after)
			require.NoError(t, err)
			var actual map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(result), &actual))
			assert.Equal(t, after, actual)
			assert.Contains(t, result, "'Keep,Quoted'")
			assert.Equal(t, sourceComments(t, content), sourceComments(t, result))
		})
	}
}

func TestEditSourceFlowSequenceScalars(t *testing.T) {
	t.Parallel()
	content := "types: ['General # Chat', 'Mixed,Category', '${{ inputs.Category }}', 5] # categories\n"
	var before, after map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &before))
	require.NoError(t, yaml.Unmarshal([]byte(content), &after))
	types := after["types"].([]any)
	types[0], types[1] = "general # chat", "mixed,category"
	result, err := EditSource(content, before, after)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(result), &actual))
	assert.Equal(t, after, actual)
	assert.Equal(t, sourceComments(t, content), sourceComments(t, result))
}
