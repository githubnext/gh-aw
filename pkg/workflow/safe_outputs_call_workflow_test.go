//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildCallWorkflowJobs_GeneratesConditionalJobs tests that call-workflow generates
// conditional `uses:` jobs for each worker in the allowlist
func TestBuildCallWorkflowJobs_GeneratesConditionalJobs(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"spring-boot-bugfix", "frontend-dep-upgrade"},
				WorkflowFiles: map[string]string{
					"spring-boot-bugfix":   "./.github/workflows/spring-boot-bugfix.lock.yml",
					"frontend-dep-upgrade": "./.github/workflows/frontend-dep-upgrade.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, "")
	require.NoError(t, err, "Should not error building call-workflow jobs")
	assert.Len(t, jobNames, 2, "Should generate 2 fan-out jobs")
	assert.Contains(t, jobNames, "call-spring-boot-bugfix", "Should generate job for spring-boot-bugfix")
	assert.Contains(t, jobNames, "call-frontend-dep-upgrade", "Should generate job for frontend-dep-upgrade")

	// Check that the jobs were added to the job manager
	job, exists := compiler.jobManager.GetJob("call-spring-boot-bugfix")
	require.True(t, exists, "call-spring-boot-bugfix job should exist in job manager")

	assert.Equal(t, []string{"safe_outputs"}, job.Needs, "Should depend on safe_outputs")
	assert.Equal(t, "needs.safe_outputs.outputs.call_workflow_name == 'spring-boot-bugfix'", job.If, "Should have correct if condition")
	assert.Equal(t, "./.github/workflows/spring-boot-bugfix.lock.yml", job.Uses, "Should use correct workflow path")
	assert.True(t, job.SecretsInherit, "Should inherit secrets")
	assert.Equal(t, "${{ needs.safe_outputs.outputs.call_workflow_payload }}", job.With["payload"], "Should pass payload")
}

// TestBuildCallWorkflowJobs_NoConfig returns nil when call-workflow is not configured
func TestBuildCallWorkflowJobs_NoConfig(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, "")
	require.NoError(t, err, "Should not error when call-workflow is not configured")
	assert.Empty(t, jobNames, "Should not generate any jobs when call-workflow is not configured")
}

// TestBuildCallWorkflowJobs_FallbackPath tests that missing WorkflowFiles falls back to default path
func TestBuildCallWorkflowJobs_FallbackPath(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
				// WorkflowFiles intentionally empty to test fallback
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, "")
	require.NoError(t, err, "Should not error with missing WorkflowFiles")
	assert.Equal(t, []string{"call-worker-a"}, jobNames, "Should generate the job")

	job, exists := compiler.jobManager.GetJob("call-worker-a")
	require.True(t, exists, "Job should exist in job manager")
	assert.Equal(t, "./.github/workflows/worker-a.lock.yml", job.Uses, "Should fall back to default path")
}

// TestSafeOutputsJobOutputs_CallWorkflow tests that safe_outputs job includes
// call_workflow_name and call_workflow_payload outputs when call-workflow is configured
func TestSafeOutputsJobOutputs_CallWorkflow(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	testFile := filepath.Join(workflowsDir, "gateway.md")
	err = os.WriteFile(testFile, []byte("# Gateway"), 0644)
	require.NoError(t, err, "Failed to create test file")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
				WorkflowFiles: map[string]string{
					"worker-a": "./.github/workflows/worker-a.lock.yml",
				},
			},
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", testFile)
	require.NoError(t, err, "Should build safe_outputs job without error")
	require.NotNil(t, job, "Job should be created")

	// Verify call_workflow outputs are declared
	assert.Contains(t, job.Outputs, "call_workflow_name", "Job should declare call_workflow_name output")
	assert.Contains(t, job.Outputs, "call_workflow_payload", "Job should declare call_workflow_payload output")
	assert.Equal(t, "${{ steps.process_safe_outputs.outputs.call_workflow_name }}", job.Outputs["call_workflow_name"])
	assert.Equal(t, "${{ steps.process_safe_outputs.outputs.call_workflow_payload }}", job.Outputs["call_workflow_payload"])
}

// TestSanitizeJobName tests job name sanitization
func TestSanitizeJobName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"spring-boot-bugfix", "spring-boot-bugfix"},
		{"frontend_dep_upgrade", "frontend-dep-upgrade"},
		{"worker123", "worker123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeJobName(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeJobName(%q) should return %q", tt.input, tt.expected)
		})
	}
}

// TestPopulateCallWorkflowFiles_ResolvesPaths tests that populateCallWorkflowFiles resolves
// paths correctly for different file types
func TestPopulateCallWorkflowFiles_ResolvesPaths(t *testing.T) {
	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create .lock.yml for worker-a
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte("name: worker-a\non:\n  workflow_call:\n"), 0644)
	require.NoError(t, err, "Failed to write worker-a.lock.yml")

	// Create .yml for worker-b
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-b.yml"), []byte("name: worker-b\non:\n  workflow_call:\n"), 0644)
	require.NoError(t, err, "Failed to write worker-b.yml")

	markdownPath := filepath.Join(awDir, "gateway.md")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				Workflows: []string{"worker-a", "worker-b"},
			},
		},
	}

	populateCallWorkflowFiles(data, markdownPath)

	assert.Equal(t, "./.github/workflows/worker-a.lock.yml", data.SafeOutputs.CallWorkflow.WorkflowFiles["worker-a"], "Should use .lock.yml for worker-a")
	assert.Equal(t, "./.github/workflows/worker-b.yml", data.SafeOutputs.CallWorkflow.WorkflowFiles["worker-b"], "Should use .yml for worker-b")
}

// TestGenerateCallWorkflowTool_WithInputs tests MCP tool generation from workflow_call inputs
func TestGenerateCallWorkflowTool_WithInputs(t *testing.T) {
	workflowInputs := map[string]any{
		"environment": map[string]any{
			"description": "Target environment",
			"type":        "choice",
			"required":    true,
			"options":     []any{"dev", "staging", "production"},
		},
		"version": map[string]any{
			"description": "Version to deploy",
			"type":        "string",
			"required":    false,
		},
	}

	tool := generateCallWorkflowTool("spring-boot-bugfix", workflowInputs)

	assert.Equal(t, "spring_boot_bugfix", tool["name"], "Tool name should be normalized")
	assert.Equal(t, "spring-boot-bugfix", tool["_call_workflow_name"], "Tool should have _call_workflow_name metadata")
	assert.Contains(t, tool["description"].(string), "spring-boot-bugfix", "Description should mention workflow name")

	schema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "Tool should have inputSchema")

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "inputSchema should have properties")

	// Verify environment input was parsed as choice type with enum
	envProp, exists := properties["environment"]
	require.True(t, exists, "Should have environment property")
	envPropMap, ok := envProp.(map[string]any)
	require.True(t, ok, "environment property should be a map")
	assert.Equal(t, "string", envPropMap["type"], "Choice type should be string")
	assert.Equal(t, []any{"dev", "staging", "production"}, envPropMap["enum"], "Should have enum values")

	// Verify version input was parsed as string type
	versionProp, exists := properties["version"]
	require.True(t, exists, "Should have version property")
	versionPropMap, ok := versionProp.(map[string]any)
	require.True(t, ok, "version property should be a map")
	assert.Equal(t, "string", versionPropMap["type"], "Should be string type")

	// Verify required fields
	required, ok := schema["required"].([]string)
	require.True(t, ok, "Should have required array")
	assert.Contains(t, required, "environment", "environment should be required")
	assert.NotContains(t, required, "version", "version should not be required")
}

// TestGenerateCallWorkflowTool_EmptyInputs tests tool generation with no inputs
func TestGenerateCallWorkflowTool_EmptyInputs(t *testing.T) {
	tool := generateCallWorkflowTool("worker-simple", make(map[string]any))

	assert.Equal(t, "worker_simple", tool["name"], "Tool name should be normalized")
	assert.Equal(t, "worker-simple", tool["_call_workflow_name"], "Tool should have _call_workflow_name metadata")

	schema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "Tool should have inputSchema")

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "inputSchema should have properties")
	assert.Empty(t, properties, "Properties should be empty with no inputs")

	_, hasRequired := schema["required"]
	assert.False(t, hasRequired, "Should not have required array with no required inputs")
}

// TestCallWorkflowJobYAMLOutput tests the YAML output of a call-workflow job
func TestCallWorkflowJobYAMLOutput(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
				WorkflowFiles: map[string]string{
					"worker-a": "./.github/workflows/worker-a.lock.yml",
				},
			},
		},
	}

	_, err := compiler.buildCallWorkflowJobs(workflowData, "")
	require.NoError(t, err, "Should build jobs without error")

	job, exists := compiler.jobManager.GetJob("call-worker-a")
	require.True(t, exists, "Job should exist")

	// Render all jobs to YAML
	yamlOutput := compiler.jobManager.RenderToYAML()

	assert.Contains(t, yamlOutput, "uses: ./.github/workflows/worker-a.lock.yml", "Should contain uses directive")
	assert.Contains(t, yamlOutput, "secrets: inherit", "Should inherit secrets")
	assert.Contains(t, yamlOutput, "payload: ${{ needs.safe_outputs.outputs.call_workflow_payload }}", "Should pass payload")
	assert.Contains(t, yamlOutput, "if: needs.safe_outputs.outputs.call_workflow_name == 'worker-a'", "Should have if condition")

	// Should not have runs-on (reusable workflows don't have this)
	assert.NotContains(t, yamlOutput, "runs-on:", "Reusable workflow jobs should not have runs-on")

	// Suppress unused variable warning
	_ = job
}

// TestExtractWorkflowCallInputs tests extracting inputs from a workflow_call workflow file
func TestExtractWorkflowCallInputs(t *testing.T) {
	tmpDir := t.TempDir()
	workerContent := `name: Worker A
on:
  workflow_call:
    inputs:
      environment:
        description: Target environment
        type: choice
        options:
          - dev
          - staging
          - production
        required: true
      payload:
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	workerFile := filepath.Join(tmpDir, "worker-a.lock.yml")
	err := os.WriteFile(workerFile, []byte(workerContent), 0644)
	require.NoError(t, err, "Failed to write worker file")

	inputs, err := extractWorkflowCallInputs(workerFile)
	require.NoError(t, err, "Should extract inputs without error")
	assert.Contains(t, inputs, "environment", "Should have environment input")
	assert.Contains(t, inputs, "payload", "Should have payload input")

	envInput, ok := inputs["environment"].(map[string]any)
	require.True(t, ok, "environment input should be a map")
	assert.Equal(t, "choice", envInput["type"], "environment should be choice type")
	assert.True(t, envInput["required"].(bool), "environment should be required")
}

// TestCallWorkflowConclusionDependencies tests that conclusion job depends on call-workflow jobs
func TestCallWorkflowConclusionDependencies(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	testFile := filepath.Join(workflowsDir, "gateway.md")
	err = os.WriteFile(testFile, []byte("# Gateway"), 0644)
	require.NoError(t, err, "Failed to create test file")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a", "worker-b"},
				WorkflowFiles: map[string]string{
					"worker-a": "./.github/workflows/worker-a.lock.yml",
					"worker-b": "./.github/workflows/worker-b.lock.yml",
				},
			},
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
			},
		},
		On: "issues",
	}

	err = compiler.buildSafeOutputsJobs(workflowData, "main", testFile)
	require.NoError(t, err, "Should build safe outputs jobs without error")

	// Check conclusion job was created and depends on call-workflow jobs
	conclusionJob, exists := compiler.jobManager.GetJob("conclusion")
	if exists {
		assert.Contains(t, conclusionJob.Needs, "call-worker-a", "Conclusion should depend on call-worker-a")
		assert.Contains(t, conclusionJob.Needs, "call-worker-b", "Conclusion should depend on call-worker-b")
	}

	// Verify call-workflow jobs exist
	_, workerAExists := compiler.jobManager.GetJob("call-worker-a")
	assert.True(t, workerAExists, "call-worker-a job should exist")

	_, workerBExists := compiler.jobManager.GetJob("call-worker-b")
	assert.True(t, workerBExists, "call-worker-b job should exist")
}

// TestBuildCallWorkflowJobs_ForwardsDeclaredInputsFromPayload tests that declared
// workflow_call inputs (except 'payload') are forwarded as fromJSON expressions
// in the generated with: block.
func TestBuildCallWorkflowJobs_ForwardsDeclaredInputsFromPayload(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Worker declares environment, version, and the canonical payload input.
	workerContent := `name: Worker
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
      environment:
        description: Target environment
        type: string
        required: true
      version:
        description: Package version
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644))

	gatewayFile := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayFile, []byte("# Gateway"), 0644))

	compiler := NewCompiler(WithVersion("1.0.0"))
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-a"},
				WorkflowFiles: map[string]string{
					"worker-a": "./.github/workflows/worker-a.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, gatewayFile)
	require.NoError(t, err, "Should not error building call-workflow jobs")
	require.Equal(t, []string{"call-worker-a"}, jobNames)

	job, exists := compiler.jobManager.GetJob("call-worker-a")
	require.True(t, exists, "call-worker-a job should exist")

	// payload is always forwarded as the canonical transport
	assert.Equal(t, "${{ needs.safe_outputs.outputs.call_workflow_payload }}", job.With["payload"],
		"Should always include payload")

	// environment and version should be derived from the payload
	assert.Equal(t,
		"${{ fromJSON(needs.safe_outputs.outputs.call_workflow_payload).environment }}",
		job.With["environment"],
		"Should forward environment from payload")
	assert.Equal(t,
		"${{ fromJSON(needs.safe_outputs.outputs.call_workflow_payload).version }}",
		job.With["version"],
		"Should forward version from payload")

	// payload must appear exactly once as the canonical (non-fromJSON) transport entry.
	// Verify it is not duplicated as a fromJSON expression.
	_, hasPayloadFromJSON := job.With["payload"]
	assert.True(t, hasPayloadFromJSON, "payload key must be present")
	payloadVal, _ := job.With["payload"].(string)
	assert.NotContains(t, payloadVal, "fromJSON",
		"payload canonical entry must be the raw step output, not a fromJSON expression")
}

// TestExtractWorkflowCallInputsFromParsed tests the parsing of workflow_call inputs
// from an already-parsed workflow map
func TestExtractWorkflowCallInputsFromParsed(t *testing.T) {
	workflow := map[string]any{
		"on": map[string]any{
			"workflow_call": map[string]any{
				"inputs": map[string]any{
					"payload": map[string]any{
						"type":     "string",
						"required": false,
					},
				},
			},
		},
	}

	inputs := extractWorkflowCallInputsFromParsed(workflow)
	assert.Contains(t, inputs, "payload", "Should extract payload input")

	payloadInput, ok := inputs["payload"].(map[string]any)
	require.True(t, ok, "payload input should be a map")
	assert.Equal(t, "string", payloadInput["type"], "payload should be string type")
}

// TestCallWorkflowConfig_WithGeneratedYAML tests that the compiled YAML for a gateway workflow
// includes the expected call-workflow fan-out jobs structure
func TestCallWorkflowConfig_WithGeneratedYAML(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create worker A with workflow_call trigger
	workerA := `name: Worker A
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerA), 0644)
	require.NoError(t, err, "Failed to write worker-a.lock.yml")

	// Create worker B with workflow_call trigger
	workerB := `name: Worker B
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-b.lock.yml"), []byte(workerB), 0644)
	require.NoError(t, err, "Failed to write worker-b.lock.yml")

	// Gateway workflow markdown
	gatewayMD := `---
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
safe-outputs:
  add-comment:
    max: 1
  call-workflow:
    workflows:
      - worker-a
      - worker-b
    max: 1
---

# Gateway

Analyse the issue and determine which worker to run.
`
	gatewayFile := filepath.Join(workflowsDir, "gateway.md")
	err = os.WriteFile(gatewayFile, []byte(gatewayMD), 0644)
	require.NoError(t, err, "Failed to write gateway.md")

	// Compile the gateway workflow to a lock file
	err = compiler.CompileWorkflow(gatewayFile)
	require.NoError(t, err, "Should compile without error")

	// Read the generated lock file
	lockFile := gatewayFile[:len(gatewayFile)-len(".md")] + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Should be able to read lock file")
	yamlOutput := string(lockContent)

	// Verify the compiled YAML contains call-workflow fan-out jobs
	assert.Contains(t, yamlOutput, "call-worker-a:", "Should contain call-worker-a job")
	assert.Contains(t, yamlOutput, "call-worker-b:", "Should contain call-worker-b job")
	assert.Contains(t, yamlOutput, "uses: ./.github/workflows/worker-a.lock.yml", "Should use worker-a path")
	assert.Contains(t, yamlOutput, "uses: ./.github/workflows/worker-b.lock.yml", "Should use worker-b path")
	assert.Contains(t, yamlOutput, "secrets: inherit", "Should inherit secrets")
	assert.Contains(t, yamlOutput, "call_workflow_name", "Should reference call_workflow_name")
	assert.Contains(t, yamlOutput, "call_workflow_payload", "Should reference call_workflow_payload")

	// Verify if conditions
	assert.True(t, strings.Contains(yamlOutput, "call_workflow_name == 'worker-a'") ||
		strings.Contains(yamlOutput, "call_workflow_name == \"worker-a\""),
		"Should contain if condition for worker-a")
	assert.True(t, strings.Contains(yamlOutput, "call_workflow_name == 'worker-b'") ||
		strings.Contains(yamlOutput, "call_workflow_name == \"worker-b\""),
		"Should contain if condition for worker-b")
}
