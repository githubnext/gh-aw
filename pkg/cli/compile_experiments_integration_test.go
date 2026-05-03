//go:build integration

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileExperimentsBareArrayForm verifies that the bare-array experiment form compiles
// into a lock file containing GH_AW_EXPERIMENT_SPEC with all declared experiments.
func TestCompileExperimentsBareArrayForm(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-experiments-bare-array.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-experiments-bare-array.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "should read source workflow file")
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "should write workflow to test dir")

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compile command should succeed\nOutput: %s", string(output))

	lockPath := filepath.Join(setup.workflowsDir, "test-experiments-bare-array.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "should read lock file")
	lockStr := string(lockContent)

	// The experiment spec JSON must be present in the compiled lock file.
	assert.Contains(t, lockStr, "GH_AW_EXPERIMENT_SPEC", "lock file should contain GH_AW_EXPERIMENT_SPEC")

	// Both declared experiment names must appear in the spec.
	assert.Contains(t, lockStr, `"prompt_style"`, "lock file should contain prompt_style experiment")
	assert.Contains(t, lockStr, `"model_temp"`, "lock file should contain model_temp experiment")

	// The variants for each experiment must be present.
	assert.Contains(t, lockStr, `"concise"`, "lock file should contain concise variant")
	assert.Contains(t, lockStr, `"verbose"`, "lock file should contain verbose variant")
	assert.Contains(t, lockStr, `"low"`, "lock file should contain low variant")
	assert.Contains(t, lockStr, `"high"`, "lock file should contain high variant")

	// The pick_experiment step must be wired up.
	assert.Contains(t, lockStr, "pick_experiment.cjs", "lock file should reference pick_experiment.cjs")
	assert.Contains(t, lockStr, "pick-experiment", "lock file should contain pick-experiment step")

	t.Logf("Bare-array experiment workflow compiled successfully to %s", lockPath)
}

// TestCompileExperimentsRichSchema verifies that the rich object-form experiment fields
// (hypothesis, secondary_metrics, guardrail_metrics, min_samples) are all
// serialised into GH_AW_EXPERIMENT_SPEC in the compiled lock file.
func TestCompileExperimentsRichSchema(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-experiments-rich-schema.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-experiments-rich-schema.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "should read source workflow file")
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "should write workflow to test dir")

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compile command should succeed\nOutput: %s", string(output))

	lockPath := filepath.Join(setup.workflowsDir, "test-experiments-rich-schema.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "should read lock file")
	lockStr := string(lockContent)

	// Extract and parse the GH_AW_EXPERIMENT_SPEC JSON from the lock file.
	specJSON := extractExperimentSpecFromLock(t, lockStr)
	require.NotEmpty(t, specJSON, "GH_AW_EXPERIMENT_SPEC should be present in lock file")

	// Parse the spec to verify fields round-trip through the compiler correctly.
	var spec map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(specJSON), &spec), "experiment spec should be valid JSON")
	require.Contains(t, spec, "prompt_style", "spec should contain prompt_style experiment")

	var cfg map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(spec["prompt_style"], &cfg), "prompt_style config should be valid JSON")

	// Verify all new fields are present and have correct values.
	assertJSONStringField(t, cfg, "hypothesis", "H0: no change in tokens. H1: concise reduces by >=15%")

	// secondary_metrics
	var secondaryMetrics []string
	require.NoError(t, json.Unmarshal(cfg["secondary_metrics"], &secondaryMetrics), "secondary_metrics should parse")
	assert.Equal(t, []string{"duration_ms", "discussion_word_count"}, secondaryMetrics, "secondary_metrics should match")

	// guardrail_metrics
	var guardrails []struct {
		Name      string `json:"name"`
		Threshold string `json:"threshold"`
	}
	require.NoError(t, json.Unmarshal(cfg["guardrail_metrics"], &guardrails), "guardrail_metrics should parse")
	require.Len(t, guardrails, 2, "should have 2 guardrail metrics")
	assert.Equal(t, "success_rate", guardrails[0].Name, "first guardrail name")
	assert.Equal(t, ">=0.95", guardrails[0].Threshold, "first guardrail threshold")
	assert.Equal(t, "empty_output_rate", guardrails[1].Name, "second guardrail name")
	assert.Equal(t, "==0", guardrails[1].Threshold, "second guardrail threshold")

	// min_samples: should be 25 (integer round-trips correctly from YAML uint64)
	var minSamples int
	require.NoError(t, json.Unmarshal(cfg["min_samples"], &minSamples), "min_samples should parse")
	assert.Equal(t, 25, minSamples, "min_samples should be 25")

	// issue: should be 1234 (integer round-trips correctly from YAML uint64)
	var issue int
	require.NoError(t, json.Unmarshal(cfg["issue"], &issue), "issue should parse")
	assert.Equal(t, 1234, issue, "issue should be 1234")

	// variants
	var variants []string
	require.NoError(t, json.Unmarshal(cfg["variants"], &variants), "variants should parse")
	assert.Equal(t, []string{"concise", "detailed"}, variants, "variants should match")

	// weight
	var weights []int
	require.NoError(t, json.Unmarshal(cfg["weight"], &weights), "weight should parse")
	assert.Equal(t, []int{60, 40}, weights, "weight should match")

	// analysis_type
	assertJSONStringField(t, cfg, "analysis_type", "t_test")

	// tags
	var tags []string
	require.NoError(t, json.Unmarshal(cfg["tags"], &tags), "tags should parse")
	assert.Equal(t, []string{"cost", "prompting"}, tags, "tags should match")

	// notify.issue
	var notify struct {
		Issue int `json:"issue"`
	}
	require.NoError(t, json.Unmarshal(cfg["notify"], &notify), "notify should parse")
	assert.Equal(t, 5678, notify.Issue, "notify.issue should be 5678")

	t.Logf("Rich-schema experiment workflow compiled successfully to %s", lockPath)
}

// extractExperimentSpecFromLock extracts the raw JSON value of GH_AW_EXPERIMENT_SPEC
// from a compiled lock file, unescaping YAML single-quote doubling.
func extractExperimentSpecFromLock(t *testing.T, lockStr string) string {
	t.Helper()
	const marker = "GH_AW_EXPERIMENT_SPEC: '"
	idx := strings.Index(lockStr, marker)
	if idx < 0 {
		return ""
	}
	rest := lockStr[idx+len(marker):]
	// The value is a YAML single-quoted scalar; the closing quote is the first unescaped '
	// (doubled '' represent a literal single-quote in YAML §7.3.3).
	var sb strings.Builder
	for i := 0; i < len(rest); i++ {
		if rest[i] == '\'' {
			if i+1 < len(rest) && rest[i+1] == '\'' {
				sb.WriteByte('\'')
				i++ // skip second quote of escaped pair
			} else {
				break // end of scalar
			}
		} else {
			sb.WriteByte(rest[i])
		}
	}
	return sb.String()
}

// assertJSONStringField decodes a JSON string field and asserts its value.
func assertJSONStringField(t *testing.T, m map[string]json.RawMessage, key, want string) {
	t.Helper()
	raw, ok := m[key]
	require.True(t, ok, "field %q should be present", key)
	var got string
	require.NoError(t, json.Unmarshal(raw, &got), "field %q should be a JSON string", key)
	assert.Equal(t, want, got, "field %q value", key)
}
