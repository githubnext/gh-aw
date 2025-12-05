package workflow

import (
	"strings"
	"testing"
)

// TestGenerateDefaultMountsFromBashTools tests the generation of default mounts from bash tools
func TestGenerateDefaultMountsFromBashTools(t *testing.T) {
	tests := []struct {
		name           string
		bashTools      []string
		explicitMounts []string
		wantMounts     []string
		wantCount      int
	}{
		{
			name:       "bash with wildcard * generates entire bin folder mount",
			bashTools:  []string{"*"},
			wantMounts: []string{"/usr/bin:/usr/bin:ro"},
			wantCount:  1,
		},
		{
			name:       "bash with :* wildcard generates entire bin folder mount",
			bashTools:  []string{":*"},
			wantMounts: []string{"/usr/bin:/usr/bin:ro"},
			wantCount:  1,
		},
		{
			name:      "bash with tool:* pattern generates tool mount",
			bashTools: []string{"git:*"},
			wantMounts: []string{
				"/usr/bin/git:/usr/bin/git:ro",
			},
			wantCount: 1,
		},
		{
			name:      "bash with tool * pattern generates tool mount",
			bashTools: []string{"git *"},
			wantMounts: []string{
				"/usr/bin/git:/usr/bin/git:ro",
			},
			wantCount: 1,
		},
		{
			name:      "bash with multiple tool patterns",
			bashTools: []string{"git:*", "cat:*", "grep:*"},
			wantMounts: []string{
				"/usr/bin/cat:/usr/bin/cat:ro",
				"/usr/bin/git:/usr/bin/git:ro",
				"/usr/bin/grep:/usr/bin/grep:ro",
			},
			wantCount: 3,
		},
		{
			name:      "bash with tool subcommand patterns",
			bashTools: []string{"git diff:*", "git log:*", "cat:*"},
			wantMounts: []string{
				"/usr/bin/cat:/usr/bin/cat:ro",
				"/usr/bin/git:/usr/bin/git:ro",
			},
			wantCount: 2,
		},
		{
			name:      "bash with simple tool names",
			bashTools: []string{"git", "cat", "grep"},
			wantMounts: []string{
				"/usr/bin/cat:/usr/bin/cat:ro",
				"/usr/bin/git:/usr/bin/git:ro",
				"/usr/bin/grep:/usr/bin/grep:ro",
			},
			wantCount: 3,
		},
		{
			name:      "bash with duplicate tools deduplicates mounts",
			bashTools: []string{"git:*", "git diff:*", "git log:*"},
			wantMounts: []string{
				"/usr/bin/git:/usr/bin/git:ro",
			},
			wantCount: 1,
		},
		{
			name:       "empty bash tools returns nil",
			bashTools:  []string{},
			wantMounts: nil,
			wantCount:  0,
		},
		{
			name:           "explicit mounts override defaults",
			bashTools:      []string{"git:*", "cat:*"},
			explicitMounts: []string{"/custom/path:/custom/path:ro"},
			wantMounts:     nil,
			wantCount:      0,
		},
		{
			name:      "bash with wildcard and other tools only generates bin folder mount",
			bashTools: []string{"*", "git:*", "cat:*"},
			wantMounts: []string{
				"/usr/bin:/usr/bin:ro",
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create workflow data
			workflowData := &WorkflowData{
				Name: "test-workflow",
				ParsedTools: &ToolsConfig{
					Bash: &BashToolConfig{
						AllowedCommands: tt.bashTools,
					},
				},
			}

			// Add explicit mounts if specified
			if len(tt.explicitMounts) > 0 {
				workflowData.SandboxConfig = &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Mounts: tt.explicitMounts,
					},
				}
			}

			// Generate mounts
			mounts := generateDefaultMountsFromBashTools(workflowData)

			// Check count
			if len(mounts) != tt.wantCount {
				t.Errorf("generateDefaultMountsFromBashTools() returned %d mounts, want %d", len(mounts), tt.wantCount)
			}

			// Check that all expected mounts are present
			if tt.wantMounts != nil {
				for _, wantMount := range tt.wantMounts {
					found := false
					for _, mount := range mounts {
						if mount == wantMount {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("generateDefaultMountsFromBashTools() missing expected mount: %s", wantMount)
					}
				}
			}

			// Check that mounts are sorted
			if len(mounts) > 1 {
				for i := 0; i < len(mounts)-1; i++ {
					if mounts[i] >= mounts[i+1] {
						t.Errorf("generateDefaultMountsFromBashTools() mounts not sorted: %s >= %s", mounts[i], mounts[i+1])
					}
				}
			}
		})
	}
}

// TestExtractToolNameFromPattern tests the extraction of tool names from bash patterns
func TestExtractToolNameFromPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		wantTool string
	}{
		{"git:*", "git"},
		{"git *", "git"},
		{"git", "git"},
		{"git diff:*", "git"},
		{"cat:*", "cat"},
		{"echo *", "echo"},
		{"*", ""},
		{":*", ""},
		{"  git:*  ", "git"},
		{"  git *  ", "git"},
		{"make", "make"},
		{"npm:*", "npm"},
		{"npm run:*", "npm"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := extractToolNameFromPattern(tt.pattern)
			if got != tt.wantTool {
				t.Errorf("extractToolNameFromPattern(%q) = %q, want %q", tt.pattern, got, tt.wantTool)
			}
		})
	}
}

// TestCopilotEngineWithDefaultMounts tests that default mounts from bash tools are included in AWF command
func TestCopilotEngineWithDefaultMounts(t *testing.T) {
	t.Run("default mounts from bash:* wildcard", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			ParsedTools: &ToolsConfig{
				Bash: &BashToolConfig{
					AllowedCommands: []string{"*"},
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

		// Check that bin folder mount is included
		if !strings.Contains(stepContent, "--mount /usr/bin:/usr/bin:ro") {
			t.Error("Expected command to contain bin folder mount '--mount /usr/bin:/usr/bin:ro'")
		}
	})

	t.Run("default mounts from bash tool patterns", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			ParsedTools: &ToolsConfig{
				Bash: &BashToolConfig{
					AllowedCommands: []string{"git:*", "cat:*", "grep:*"},
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

		// Check that tool mounts are included
		if !strings.Contains(stepContent, "--mount /usr/bin/git:/usr/bin/git:ro") {
			t.Error("Expected command to contain git mount '--mount /usr/bin/git:/usr/bin/git:ro'")
		}

		if !strings.Contains(stepContent, "--mount /usr/bin/cat:/usr/bin/cat:ro") {
			t.Error("Expected command to contain cat mount '--mount /usr/bin/cat:/usr/bin/cat:ro'")
		}

		if !strings.Contains(stepContent, "--mount /usr/bin/grep:/usr/bin/grep:ro") {
			t.Error("Expected command to contain grep mount '--mount /usr/bin/grep:/usr/bin/grep:ro'")
		}
	})

	t.Run("explicit mounts override default mounts", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			ParsedTools: &ToolsConfig{
				Bash: &BashToolConfig{
					AllowedCommands: []string{"git:*", "cat:*"},
				},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
					Mounts: []string{
						"/custom/path:/custom/path:ro",
					},
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

		// Check that custom mount is included
		if !strings.Contains(stepContent, "--mount /custom/path:/custom/path:ro") {
			t.Error("Expected command to contain custom mount '--mount /custom/path:/custom/path:ro'")
		}

		// Check that default mounts from bash tools are NOT included
		if strings.Contains(stepContent, "--mount /usr/bin/git:/usr/bin/git:ro") {
			t.Error("Did not expect git mount when explicit mounts are configured")
		}

		if strings.Contains(stepContent, "--mount /usr/bin/cat:/usr/bin/cat:ro") {
			t.Error("Did not expect cat mount when explicit mounts are configured")
		}
	})

	t.Run("no default mounts when bash tools not configured", func(t *testing.T) {
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

		// Verify standard mounts are still present
		if !strings.Contains(stepContent, "--mount /tmp:/tmp:rw") {
			t.Error("Expected command to contain standard mount '--mount /tmp:/tmp:rw'")
		}

		// Should not have extra tool mounts
		// Only the standard gh, yq, date should be present (from the engine itself)
	})
}
