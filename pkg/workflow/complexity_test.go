package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplexityTier_String(t *testing.T) {
	tests := []struct {
		name string
		tier ComplexityTier
		want string
	}{
		{
			name: "basic tier",
			tier: ComplexityBasic,
			want: "basic",
		},
		{
			name: "intermediate tier",
			tier: ComplexityIntermediate,
			want: "intermediate",
		},
		{
			name: "advanced tier",
			tier: ComplexityAdvanced,
			want: "advanced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tier.String(), "String() should return correct value")
		})
	}
}

func TestComplexityTier_IsValid(t *testing.T) {
	tests := []struct {
		name string
		tier ComplexityTier
		want bool
	}{
		{
			name: "valid basic",
			tier: ComplexityBasic,
			want: true,
		},
		{
			name: "valid intermediate",
			tier: ComplexityIntermediate,
			want: true,
		},
		{
			name: "valid advanced",
			tier: ComplexityAdvanced,
			want: true,
		},
		{
			name: "invalid tier",
			tier: ComplexityTier("invalid"),
			want: false,
		},
		{
			name: "empty tier",
			tier: ComplexityTier(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tier.IsValid(), "IsValid() should return correct value")
		})
	}
}

func TestDetectWorkflowComplexity_Basic(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		description string
		wantTier    ComplexityTier
		wantScore   int
	}{
		{
			name: "minimal workflow - single standard trigger",
			content: `---
name: Simple Workflow
on: push
---
Do something simple`,
			description: "A simple workflow",
			wantTier:    ComplexityBasic,
			wantScore:   0,
		},
		{
			name: "single trigger with simple description",
			content: `---
name: Test Workflow
on: pull_request
---
Test the code`,
			description: "Run tests on pull requests",
			wantTier:    ComplexityBasic,
			wantScore:   0,
		},
		{
			name: "workflow_dispatch trigger",
			content: `---
name: Manual Workflow
on: workflow_dispatch
---
Manual trigger`,
			description: "Manual workflow",
			wantTier:    ComplexityBasic,
			wantScore:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectWorkflowComplexity(tt.content, tt.description)
			require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
			assert.Equal(t, tt.wantTier, result.Tier, "Tier should match expected")
			assert.Equal(t, tt.wantScore, result.Score, "Score should match expected")
		})
	}
}

func TestDetectWorkflowComplexity_Intermediate(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		description string
		wantTier    ComplexityTier
		minScore    int
		maxScore    int
	}{
		{
			name: "multiple triggers with conditional logic",
			content: `---
name: Multi-trigger Workflow
on: [push, pull_request]
---
Run on multiple events`,
			description: "If code changes, run tests. When tests pass, deploy to production.",
			wantTier:    ComplexityBasic, // Score: 2 (triggers) + 1 (conditional) = 3 (basic)
			minScore:    3,
			maxScore:    7,
		},
		{
			name: "workflow with tools and file paths",
			content: `---
name: Tool Workflow
on: push
tools:
  github:
    toolsets: [default]
  myapi:
    url: https://example.com
  custom-tool:
    url: https://custom.example.com
---
Use multiple tools`,
			description: "Workflow with custom tools targeting src/main.go when needed",
			wantTier:    ComplexityIntermediate,
			minScore:    4, // 3 for tools (3 tools after removing github) + 1 for file path
			maxScore:    7,
		},
		{
			name: "multiple triggers with tools",
			content: `---
name: Complex Trigger Workflow
on: [push, pull_request, issues]
tools:
  api1:
    url: https://api1.example.com
---
Handle events`,
			description: "Multi-event workflow with API integration",
			wantTier:    ComplexityIntermediate,
			minScore:    4, // 3 for triggers + 2 for tool + 1 for integration
			maxScore:    7,
		},
		{
			name: "trigger with configuration and conditional",
			content: `---
name: Configured Trigger
on:
  push:
    branches: [main, develop]
  pull_request:
    types: [opened, synchronize]
---
Handle branch pushes`,
			description: "Run on specific branches. If main, deploy; if develop, test only. Otherwise skip.",
			wantTier:    ComplexityIntermediate,
			minScore:    4, // 3 for trigger config (multiple with config) + 1 for conditional
			maxScore:    7,
		},
		{
			name: "network restrictions with integration",
			content: `---
name: Network Workflow
on:
  push:
    branches: [main]
network:
  allowed:
    - api.example.com
tools:
  external-api:
    url: https://api.example.com
---
Call APIs`,
			description: "Integrate with external API services for data processing",
			wantTier:    ComplexityIntermediate,
			minScore:    4, // 1 for trigger config + 1 for network + 2 for tool + 1 for integration
			maxScore:    7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectWorkflowComplexity(tt.content, tt.description)
			require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
			assert.Equal(t, tt.wantTier, result.Tier, "Tier should match expected")
			assert.GreaterOrEqual(t, result.Score, tt.minScore, "Score should be at least minimum")
			assert.LessOrEqual(t, result.Score, tt.maxScore, "Score should be at most maximum")
		})
	}
}

func TestDetectWorkflowComplexity_Advanced(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		description string
		wantTier    ComplexityTier
		minScore    int
	}{
		{
			name: "many tools and triggers",
			content: `---
name: Complex Workflow
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize]
  schedule:
    - cron: '0 0 * * *'
tools:
  github:
    toolsets: [default]
  api1:
    url: https://api1.example.com
  api2:
    url: https://api2.example.com
  mcp-server:
    mode: remote
    url: https://mcp.example.com
---
Complex workflow`,
			description: "Advanced multi-stage workflow with integrations",
			wantTier:    ComplexityAdvanced,
			minScore:    8,
		},
		{
			name: "security compliance with performance and multiple tools",
			content: `---
name: Secure Workflow
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize]
tools:
  github:
    toolsets: [default]
  security-api:
    url: https://security.example.com
  performance-api:
    url: https://perf.example.com
  audit-api:
    url: https://audit.example.com
network:
  allowed:
    - api.example.com
    - secure.example.com
---
Secure workflow`,
			description: "High-performance workflow with security compliance, encryption requirements, and audit monitoring for distributed systems",
			wantTier:    ComplexityAdvanced,
			minScore:    8, // 3 for triggers + 4 for tools (4+ after removing github) + 2 for network + 2-3 for keywords
		},
		{
			name: "multi-stage with dependencies and tools",
			content: `---
name: Pipeline Workflow
on:
  push:
    branches: [main]
  pull_request:
    types: [opened]
tools:
  build-tool:
    url: https://build.example.com
  test-tool:
    url: https://test.example.com
  deploy-tool:
    url: https://deploy.example.com
jobs:
  build:
    runs-on: ubuntu-latest
  test:
    runs-on: ubuntu-latest
    needs: build
  deploy:
    runs-on: ubuntu-latest
    needs: test
---
Pipeline workflow`,
			description: "Multi-stage deployment pipeline with orchestration for distributed microservices",
			wantTier:    ComplexityAdvanced,
			minScore:    8, // 3 for triggers + 4 for tools (3 tools) + 4 for jobs + 2 for keywords (orchestration, distributed, microservice)
		},
		{
			name: "custom sandbox with security and integration",
			content: `---
name: Sandboxed Workflow
on:
  push:
    branches: [main]
  schedule:
    - cron: '0 0 * * *'
tools:
  secure-api:
    url: https://api.example.com
  integration-api:
    url: https://integration.example.com
sandbox:
  mounts:
    - source: /data
      target: /mnt/data
  env:
    CUSTOM_VAR: value
---
Sandboxed workflow`,
			description: "Custom sandbox environment with security compliance for external service integration and performance monitoring",
			wantTier:    ComplexityAdvanced,
			minScore:    8, // 4 for triggers (2 with sched) + 3 for tools + 3 for sandbox + 2 for keywords
		},
		{
			name: "network restrictions with many tools",
			content: `---
name: Network Restricted
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize]
network:
  allowed:
    - api.example.com
  blocked:
    - malicious.com
tools:
  api1:
    url: https://api1.example.com
  api2:
    url: https://api2.example.com
  api3:
    url: https://api3.example.com
  api4:
    url: https://api4.example.com
  api5:
    url: https://api5.example.com
---
Network restricted`,
			description: "Workflow with network security policies and distributed service integration for scalability",
			wantTier:    ComplexityAdvanced,
			minScore:    8, // 3 for triggers + 4 for tools (5+ tools) + 2 for network + 2 for keywords
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectWorkflowComplexity(tt.content, tt.description)
			require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
			assert.Equal(t, tt.wantTier, result.Tier, "Tier should match expected")
			assert.GreaterOrEqual(t, result.Score, tt.minScore, "Score should be at least minimum")
		})
	}
}

func TestDetectWorkflowComplexity_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		description string
		wantTier    ComplexityTier
	}{
		{
			name: "score at basic-intermediate boundary (3)",
			content: `---
name: Boundary Workflow
on:
  push:
    branches: [main]
---
Boundary test`,
			description: "Workflow at boundary with conditional logic if needed",
			wantTier:    ComplexityBasic,
		},
		{
			name: "score at intermediate-advanced boundary (7)",
			content: `---
name: Boundary Workflow
on:
  push:
    branches: [main]
  pull_request:
    types: [opened]
  schedule:
    - cron: '0 0 * * *'
tools:
  api1:
    url: https://api1.example.com
  api2:
    url: https://api2.example.com
---
Boundary test`,
			description: "Workflow at boundary with conditional logic",
			wantTier:    ComplexityIntermediate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectWorkflowComplexity(tt.content, tt.description)
			require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
			assert.Equal(t, tt.wantTier, result.Tier, "Tier should match expected")
		})
	}
}

func TestDetectWorkflowComplexity_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		description      string
		wantTier         ComplexityTier
		shouldHaveResult bool
	}{
		{
			name:             "empty content and description",
			content:          "",
			description:      "",
			wantTier:         ComplexityBasic,
			shouldHaveResult: true,
		},
		{
			name:             "invalid frontmatter",
			content:          "not valid yaml",
			description:      "Test workflow",
			wantTier:         ComplexityBasic,
			shouldHaveResult: true,
		},
		{
			name: "no on section",
			content: `---
name: No Trigger
---
No trigger`,
			description:      "No trigger defined",
			wantTier:         ComplexityBasic,
			shouldHaveResult: true,
		},
		{
			name: "empty tools section",
			content: `---
name: Empty Tools
on: push
tools: {}
---
Empty tools`,
			description:      "Empty tools",
			wantTier:         ComplexityBasic,
			shouldHaveResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectWorkflowComplexity(tt.content, tt.description)
			if tt.shouldHaveResult {
				require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
				assert.Equal(t, tt.wantTier, result.Tier, "Tier should match expected")
			}
		})
	}
}

func TestDetectWorkflowComplexity_DescriptionPatterns(t *testing.T) {
	tests := []struct {
		name        string
		description string
		minScore    int
		indicators  []string
	}{
		{
			name:        "performance keyword",
			description: "Optimize performance of the API",
			minScore:    1,
			indicators:  []string{"advanced requirement: performance"},
		},
		{
			name:        "security keyword",
			description: "Ensure security compliance",
			minScore:    1,
			indicators:  []string{"advanced requirement: security"},
		},
		{
			name:        "conditional logic",
			description: "If the test passes, deploy to production",
			minScore:    1,
			indicators:  []string{"conditional logic mentioned"},
		},
		{
			name:        "file path reference",
			description: "Update the file at src/main.go",
			minScore:    1,
			indicators:  []string{"specific file paths referenced"},
		},
		{
			name:        "integration requirement",
			description: "Integrate with external API service",
			minScore:    1,
			indicators:  []string{"integration requirements"},
		},
		{
			name:        "multiple patterns",
			description: "If security check passes, integrate with the external API at /api/v1/endpoint for performance optimization with monitoring",
			minScore:    3, // Actually 3: 1 each for security/performance, conditional, integration (file path not matching)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use empty content to test description analysis only
			result := DetectWorkflowComplexity("", tt.description)
			require.NotNil(t, result, "DetectWorkflowComplexity should return a result")
			assert.GreaterOrEqual(t, result.Score, tt.minScore, "Score should be at least minimum")

			if len(tt.indicators) > 0 {
				for _, indicator := range tt.indicators {
					assert.Contains(t, result.Indicators, indicator, "Should contain expected indicator")
				}
			}
		})
	}
}

func TestIsStandardEvent(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  bool
	}{
		{
			name:  "push event",
			event: "push",
			want:  true,
		},
		{
			name:  "pull_request event",
			event: "pull_request",
			want:  true,
		},
		{
			name:  "workflow_dispatch event",
			event: "workflow_dispatch",
			want:  true,
		},
		{
			name:  "custom event",
			event: "custom_trigger",
			want:  false,
		},
		{
			name:  "case insensitive",
			event: "PUSH",
			want:  true,
		},
		{
			name:  "with whitespace",
			event: "  push  ",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isStandardEvent(tt.event), "isStandardEvent should return correct value")
		})
	}
}

func TestDetermineTier(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  ComplexityTier
	}{
		{
			name:  "score 0 is basic",
			score: 0,
			want:  ComplexityBasic,
		},
		{
			name:  "score 3 is basic (boundary)",
			score: 3,
			want:  ComplexityBasic,
		},
		{
			name:  "score 4 is intermediate",
			score: 4,
			want:  ComplexityIntermediate,
		},
		{
			name:  "score 7 is intermediate (boundary)",
			score: 7,
			want:  ComplexityIntermediate,
		},
		{
			name:  "score 8 is advanced",
			score: 8,
			want:  ComplexityAdvanced,
		},
		{
			name:  "score 15 is advanced",
			score: 15,
			want:  ComplexityAdvanced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, determineTier(tt.score), "determineTier should return correct tier")
		})
	}
}
