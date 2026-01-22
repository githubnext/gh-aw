package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRuntimeMounts(t *testing.T) {
	tests := []struct {
		name              string
		requirements      []RuntimeRequirement
		expectedContains  []string
		expectedExcludes  []string
		expectedMinCount  int
	}{
		{
			name: "node runtime includes toolcache and npm cache",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("node"),
					Version: "20",
				},
			},
			expectedContains: []string{
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
			},
			expectedMinCount: 2,
		},
		{
			name: "python runtime includes toolcache and pip cache",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("python"),
					Version: "3.11",
				},
			},
			expectedContains: []string{
				"/opt/hostedtoolcache/Python:/opt/hostedtoolcache/Python:ro",
				"/home/runner/.cache/pip:/home/runner/.cache/pip:rw",
			},
			expectedMinCount: 2,
		},
		{
			name: "go runtime includes toolcache, GOPATH, and build cache",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("go"),
					Version: "1.22",
				},
			},
			expectedContains: []string{
				"/opt/hostedtoolcache/go:/opt/hostedtoolcache/go:ro",
				"/home/runner/go:/home/runner/go:rw",
				"/home/runner/.cache/go-build:/home/runner/.cache/go-build:rw",
			},
			expectedMinCount: 3,
		},
		{
			name: "multiple runtimes combine mounts",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("node"),
					Version: "20",
				},
				{
					Runtime: findRuntimeByID("python"),
					Version: "3.11",
				},
			},
			expectedContains: []string{
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
				"/opt/hostedtoolcache/Python:/opt/hostedtoolcache/Python:ro",
				"/home/runner/.cache/pip:/home/runner/.cache/pip:rw",
			},
			expectedMinCount: 4,
		},
		{
			name: "uv runtime includes cache",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("uv"),
					Version: "",
				},
			},
			expectedContains: []string{
				"/home/runner/.cache/uv:/home/runner/.cache/uv:rw",
			},
			expectedMinCount: 1,
		},
		{
			name: "java runtime includes toolcache, maven, and gradle",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("java"),
					Version: "17",
				},
			},
			expectedContains: []string{
				"/opt/hostedtoolcache/Java_Temurin-Hotspot_jdk:/opt/hostedtoolcache/Java_Temurin-Hotspot_jdk:ro",
				"/home/runner/.m2:/home/runner/.m2:rw",
				"/home/runner/.gradle:/home/runner/.gradle:rw",
			},
			expectedMinCount: 3,
		},
		{
			name:             "no requirements returns empty list",
			requirements:     []RuntimeRequirement{},
			expectedContains: []string{},
			expectedMinCount: 0,
		},
		{
			name: "deduplicates identical mounts from multiple runtimes",
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("python"),
					Version: "3.11",
				},
				{
					Runtime: findRuntimeByID("uv"),
					Version: "",
				},
			},
			// Both python and uv might share cache directories
			// Ensure no duplicates in final list
			expectedMinCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts := GetRuntimeMounts(tt.requirements)

			// Check minimum count
			assert.GreaterOrEqual(t, len(mounts), tt.expectedMinCount,
				"Should have at least %d mounts, got %d", tt.expectedMinCount, len(mounts))

			// Check that expected mounts are present
			for _, expectedMount := range tt.expectedContains {
				assert.Contains(t, mounts, expectedMount,
					"Expected mount %q to be present", expectedMount)
			}

			// Check that excluded mounts are not present
			for _, excludedMount := range tt.expectedExcludes {
				assert.NotContains(t, mounts, excludedMount,
					"Expected mount %q to be excluded", excludedMount)
			}

			// Verify mounts are sorted
			if len(mounts) > 1 {
				for i := 0; i < len(mounts)-1; i++ {
					assert.LessOrEqual(t, mounts[i], mounts[i+1],
						"Mounts should be sorted alphabetically")
				}
			}

			// Verify no duplicate mounts
			seen := make(map[string]bool)
			for _, mount := range mounts {
				assert.False(t, seen[mount], "Found duplicate mount: %s", mount)
				seen[mount] = true
			}

			// Verify mount format (source:dest:mode)
			for _, mount := range mounts {
				parts := strings.Split(mount, ":")
				assert.Equal(t, 3, len(parts),
					"Mount %q should have 3 parts (source:dest:mode)", mount)
				assert.NotEmpty(t, parts[0], "Source path should not be empty in mount %q", mount)
				assert.NotEmpty(t, parts[1], "Destination path should not be empty in mount %q", mount)
				assert.Contains(t, []string{"ro", "rw"}, parts[2],
					"Mode in mount %q should be 'ro' or 'rw'", mount)
			}
		})
	}
}

func TestGetRuntimeMountDefinition(t *testing.T) {
	tests := []struct {
		name             string
		runtimeID        string
		expectNil        bool
		expectedMinMounts int
	}{
		{
			name:             "node has mount definitions",
			runtimeID:        "node",
			expectNil:        false,
			expectedMinMounts: 2,
		},
		{
			name:             "python has mount definitions",
			runtimeID:        "python",
			expectNil:        false,
			expectedMinMounts: 2,
		},
		{
			name:             "go has mount definitions",
			runtimeID:        "go",
			expectNil:        false,
			expectedMinMounts: 3,
		},
		{
			name:             "unknown runtime returns nil",
			runtimeID:        "unknown-runtime",
			expectNil:        true,
			expectedMinMounts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := getRuntimeMountDefinition(tt.runtimeID)

			if tt.expectNil {
				assert.Nil(t, def, "Expected nil for runtime %s", tt.runtimeID)
			} else {
				require.NotNil(t, def, "Expected non-nil definition for runtime %s", tt.runtimeID)
				assert.Equal(t, tt.runtimeID, def.RuntimeID,
					"RuntimeID should match")
				assert.GreaterOrEqual(t, len(def.Mounts), tt.expectedMinMounts,
					"Should have at least %d mounts", tt.expectedMinMounts)
			}
		})
	}
}

func TestContributeRuntimeMounts(t *testing.T) {
	tests := []struct {
		name             string
		agentConfig      *AgentSandboxConfig
		requirements     []RuntimeRequirement
		expectedMinMounts int
		expectNoChange   bool
	}{
		{
			name: "adds runtime mounts to empty agent config",
			agentConfig: &AgentSandboxConfig{
				ID: "awf",
			},
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("node"),
					Version: "20",
				},
			},
			expectedMinMounts: 2,
			expectNoChange:   false,
		},
		{
			name: "adds runtime mounts to existing user mounts",
			agentConfig: &AgentSandboxConfig{
				ID: "awf",
				Mounts: []string{
					"/custom/path:/custom/path:ro",
				},
			},
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("python"),
					Version: "3.11",
				},
			},
			expectedMinMounts: 3, // 1 custom + 2 runtime
			expectNoChange:   false,
		},
		{
			name:             "nil agent config does not panic",
			agentConfig:      nil,
			requirements:     []RuntimeRequirement{},
			expectedMinMounts: 0,
			expectNoChange:   true,
		},
		{
			name: "no requirements does not add mounts",
			agentConfig: &AgentSandboxConfig{
				ID: "awf",
			},
			requirements:     []RuntimeRequirement{},
			expectedMinMounts: 0,
			expectNoChange:   true,
		},
		{
			name: "multiple runtimes add combined mounts",
			agentConfig: &AgentSandboxConfig{
				ID: "awf",
			},
			requirements: []RuntimeRequirement{
				{
					Runtime: findRuntimeByID("node"),
					Version: "20",
				},
				{
					Runtime: findRuntimeByID("go"),
					Version: "1.22",
				},
			},
			expectedMinMounts: 5, // 2 (node) + 3 (go)
			expectNoChange:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMountCount := 0
			if tt.agentConfig != nil {
				originalMountCount = len(tt.agentConfig.Mounts)
			}

			ContributeRuntimeMounts(tt.agentConfig, tt.requirements)

			if tt.expectNoChange {
				if tt.agentConfig != nil {
					assert.Equal(t, originalMountCount, len(tt.agentConfig.Mounts),
						"Mount count should not change")
				}
			} else {
				require.NotNil(t, tt.agentConfig, "Agent config should not be nil")
				assert.GreaterOrEqual(t, len(tt.agentConfig.Mounts), tt.expectedMinMounts,
					"Should have at least %d mounts after contribution", tt.expectedMinMounts)
			}
		})
	}
}

func TestGetRuntimeMountsForWorkflow(t *testing.T) {
	tests := []struct {
		name             string
		workflowData     *WorkflowData
		expectedMinMounts int
	}{
		{
			name: "workflow with node in custom steps",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				CustomSteps: `      - name: Install deps
        run: npm install`,
			},
			expectedMinMounts: 2, // node mounts
		},
		{
			name: "workflow with python in custom steps",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				CustomSteps: `      - name: Run script
        run: python script.py`,
			},
			expectedMinMounts: 2, // python mounts
		},
		{
			name:             "nil workflow returns empty list",
			workflowData:     nil,
			expectedMinMounts: 0,
		},
		{
			name: "workflow with no runtime commands",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				CustomSteps: `      - name: Echo
        run: echo "hello"`,
			},
			expectedMinMounts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts := GetRuntimeMountsForWorkflow(tt.workflowData)

			assert.GreaterOrEqual(t, len(mounts), tt.expectedMinMounts,
				"Should have at least %d mounts", tt.expectedMinMounts)
		})
	}
}
