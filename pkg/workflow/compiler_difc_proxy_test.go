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
			name: "guard policy + custom steps with GH_TOKEN but integrity-proxy disabled",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity":   "approved",
						"integrity-proxy": false,
					},
				},
				CustomSteps: "steps:\n  - name: Fetch issues\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			},
			expected: false,
			desc:     "integrity-proxy: false → proxy not triggered even when guard policy and GH_TOKEN present",
		},
		{
			name: "guard policy + custom steps with GH_TOKEN (proxy enabled by default)",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "approved",
					},
				},
				CustomSteps: "steps:\n  - name: Fetch issues\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			},
			expected: true,
			desc:     "guard policy + custom steps with GH_TOKEN should trigger proxy by default (no feature flag needed)",
		},
		{
			name: "guard policy + custom steps with GH_TOKEN + integrity-proxy explicitly true",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity":   "approved",
						"integrity-proxy": true,
					},
				},
				CustomSteps: "steps:\n  - name: Fetch issues\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			},
			expected: true,
			desc:     "integrity-proxy: true explicitly set should trigger proxy",
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
			name: "guard policy with allowed-repos + custom steps with GH_TOKEN (default enabled)",
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
			desc:     "allowed-repos + min-integrity + GH_TOKEN custom steps should trigger proxy by default",
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
		assert.Contains(t, result, "GITHUB_SERVER_URL:", "step should include GITHUB_SERVER_URL env var")
		assert.Contains(t, result, "DIFC_PROXY_POLICY:", "step should include policy as env var")
		assert.Contains(t, result, "DIFC_PROXY_IMAGE:", "step should include image as env var")
		assert.Contains(t, result, "start_difc_proxy.sh", "step should call the proxy script")
		assert.Contains(t, result, `"allow-only"`, "step should include guard policy JSON in env var")
		assert.Contains(t, result, `"min-integrity":"approved"`, "step should include min-integrity in policy env var")
		assert.Contains(t, result, "ghcr.io/github/gh-aw-mcpg", "step should include container image in env var")
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
		require.Len(t, paths, 2, "should return include path and exclusion path")
		assert.Contains(t, paths[0], "proxy-logs", "first path should include proxy-logs directory")
		assert.Contains(t, paths[1], "proxy-tls", "second path should exclude proxy-tls directory")
		assert.True(t, strings.HasPrefix(paths[1], "!"), "exclusion path should start with !")
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

Test that DIFC proxy is injected by default when min-integrity is set with custom steps using GH_TOKEN.
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

	// Verify the "Set GH_REPO" step is present
	assert.Contains(t, result, "Set GH_REPO for proxied steps",
		"compiled workflow should contain Set GH_REPO step")

	// Verify step ordering: Start proxy must come before Stop proxy
	startIdx := strings.Index(result, "Start DIFC proxy for pre-agent gh calls")
	setRepoIdx := strings.Index(result, "Set GH_REPO for proxied steps")
	stopIdx := strings.Index(result, "Stop DIFC proxy")
	require.Greater(t, startIdx, -1, "start proxy step should be in output")
	require.Greater(t, setRepoIdx, -1, "set GH_REPO step should be in output")
	require.Greater(t, stopIdx, -1, "stop proxy step should be in output")
	assert.Less(t, startIdx, setRepoIdx, "Start DIFC proxy must come before Set GH_REPO")
	assert.Less(t, startIdx, stopIdx, "Start DIFC proxy must come before Stop DIFC proxy")

	// Verify "Set GH_REPO" step is before custom step ("Fetch repo data")
	customStepIdx := strings.Index(result, "Fetch repo data")
	require.Greater(t, customStepIdx, -1, "custom step should be in output")
	assert.Less(t, startIdx, customStepIdx, "Start DIFC proxy must come before custom step")
	assert.Less(t, setRepoIdx, customStepIdx, "Set GH_REPO must come before custom step")

	// Verify proxy stop is before MCP gateway start
	gatewayIdx := strings.Index(result, "Start MCP Gateway")
	require.Greater(t, gatewayIdx, -1, "gateway start step should be in output")
	assert.Less(t, stopIdx, gatewayIdx, "Stop DIFC proxy must come before Start MCP Gateway")

	// Verify start_difc_proxy.sh and stop_difc_proxy.sh are referenced
	assert.Contains(t, result, "start_difc_proxy.sh", "should reference start script")
	assert.Contains(t, result, "stop_difc_proxy.sh", "should reference stop script")

	// Verify the policy JSON in the proxy start step does NOT contain dynamic fields.
	// Note: the MCP gateway config may include approval-labels/blocked-users, but the proxy policy must not.
	// The policy is stored in the DIFC_PROXY_POLICY env var line.
	proxyPolicyLine := ""
	for line := range strings.SplitSeq(result, "\n") {
		if strings.Contains(line, "DIFC_PROXY_POLICY") {
			proxyPolicyLine = line
			break
		}
	}
	require.NotEmpty(t, proxyPolicyLine, "should find the DIFC_PROXY_POLICY env var line")
	assert.NotContains(t, proxyPolicyLine, "blocked-users", "proxy policy should not include blocked-users")
	assert.NotContains(t, proxyPolicyLine, "approval-labels", "proxy policy should not include approval-labels")
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

// TestDIFCProxyNotInjectedWhenIntegrityProxyFalse verifies no proxy injection when
// guard policies are configured but tools.github.integrity-proxy: false is set.
func TestDIFCProxyNotInjectedWhenIntegrityProxyFalse(t *testing.T) {
	workflow := `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
    min-integrity: approved
    integrity-proxy: false
steps:
  - name: Fetch repo data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: gh issue list
---

# Test Workflow

Test that DIFC proxy is NOT injected when integrity-proxy: false is set.
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
			name: "github tool with min-integrity (enabled by default)",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{"min-integrity": "approved"},
				},
			},
			expected: true,
		},
		{
			name: "github tool with min-integrity and integrity-proxy: true",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity":   "approved",
						"integrity-proxy": true,
					},
				},
			},
			expected: true,
		},
		{
			name: "github tool with min-integrity and integrity-proxy: false",
			data: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity":   "approved",
						"integrity-proxy": false,
					},
				},
			},
			expected: false,
		},
		{
			name: "github tool with allowed-repos and min-integrity (enabled by default)",
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

// TestBuildSetGHRepoStepYAML verifies the YAML generated for the "Set GH_REPO" step.
func TestBuildSetGHRepoStepYAML(t *testing.T) {
	result := buildSetGHRepoStepYAML()

	assert.Contains(t, result, "Set GH_REPO for proxied steps", "step name should be present")
	assert.Contains(t, result, "GH_REPO=${GITHUB_REPOSITORY}", "should set GH_REPO from GITHUB_REPOSITORY")
	assert.Contains(t, result, "GITHUB_ENV", "should write GH_REPO to GITHUB_ENV")
	assert.NotContains(t, result, "GH_HOST", "should not modify GH_HOST (proxy must keep routing)")
}

// TestGenerateSetGHRepoAfterDIFCProxyStep verifies that the step is emitted only when
// the DIFC proxy is needed (guard policies configured + pre-agent GH_TOKEN steps).
func TestGenerateSetGHRepoAfterDIFCProxyStep(t *testing.T) {
	c := &Compiler{}

	t.Run("no step when guard policy not configured", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"toolsets": []string{"default"}},
			},
			CustomSteps:   "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			SandboxConfig: &SandboxConfig{},
		}
		c.generateSetGHRepoAfterDIFCProxyStep(&yaml, data)
		assert.Empty(t, yaml.String(), "should not generate step without guard policy")
	})

	t.Run("no step when no GH_TOKEN pre-agent steps", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			SandboxConfig: &SandboxConfig{},
		}
		c.generateSetGHRepoAfterDIFCProxyStep(&yaml, data)
		assert.Empty(t, yaml.String(), "should not generate step without pre-agent GH_TOKEN steps")
	})

	t.Run("generates set GH_REPO step when guard policy and custom steps with GH_TOKEN", func(t *testing.T) {
		var yaml strings.Builder
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
			CustomSteps:   "steps:\n  - name: Fetch\n    env:\n      GH_TOKEN: ${{ github.token }}\n    run: gh issue list",
			SandboxConfig: &SandboxConfig{},
		}
		c.generateSetGHRepoAfterDIFCProxyStep(&yaml, data)

		result := yaml.String()
		require.NotEmpty(t, result, "should generate set GH_REPO step")
		assert.Contains(t, result, "Set GH_REPO for proxied steps", "step name should be present")
		assert.Contains(t, result, "GH_REPO=${GITHUB_REPOSITORY}", "should set GH_REPO from GITHUB_REPOSITORY")
		assert.Contains(t, result, "GITHUB_ENV", "should write to GITHUB_ENV")
		assert.NotContains(t, result, "GH_HOST", "should not touch GH_HOST")
	})
}
