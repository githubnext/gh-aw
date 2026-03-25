//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHasDIFCProxyNeeded verifies that DIFC proxy injection is triggered only
// when guard policies are configured AND pre-agent steps have GH_TOKEN.
func TestHasDIFCProxyNeeded(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
		desc     string
	}{
		{
			name:     "nil workflow data",
			data:     nil,
			expected: false,
			desc:     "nil data should never need proxy",
		},
		{
			name:     "no github tool",
			data:     &WorkflowData{Tools: map[string]any{}},
			expected: false,
			desc:     "no github tool means no guard policy, proxy not needed",
		},
		{
			name: "github tool disabled",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": false,
				},
			},
			expected: false,
			desc:     "disabled github tool should not trigger proxy",
		},
		{
			name: "github tool without guard policy",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{"toolsets": []string{"default"}},
				},
				CustomSteps: "steps:\n  - name: Fetch data\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			},
			expected: false,
			desc:     "no guard policy (auto-lockdown only) should not trigger proxy",
		},
		{
			name: "guard policy configured but no pre-agent steps with GH_TOKEN",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "approved",
					},
				},
			},
			expected: false,
			desc:     "guard policy without GH_TOKEN pre-agent steps should not trigger proxy",
		},
		{
			name: "guard policy + custom steps with GH_TOKEN",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "approved",
					},
				},
				CustomSteps: "steps:\n  - name: Fetch issues\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			},
			expected: true,
			desc:     "guard policy + custom steps with GH_TOKEN should trigger proxy",
		},
		{
			name: "guard policy + repo-memory configured",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "approved",
						"repos":         "all",
					},
				},
				RepoMemoryConfig: &RepoMemoryConfig{
					Memories: []RepoMemoryEntry{{ID: "memory"}},
				},
			},
			expected: false,
			desc:     "guard policy + repo-memory should NOT trigger proxy: repo-memory clones use direct git URLs, not GH_HOST",
		},
		{
			name: "guard policy with allowed-repos + custom steps with GH_TOKEN",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "merged",
						"allowed-repos": []string{"owner/repo"},
					},
				},
				CustomSteps: "steps:\n  - name: Fetch PRs\n    env:\n      GH_TOKEN: ${{ secrets.MY_TOKEN }}\n    run: gh pr list",
			},
			expected: true,
			desc:     "allowed-repos + min-integrity + GH_TOKEN custom steps should trigger proxy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasDIFCProxyNeeded(tt.data)
			assert.Equal(t, tt.expected, got, "hasDIFCProxyNeeded: %s", tt.desc)
		})
	}
}

// TestHasPreAgentStepsWithGHToken verifies detection of pre-agent steps with GH_TOKEN.
func TestHasPreAgentStepsWithGHToken(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: false,
		},
		{
			name:     "empty data",
			data:     &WorkflowData{},
			expected: false,
		},
		{
			name: "custom steps without GH_TOKEN",
			data: &WorkflowData{
				CustomSteps: "steps:\n  - name: Build\n    run: make build\n",
			},
			expected: false,
		},
		{
			name: "custom steps with GH_TOKEN",
			data: &WorkflowData{
				CustomSteps: "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list\n",
			},
			expected: true,
		},
		{
			name: "repo-memory configured",
			data: &WorkflowData{
				RepoMemoryConfig: &RepoMemoryConfig{
					Memories: []RepoMemoryEntry{{ID: "memory"}},
				},
			},
			expected: false,
			// repo-memory clone steps use direct "git clone https://x-access-token:${GH_TOKEN}@..."
			// URLs derived from GITHUB_SERVER_URL, not GH_HOST, so the proxy does not intercept them.
		},
		{
			name: "repo-memory with empty memories (no clone steps generated)",
			data: &WorkflowData{
				RepoMemoryConfig: &RepoMemoryConfig{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPreAgentStepsWithGHToken(tt.data)
			assert.Equal(t, tt.expected, got, "test: %s", tt.name)
		})
	}
}

// TestGetDIFCProxyPolicyJSON verifies that the proxy policy JSON contains
// only the static fields (min-integrity and repos) without dynamic expressions.
func TestGetDIFCProxyPolicyJSON(t *testing.T) {
	tests := []struct {
		name             string
		githubTool       any
		expectedContains []string
		expectedAbsent   []string
		expectEmpty      bool
	}{
		{
			name:        "nil tool",
			githubTool:  nil,
			expectEmpty: true,
		},
		{
			name:        "non-map tool",
			githubTool:  false,
			expectEmpty: true,
		},
		{
			name: "min-integrity only",
			githubTool: map[string]any{
				"min-integrity": "approved",
			},
			expectedContains: []string{`"allow-only"`, `"min-integrity":"approved"`, `"repos":"all"`},
			expectedAbsent:   []string{"blocked-users", "approval-labels", "steps.parse-guard-vars", "__GH_AW_GUARD_EXPR"},
		},
		{
			name: "min-integrity and repos",
			githubTool: map[string]any{
				"min-integrity": "merged",
				"repos":         "all",
			},
			expectedContains: []string{`"allow-only"`, `"min-integrity":"merged"`, `"repos":"all"`},
			expectedAbsent:   []string{"blocked-users", "approval-labels"},
		},
		{
			name: "allowed-repos (preferred field name)",
			githubTool: map[string]any{
				"min-integrity": "unapproved",
				"allowed-repos": "owner/*",
			},
			expectedContains: []string{`"min-integrity":"unapproved"`, `"repos":"owner/*"`},
			expectedAbsent:   []string{"blocked-users", "approval-labels"},
		},
		{
			name: "tool without guard policy fields",
			githubTool: map[string]any{
				"toolsets": []string{"default"},
			},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDIFCProxyPolicyJSON(tt.githubTool)

			if tt.expectEmpty {
				assert.Empty(t, got, "policy JSON should be empty for: %s", tt.name)
				return
			}

			require.NotEmpty(t, got, "policy JSON should not be empty for: %s", tt.name)

			for _, s := range tt.expectedContains {
				assert.Contains(t, got, s, "policy JSON should contain %q for: %s", s, tt.name)
			}
			for _, s := range tt.expectedAbsent {
				assert.NotContains(t, got, s, "policy JSON should NOT contain %q for: %s", s, tt.name)
			}
		})
	}
}

// TestGenerateStartDIFCProxyStep verifies the YAML generated for the proxy start step.
func TestGenerateStartDIFCProxyStep(t *testing.T) {
	c := &Compiler{}

	t.Run("no proxy when guard policy not configured", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			CustomSteps:   "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			SandboxConfig: &SandboxConfig{},
		}
		c.generateStartDIFCProxyStep(&yaml, data)
		assert.Empty(t, yaml.String(), "should not generate proxy step without guard policy")
	})

	t.Run("no proxy when no GH_TOKEN pre-agent steps", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			SandboxConfig: &SandboxConfig{},
		}
		c.generateStartDIFCProxyStep(&yaml, data)
		assert.Empty(t, yaml.String(), "should not generate proxy step without pre-agent GH_TOKEN steps")
	})

	t.Run("generates start step with guard policy and custom steps", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
			CustomSteps:   "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)
		c.generateStartDIFCProxyStep(&yaml, data)

		result := yaml.String()
		require.NotEmpty(t, result, "should generate proxy start step")
		assert.Contains(t, result, "Start DIFC proxy for pre-agent gh calls", "step name should be present")
		assert.Contains(t, result, "GH_TOKEN:", "step should include GH_TOKEN env var")
		assert.Contains(t, result, "start_difc_proxy.sh", "step should call the proxy script")
		assert.Contains(t, result, `"allow-only"`, "step should include guard policy JSON")
		assert.Contains(t, result, `"min-integrity":"approved"`, "step should include min-integrity in policy")
		assert.Contains(t, result, "ghcr.io/github/gh-aw-mcpg", "step should include container image")
		assert.NotContains(t, result, "blocked-users", "proxy policy should not include dynamic blocked-users")
		assert.NotContains(t, result, "approval-labels", "proxy policy should not include dynamic approval-labels")
	})
}

// TestGenerateStopDIFCProxyStep verifies the YAML generated for the proxy stop step.
func TestGenerateStopDIFCProxyStep(t *testing.T) {
	c := &Compiler{}

	t.Run("no stop step when proxy not needed", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			SandboxConfig: &SandboxConfig{},
		}
		c.generateStopDIFCProxyStep(&yaml, data)
		assert.Empty(t, yaml.String(), "should not generate stop step when proxy not needed")
	})

	t.Run("generates stop step when proxy is needed", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			CustomSteps:   "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			SandboxConfig: &SandboxConfig{},
		}
		c.generateStopDIFCProxyStep(&yaml, data)

		result := yaml.String()
		require.NotEmpty(t, result, "should generate proxy stop step")
		assert.Contains(t, result, "Stop DIFC proxy", "step name should be present")
		assert.Contains(t, result, "stop_difc_proxy.sh", "step should call the stop script")
	})
}

// TestDIFCProxyLogPaths verifies the artifact paths returned for DIFC proxy logs.
func TestDIFCProxyLogPaths(t *testing.T) {
	t.Run("no log paths when proxy not needed", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
		}
		paths := difcProxyLogPaths(data)
		assert.Empty(t, paths, "should return no log paths when proxy not needed")
	})

	t.Run("returns proxy-logs path when proxy is needed", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			CustomSteps: "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
		}
		paths := difcProxyLogPaths(data)
		require.Len(t, paths, 2, "should return inclusion path and proxy-tls exclusion path")
		assert.Contains(t, paths[0], "proxy-logs", "first path should include proxy-logs directory")
		assert.Equal(t, "!/tmp/gh-aw/proxy-logs/proxy-tls/", paths[1], "second path should exclude proxy-tls directory")
	})
}

// TestDIFCProxyStepOrderInCompiledWorkflow verifies that proxy steps are injected
// at the correct positions in the generated workflow YAML.
func TestDIFCProxyStepOrderInCompiledWorkflow(t *testing.T) {
	workflow := `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
    min-integrity: approved
steps:
  - name: Fetch repo data
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      gh issue list -R $GITHUB_REPOSITORY --state open --limit 500 \
        --json number,labels > /tmp/gh-aw/issues.json 2>/dev/null \
        || echo '[]' > /tmp/gh-aw/issues.json
---

# Test Workflow

Test that DIFC proxy is injected when min-integrity is set with custom steps using GH_TOKEN.
`
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
	require.NoError(t, err, "parsing should succeed")

	result, err := compiler.CompileToYAML(data, "test-workflow.md")
	require.NoError(t, err, "compilation should succeed")

	// Verify proxy start step is present
	assert.Contains(t, result, "Start DIFC proxy for pre-agent gh calls",
		"compiled workflow should contain proxy start step")

	// Verify proxy stop step is present
	assert.Contains(t, result, "Stop DIFC proxy",
		"compiled workflow should contain proxy stop step")

	// Verify step ordering: Start proxy must come before Stop proxy
	startIdx := strings.Index(result, "Start DIFC proxy for pre-agent gh calls")
	stopIdx := strings.Index(result, "Stop DIFC proxy")
	require.Greater(t, startIdx, -1, "start proxy step should be in output")
	require.Greater(t, stopIdx, -1, "stop proxy step should be in output")
	assert.Less(t, startIdx, stopIdx, "Start DIFC proxy must come before Stop DIFC proxy")

	// Verify proxy start is before custom step ("Fetch repo data")
	customStepIdx := strings.Index(result, "Fetch repo data")
	require.Greater(t, customStepIdx, -1, "custom step should be in output")
	assert.Less(t, startIdx, customStepIdx, "Start DIFC proxy must come before custom step")

	// Verify proxy stop is before MCP gateway start
	gatewayIdx := strings.Index(result, "Start MCP Gateway")
	require.Greater(t, gatewayIdx, -1, "gateway start step should be in output")
	assert.Less(t, stopIdx, gatewayIdx, "Stop DIFC proxy must come before Start MCP Gateway")

	// Verify start_difc_proxy.sh and stop_difc_proxy.sh are referenced
	assert.Contains(t, result, "start_difc_proxy.sh", "should reference start script")
	assert.Contains(t, result, "stop_difc_proxy.sh", "should reference stop script")

	// Verify the policy JSON in the proxy start step does NOT contain dynamic fields.
	// Note: the MCP gateway config may include approval-labels/blocked-users, but the proxy policy must not.
	proxyStartLine := ""
	for line := range strings.SplitSeq(result, "\n") {
		if strings.Contains(line, "start_difc_proxy.sh") {
			proxyStartLine = line
			break
		}
	}
	require.NotEmpty(t, proxyStartLine, "should find the start_difc_proxy.sh invocation line")
	assert.NotContains(t, proxyStartLine, "blocked-users", "proxy policy invocation should not include blocked-users")
	assert.NotContains(t, proxyStartLine, "approval-labels", "proxy policy invocation should not include approval-labels")
}

// TestDIFCProxyNotInjectedWithoutGuardPolicy verifies no proxy injection without guard policy.
func TestDIFCProxyNotInjectedWithoutGuardPolicy(t *testing.T) {
	workflow := `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
steps:
  - name: Fetch repo data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: gh issue list
---

# Test Workflow

Test that DIFC proxy is NOT injected when min-integrity is not set.
`
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
	require.NoError(t, err, "parsing should succeed")

	result, err := compiler.CompileToYAML(data, "test-workflow.md")
	require.NoError(t, err, "compilation should succeed")

	assert.NotContains(t, result, "Start DIFC proxy",
		"compiled workflow should NOT contain proxy start step without guard policy")
	assert.NotContains(t, result, "Stop DIFC proxy",
		"compiled workflow should NOT contain proxy stop step without guard policy")
}

// TestHasDIFCGuardsConfigured verifies the base guard policy check.
func TestHasDIFCGuardsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: false,
		},
		{
			name:     "no github tool",
			data:     &WorkflowData{Tools: map[string]any{}},
			expected: false,
		},
		{
			name: "github tool without guard policy",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{"toolsets": []string{"default"}},
				},
			},
			expected: false,
		},
		{
			name: "github tool with min-integrity",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{"min-integrity": "approved"},
				},
			},
			expected: true,
		},
		{
			name: "github tool with allowed-repos and min-integrity",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"allowed-repos": "all",
						"min-integrity": "merged",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasDIFCGuardsConfigured(tt.data)
			assert.Equal(t, tt.expected, got, "hasDIFCGuardsConfigured: %s", tt.name)
		})
	}
}

// TestDIFCProxyInjectedInIndexingJob verifies that DIFC proxy steps are injected
// around qmd index-building steps when guard policies are configured.
func TestDIFCProxyInjectedInIndexingJob(t *testing.T) {
	c := &Compiler{}

	t.Run("no proxy when guard policy not configured", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			QmdConfig:     &QmdToolConfig{},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		// hasDIFCGuardsConfigured should return false
		assert.False(t, hasDIFCGuardsConfigured(data), "no guard policy should not need DIFC proxy")

		// buildStartDIFCProxyStepYAML should return empty when no guard policy
		// (won't be called in practice, but validate the logic)
		data.Tools["github"] = map[string]any{"toolsets": []string{"default"}}
		result := c.buildStartDIFCProxyStepYAML(data)
		assert.Empty(t, result, "no guard policy → no start step")
	})

	t.Run("proxy steps present when guard policy configured", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			QmdConfig:     &QmdToolConfig{},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		assert.True(t, hasDIFCGuardsConfigured(data), "guard policy configured → DIFC proxy needed in indexing")

		startStep := c.buildStartDIFCProxyStepYAML(data)
		require.NotEmpty(t, startStep, "should generate start proxy step for indexing job")
		assert.Contains(t, startStep, "Start DIFC proxy for pre-agent gh calls")
		assert.Contains(t, startStep, "start_difc_proxy.sh")
		assert.Contains(t, startStep, `"allow-only"`)

		stopStep := buildStopDIFCProxyStepYAML()
		assert.Contains(t, stopStep, "Stop DIFC proxy")
		assert.Contains(t, stopStep, "stop_difc_proxy.sh")
	})

	t.Run("buildQmdIndexingJob wraps steps with proxy when guard policy configured", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			QmdConfig: &QmdToolConfig{
				Searches: []*QmdSearchEntry{{Query: "repo:owner/repo language:Markdown"}},
				CacheKey: "qmd-test",
			},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildQmdIndexingJob(data)
		require.NoError(t, err, "buildQmdIndexingJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		require.Contains(t, allSteps, "Start DIFC proxy for pre-agent gh calls",
			"indexing job should include proxy start step when guard policy is configured")
		require.Contains(t, allSteps, "Stop DIFC proxy",
			"indexing job should include proxy stop step when guard policy is configured")

		// Proxy start must come before the qmd index step and proxy stop must come after.
		startIdx := strings.Index(allSteps, "Start DIFC proxy for pre-agent gh calls")
		stopIdx := strings.Index(allSteps, "Stop DIFC proxy")
		assert.Less(t, startIdx, stopIdx, "Start proxy must come before Stop proxy in indexing job")
	})

	t.Run("buildQmdIndexingJob has no proxy steps without guard policy", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			QmdConfig: &QmdToolConfig{
				Searches: []*QmdSearchEntry{{Query: "repo:owner/repo language:Markdown"}},
				CacheKey: "qmd-test",
			},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildQmdIndexingJob(data)
		require.NoError(t, err, "buildQmdIndexingJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		assert.NotContains(t, allSteps, "Start DIFC proxy",
			"indexing job should NOT include proxy start step without guard policy")
		assert.NotContains(t, allSteps, "Stop DIFC proxy",
			"indexing job should NOT include proxy stop step without guard policy")
	})
}

// TestDIFCProxyInjectedInActivationJob verifies that DIFC proxy steps are injected
// into the activation job when guard policies are configured.
func TestDIFCProxyInjectedInActivationJob(t *testing.T) {
	t.Run("proxy injected in activation job when guard policy configured", func(t *testing.T) {
		workflow := `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
    min-integrity: approved
---

# Test Workflow

Test that DIFC proxy is injected into the activation job when min-integrity is set.
`
		compiler := NewCompiler()
		data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
		require.NoError(t, err, "parsing should succeed")

		result, err := compiler.CompileToYAML(data, "test-workflow.md")
		require.NoError(t, err, "compilation should succeed")

		// Find the activation job section
		activationIdx := strings.Index(result, "activation:")
		require.Greater(t, activationIdx, -1, "activation job should be present")

		// Find the agent job section (to bound our search to the activation job)
		agentIdx := strings.Index(result, "agent:")
		require.Greater(t, agentIdx, -1, "agent job should be present")

		// Extract activation job content (before agent job)
		activationSection := result[activationIdx:agentIdx]

		// Proxy start must be present in activation job
		assert.Contains(t, activationSection, "Start DIFC proxy for pre-agent gh calls",
			"activation job should contain proxy start step when guard policy is configured")

		// Proxy stop must be present in activation job
		assert.Contains(t, activationSection, "Stop DIFC proxy",
			"activation job should contain proxy stop step when guard policy is configured")

		// Proxy start must come before proxy stop
		startIdx := strings.Index(activationSection, "Start DIFC proxy for pre-agent gh calls")
		stopIdx := strings.Index(activationSection, "Stop DIFC proxy")
		assert.Less(t, startIdx, stopIdx, "Start proxy must come before Stop proxy in activation job")

		// Proxy start must come before the first github-script step (add reaction, timestamp check, etc.)
		// Verify start comes before the "Upload activation artifact" step
		uploadIdx := strings.Index(activationSection, "Upload activation artifact")
		require.Greater(t, uploadIdx, -1, "activation artifact upload step should be present")
		assert.Less(t, stopIdx, uploadIdx, "Stop DIFC proxy must come before artifact upload")
	})

	t.Run("proxy not injected in activation job without guard policy", func(t *testing.T) {
		workflow := `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that DIFC proxy is NOT injected into the activation job when min-integrity is not set.
`
		compiler := NewCompiler()
		data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
		require.NoError(t, err, "parsing should succeed")

		result, err := compiler.CompileToYAML(data, "test-workflow.md")
		require.NoError(t, err, "compilation should succeed")

		// Find the activation job section
		activationIdx := strings.Index(result, "activation:")
		require.Greater(t, activationIdx, -1, "activation job should be present")

		agentIdx := strings.Index(result, "agent:")
		require.Greater(t, agentIdx, -1, "agent job should be present")

		activationSection := result[activationIdx:agentIdx]

		assert.NotContains(t, activationSection, "Start DIFC proxy",
			"activation job should NOT contain proxy start step without guard policy")
		assert.NotContains(t, activationSection, "Stop DIFC proxy",
			"activation job should NOT contain proxy stop step without guard policy")
	})

	t.Run("buildActivationJob includes proxy steps when guard policy configured", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			AI:            "copilot",
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildActivationJob(data, false, "", "test-workflow.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		assert.Contains(t, allSteps, "Start DIFC proxy for pre-agent gh calls",
			"activation job should include proxy start step when guard policy is configured")
		assert.Contains(t, allSteps, "Stop DIFC proxy",
			"activation job should include proxy stop step when guard policy is configured")

		startIdx := strings.Index(allSteps, "Start DIFC proxy for pre-agent gh calls")
		stopIdx := strings.Index(allSteps, "Stop DIFC proxy")
		assert.Less(t, startIdx, stopIdx, "Start proxy must come before Stop proxy in activation job")

		// Stop proxy must come before artifact upload
		uploadIdx := strings.Index(allSteps, "Upload activation artifact")
		require.Greater(t, uploadIdx, -1, "artifact upload step should be present")
		assert.Less(t, stopIdx, uploadIdx, "Stop DIFC proxy must come before artifact upload")
	})

	t.Run("buildActivationJob has no proxy steps without guard policy", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			AI:            "copilot",
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildActivationJob(data, false, "", "test-workflow.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		assert.NotContains(t, allSteps, "Start DIFC proxy",
			"activation job should NOT include proxy start step without guard policy")
		assert.NotContains(t, allSteps, "Stop DIFC proxy",
			"activation job should NOT include proxy stop step without guard policy")
	})
}

// TestDIFCProxyInjectedInPreActivationJob verifies that DIFC proxy steps are injected
// into the pre-activation job (which contains user-defined on.steps and compiler-added
// checks) when guard policies are configured.
func TestDIFCProxyInjectedInPreActivationJob(t *testing.T) {
	t.Run("proxy injected in pre-activation job when guard policy configured", func(t *testing.T) {
		// Note: ParseWorkflowString does not run processOnSectionAndFilters so OnSteps is
		// empty in this path. The pre-activation job is still created because the workflow
		// uses on.issues (an unsafe event) triggering the membership check.
		// The proxy injection is gated on hasDIFCGuardsConfigured which only requires
		// min-integrity to be set in the github tool config.
		workflow := `---
on:
  issues:
    types: [opened]
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
    min-integrity: approved
permissions:
  issues: read
  pull-requests: read
  contents: read
---

# Test Workflow

Test that DIFC proxy is injected into the pre-activation job when min-integrity is set.
`
		compiler := NewCompiler()
		data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
		require.NoError(t, err, "parsing should succeed")

		result, err := compiler.CompileToYAML(data, "test-workflow.md")
		require.NoError(t, err, "compilation should succeed")

		// Extract the pre_activation job section from the full YAML.
		// Jobs may appear in any order in the map; find "pre_activation:" and take from there.
		preActivationMarker := "\n  pre_activation:"
		preActivationIdx := strings.Index(result, preActivationMarker)
		require.Greater(t, preActivationIdx, -1, "pre_activation job should be present in compiled YAML")

		preActivationSection := result[preActivationIdx:]

		// Proxy start must be present in pre_activation section
		assert.Contains(t, preActivationSection, "Start DIFC proxy for pre-agent gh calls",
			"pre-activation job should contain proxy start step when guard policy is configured")

		// Proxy stop must be present in pre_activation section
		assert.Contains(t, preActivationSection, "Stop DIFC proxy",
			"pre-activation job should contain proxy stop step when guard policy is configured")

		// Proxy start must come before proxy stop
		startIdx := strings.Index(preActivationSection, "Start DIFC proxy for pre-agent gh calls")
		stopIdx := strings.Index(preActivationSection, "Stop DIFC proxy")
		assert.Less(t, startIdx, stopIdx, "Start proxy must come before Stop proxy in pre-activation job")
	})

	t.Run("proxy not injected in pre-activation job without guard policy", func(t *testing.T) {
		workflow := `---
on:
  issues:
    types: [opened]
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
permissions:
  issues: read
  pull-requests: read
---

# Test Workflow

Test that DIFC proxy is NOT injected into the pre-activation job when min-integrity is not set.
`
		compiler := NewCompiler()
		data, err := compiler.ParseWorkflowString(workflow, "test-workflow.md")
		require.NoError(t, err, "parsing should succeed")

		result, err := compiler.CompileToYAML(data, "test-workflow.md")
		require.NoError(t, err, "compilation should succeed")

		preActivationMarker := "\n  pre_activation:"
		preActivationIdx := strings.Index(result, preActivationMarker)
		require.Greater(t, preActivationIdx, -1, "pre_activation job should be present in compiled YAML")

		preActivationSection := result[preActivationIdx:]

		assert.NotContains(t, preActivationSection, "Start DIFC proxy",
			"pre-activation job should NOT contain proxy start step without guard policy")
		assert.NotContains(t, preActivationSection, "Stop DIFC proxy",
			"pre-activation job should NOT contain proxy stop step without guard policy")
	})

	t.Run("buildPreActivationJob includes proxy steps when guard policy configured with on.steps", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			AI: "copilot",
			OnSteps: []map[string]any{
				{
					"name": "Custom gate check",
					"id":   "gate",
					"uses": "actions/github-script@v7",
					"with": map[string]any{
						"script": "core.setOutput('approved', 'true')",
					},
				},
			},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildPreActivationJob(data, false)
		require.NoError(t, err, "buildPreActivationJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		assert.Contains(t, allSteps, "Start DIFC proxy for pre-agent gh calls",
			"pre-activation job should include proxy start step when guard policy is configured")
		assert.Contains(t, allSteps, "Stop DIFC proxy",
			"pre-activation job should include proxy stop step when guard policy is configured")

		startIdx := strings.Index(allSteps, "Start DIFC proxy for pre-agent gh calls")
		stopIdx := strings.Index(allSteps, "Stop DIFC proxy")
		assert.Less(t, startIdx, stopIdx, "Start proxy must come before Stop proxy in pre-activation job")

		// User-defined on.step must be between start and stop
		gateIdx := strings.Index(allSteps, "Custom gate check")
		require.Greater(t, gateIdx, -1, "on.steps should appear in pre-activation steps")
		assert.Less(t, startIdx, gateIdx, "Proxy start must come before user-defined on.steps")
		assert.Less(t, gateIdx, stopIdx, "on.steps must come before proxy stop")
	})

	t.Run("buildPreActivationJob has no proxy steps without guard policy", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			AI: "copilot",
			// OnSteps is required to create a valid pre-activation job without
			// permission checks or stop-time.
			OnSteps: []map[string]any{
				{
					"name": "Custom gate check",
					"id":   "gate",
					"uses": "actions/github-script@v7",
					"with": map[string]any{
						"script": "core.setOutput('approved', 'true')",
					},
				},
			},
			SandboxConfig: &SandboxConfig{},
		}
		ensureDefaultMCPGatewayConfig(data)

		job, err := c.buildPreActivationJob(data, false)
		require.NoError(t, err, "buildPreActivationJob should succeed")
		require.NotNil(t, job, "job should not be nil")

		allSteps := strings.Join(job.Steps, "\n")
		assert.NotContains(t, allSteps, "Start DIFC proxy",
			"pre-activation job should NOT include proxy start step without guard policy")
		assert.NotContains(t, allSteps, "Stop DIFC proxy",
			"pre-activation job should NOT include proxy stop step without guard policy")
	})
}
