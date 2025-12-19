package workflow

import (
	"strings"
	"testing"
)

// TestValidateToolchains tests the toolchain validation function
func TestValidateToolchains(t *testing.T) {
	tests := []struct {
		name       string
		toolchains []string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid go toolchain",
			toolchains: []string{"go"},
			wantErr:    false,
		},
		{
			name:       "valid node toolchain",
			toolchains: []string{"node"},
			wantErr:    false,
		},
		{
			name:       "valid python toolchain",
			toolchains: []string{"python"},
			wantErr:    false,
		},
		{
			name:       "valid multiple toolchains",
			toolchains: []string{"go", "node", "python"},
			wantErr:    false,
		},
		{
			name:       "valid all toolchains",
			toolchains: []string{"go", "node", "python", "ruby", "rust", "java", "dotnet"},
			wantErr:    false,
		},
		{
			name:       "empty toolchains list",
			toolchains: []string{},
			wantErr:    false,
		},
		{
			name:       "invalid toolchain",
			toolchains: []string{"invalid-toolchain"},
			wantErr:    true,
			errMsg:     "unsupported toolchain",
		},
		{
			name:       "mixed valid and invalid toolchains",
			toolchains: []string{"go", "invalid"},
			wantErr:    true,
			errMsg:     "unsupported toolchain at index 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolchains(tt.toolchains)

			if tt.wantErr && err == nil {
				t.Errorf("validateToolchains() expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateToolchains() unexpected error: %v", err)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateToolchains() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestGetToolchainMounts tests that toolchain mounts are generated correctly
func TestGetToolchainMounts(t *testing.T) {
	tests := []struct {
		name         string
		toolchains   []string
		expectMounts []string
	}{
		{
			name:       "go toolchain",
			toolchains: []string{"go"},
			expectMounts: []string{
				"/usr/local/go/bin:/usr/local/go/bin:ro",
			},
		},
		{
			name:       "node toolchain",
			toolchains: []string{"node"},
			expectMounts: []string{
				"/usr/local/bin/node:/usr/local/bin/node:ro",
				"/usr/local/bin/npm:/usr/local/bin/npm:ro",
				"/usr/local/bin/npx:/usr/local/bin/npx:ro",
			},
		},
		{
			name:       "multiple toolchains",
			toolchains: []string{"go", "python"},
			expectMounts: []string{
				"/usr/local/go/bin:/usr/local/go/bin:ro",
				"/usr/bin/python3:/usr/bin/python3:ro",
				"/usr/bin/pip3:/usr/bin/pip3:ro",
			},
		},
		{
			name:         "empty toolchains",
			toolchains:   []string{},
			expectMounts: []string{},
		},
		{
			name:         "unsupported toolchain (ignored)",
			toolchains:   []string{"unsupported"},
			expectMounts: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts := GetToolchainMounts(tt.toolchains)

			if len(mounts) != len(tt.expectMounts) {
				t.Errorf("GetToolchainMounts() got %d mounts, want %d mounts", len(mounts), len(tt.expectMounts))
				t.Logf("Got mounts: %v", mounts)
				t.Logf("Expected mounts: %v", tt.expectMounts)
				return
			}

			for _, expectedMount := range tt.expectMounts {
				found := false
				for _, mount := range mounts {
					if mount == expectedMount {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetToolchainMounts() missing expected mount: %s", expectedMount)
				}
			}
		})
	}
}

// TestGetToolchainPATHAdditions tests that PATH additions are generated correctly
func TestGetToolchainPATHAdditions(t *testing.T) {
	tests := []struct {
		name        string
		toolchains  []string
		expectPaths []string
	}{
		{
			name:        "go toolchain",
			toolchains:  []string{"go"},
			expectPaths: []string{"/usr/local/go/bin"},
		},
		{
			name:        "node toolchain",
			toolchains:  []string{"node"},
			expectPaths: []string{"/usr/local/bin"},
		},
		{
			name:        "multiple toolchains with overlapping paths",
			toolchains:  []string{"node", "python"},
			expectPaths: []string{"/usr/local/bin", "/usr/bin"},
		},
		{
			name:        "empty toolchains",
			toolchains:  []string{},
			expectPaths: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := GetToolchainPATHAdditions(tt.toolchains)

			if len(paths) != len(tt.expectPaths) {
				t.Errorf("GetToolchainPATHAdditions() got %d paths, want %d paths", len(paths), len(tt.expectPaths))
				t.Logf("Got paths: %v", paths)
				t.Logf("Expected paths: %v", tt.expectPaths)
				return
			}

			for _, expectedPath := range tt.expectPaths {
				found := false
				for _, path := range paths {
					if path == expectedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetToolchainPATHAdditions() missing expected path: %s", expectedPath)
				}
			}
		})
	}
}

// TestSandboxConfigWithToolchains tests that sandbox configuration with toolchains is validated
func TestSandboxConfigWithToolchains(t *testing.T) {
	tests := []struct {
		name    string
		data    *WorkflowData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid toolchains in agent config",
			data: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:         "awf",
						Toolchains: []string{"go", "node"},
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no toolchains in agent config",
			data: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid toolchain in agent config",
			data: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:         "awf",
						Toolchains: []string{"go", "invalid-toolchain"},
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			wantErr: true,
			errMsg:  "unsupported toolchain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSandboxConfig(tt.data)

			if tt.wantErr && err == nil {
				t.Errorf("validateSandboxConfig() expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateSandboxConfig() unexpected error: %v", err)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateSandboxConfig() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestCopilotEngineWithToolchains tests that toolchains are included in AWF command
func TestCopilotEngineWithToolchains(t *testing.T) {
	t.Run("toolchains are included in AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:         "awf",
					Toolchains: []string{"go"},
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that go toolchain mount is included
		if !strings.Contains(stepContent, "--mount /usr/local/go/bin:/usr/local/go/bin:ro") {
			t.Error("Expected command to contain go toolchain mount '--mount /usr/local/go/bin:/usr/local/go/bin:ro'")
		}

		// Check that PATH is modified
		if !strings.Contains(stepContent, "PATH: /usr/local/go/bin:$PATH") {
			t.Error("Expected step to contain PATH modification 'PATH: /usr/local/go/bin:$PATH'")
		}
	})

	t.Run("multiple toolchains are included", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:         "awf",
					Toolchains: []string{"go", "python"},
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that go toolchain mount is included
		if !strings.Contains(stepContent, "--mount /usr/local/go/bin:/usr/local/go/bin:ro") {
			t.Error("Expected command to contain go toolchain mount")
		}

		// Check that python toolchain mount is included
		if !strings.Contains(stepContent, "--mount /usr/bin/python3:/usr/bin/python3:ro") {
			t.Error("Expected command to contain python toolchain mount")
		}
	})

	t.Run("no toolchains when not specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that go toolchain mount is NOT included
		if strings.Contains(stepContent, "--mount /usr/local/go/bin:/usr/local/go/bin:ro") {
			t.Error("Did not expect go toolchain mount in output when not configured")
		}
	})
}
