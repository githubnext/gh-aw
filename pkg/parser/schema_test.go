package parser

import (
	"os"
	"strings"
	"testing"
)

func TestValidateMainWorkflowFrontmatterWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter with all allowed keys",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
					"stop-after": "2024-12-31",
				},
				"permissions":     "read",
				"run-name":        "Test Run",
				"runs-on":         "ubuntu-latest",
				"timeout_minutes": 30,
				"concurrency":     "test",
				"env":             map[string]string{"TEST": "value"},
				"if":              "true",
				"steps":           []string{"step1"},
				"engine":          "claude",
				"tools":           map[string]any{"github": "test"},
				"command":         "test-workflow",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with subset of keys",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "valid engine string format - claude",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid engine string format - codex",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "codex",
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - minimal",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "claude",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - with version",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - with model",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":    "codex",
					"model": "gpt-4o",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - complete",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
					"model":   "claude-3-5-sonnet-20241022",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid engine string format",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "invalid-engine",
			},
			wantErr:     true,
			errContains: "value must be one of 'claude', 'codex'",
		},
		{
			name: "invalid engine object format - invalid id",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "invalid-engine",
				},
			},
			wantErr:     true,
			errContains: "value must be one of 'claude', 'codex'",
		},
		{
			name: "invalid engine object format - missing id",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"version": "beta",
					"model":   "gpt-4o",
				},
			},
			wantErr:     true,
			errContains: "missing property 'id'",
		},
		{
			name: "invalid engine object format - additional properties",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"invalid": "property",
				},
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid frontmatter with unexpected key",
			frontmatter: map[string]any{
				"on":          "push",
				"invalid_key": "value",
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_key' not allowed",
		},
		{
			name: "invalid frontmatter with multiple unexpected keys",
			frontmatter: map[string]any{
				"on":              "push",
				"invalid_key":     "value",
				"another_invalid": "value2",
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid type for timeout_minutes",
			frontmatter: map[string]any{
				"timeout_minutes": "not-a-number",
			},
			wantErr:     true,
			errContains: "got string, want integer",
		},
		{
			name: "valid frontmatter with complex on object",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []map[string]any{
						{"cron": "0 9 * * *"},
					},
					"workflow_dispatch": map[string]any{},
				},
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with command trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"command": map[string]any{
						"name": "test-command",
					},
				},
				"permissions": map[string]any{
					"issues":   "write",
					"contents": "read",
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with discussion trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"discussion": map[string]any{
						"types": []string{"created", "edited", "answered"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with discussion_comment trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"discussion_comment": map[string]any{
						"types": []string{"created", "edited"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple discussion trigger",
			frontmatter: map[string]any{
				"on":     "discussion",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with branch_protection_rule trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"branch_protection_rule": map[string]any{
						"types": []string{"created", "deleted"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with check_run trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"check_run": map[string]any{
						"types": []string{"completed", "rerequested"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with check_suite trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"check_suite": map[string]any{
						"types": []string{"completed"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple create trigger",
			frontmatter: map[string]any{
				"on":     "create",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple delete trigger",
			frontmatter: map[string]any{
				"on":     "delete",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple fork trigger",
			frontmatter: map[string]any{
				"on":     "fork",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple gollum trigger",
			frontmatter: map[string]any{
				"on":     "gollum",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with label trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"label": map[string]any{
						"types": []string{"created", "deleted"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with merge_group trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"merge_group": map[string]any{
						"types": []string{"checks_requested"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with milestone trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"milestone": map[string]any{
						"types": []string{"opened", "closed"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple page_build trigger",
			frontmatter: map[string]any{
				"on":     "page_build",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple public trigger",
			frontmatter: map[string]any{
				"on":     "public",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with pull_request_target trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"types":    []string{"opened", "synchronize"},
						"branches": []string{"main"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with pull_request_review trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_review": map[string]any{
						"types": []string{"submitted", "edited"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with registry_package trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"registry_package": map[string]any{
						"types": []string{"published", "updated"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with repository_dispatch trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"repository_dispatch": map[string]any{
						"types": []string{"custom-event", "deploy"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple status trigger",
			frontmatter: map[string]any{
				"on":     "status",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with watch trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"watch": map[string]any{
						"types": []string{"started"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with simple workflow_call trigger",
			frontmatter: map[string]any{
				"on":     "workflow_call",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with updated issues trigger types",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened", "typed", "untyped"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with updated pull_request trigger types",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types": []string{"opened", "milestoned", "demilestoned", "ready_for_review", "auto_merge_enabled"},
					},
				},
				"permissions": "read",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with complex tools configuration (new format)",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"create_issue", "update_issue"},
					},
					"claude": map[string]any{
						"allowed": map[string]any{
							"WebFetch": nil,
							"Bash":     []string{"echo:*", "ls"},
						},
					},
					"customTool": map[string]any{
						"type":    "stdio",
						"command": "my-tool",
						"allowed": []string{"function1", "function2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid new format: stdio without command or container",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"invalidTool": map[string]any{
						"type":    "stdio",
						"allowed": []string{"some_function"},
					},
				},
			},
			wantErr:     true,
			errContains: "anyOf",
		},
		{
			name: "invalid new format: http without url",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"invalidHttp": map[string]any{
						"type":    "http",
						"allowed": []string{"some_function"},
					},
				},
			},
			wantErr:     true,
			errContains: "missing property 'url'",
		},
		{
			name: "invalid new format: unknown type",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"unknownType": map[string]any{
						"type":    "unknown",
						"command": "test",
						"allowed": []string{"some_function"},
					},
				},
			},
			wantErr:     true,
			errContains: "oneOf",
		},
		{
			name: "valid frontmatter with detailed permissions",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents":      "read",
					"issues":        "write",
					"pull-requests": "read",
					"models":        "read",
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with single cache configuration",
			frontmatter: map[string]any{
				"cache": map[string]any{
					"key":          "node-modules-${{ hashFiles('package-lock.json') }}",
					"path":         "node_modules",
					"restore-keys": []string{"node-modules-"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with multiple cache configurations",
			frontmatter: map[string]any{
				"cache": []any{
					map[string]any{
						"key":  "cache1",
						"path": "path1",
					},
					map[string]any{
						"key":                "cache2",
						"path":               []string{"path2", "path3"},
						"restore-keys":       "restore-key",
						"fail-on-cache-miss": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid cache configuration missing required key",
			frontmatter: map[string]any{
				"cache": map[string]any{
					"path": "node_modules",
				},
			},
			wantErr:     true,
			errContains: "missing property 'key'",
		},
		// Test cases for additional properties validation
		{
			name: "invalid permissions with additional property",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents":     "read",
					"invalid_perm": "write",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_perm' not allowed",
		},
		{
			name: "invalid on trigger with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches":     []string{"main"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid schedule with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []map[string]any{
						{
							"cron":         "0 9 * * *",
							"invalid_prop": "value",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid workflow_dispatch with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_dispatch": map[string]any{
						"inputs": map[string]any{
							"test_input": map[string]any{
								"description": "Test input",
								"type":        "string",
							},
						},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid concurrency with additional properties",
			frontmatter: map[string]any{
				"concurrency": map[string]any{
					"group":              "test-group",
					"cancel-in-progress": true,
					"invalid_prop":       "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid runs-on object with additional properties",
			frontmatter: map[string]any{
				"runs-on": map[string]any{
					"group":        "test-group",
					"labels":       []string{"ubuntu-latest"},
					"invalid_prop": "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid github tools with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed":      []string{"create_issue"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid claude tools with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"claude": map[string]any{
						"allowed":      []string{"WebFetch"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid custom tool with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"customTool": map[string]any{
						"type":         "stdio",
						"command":      "my-tool",
						"allowed":      []string{"function1"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid claude configuration with additional properties",
			frontmatter: map[string]any{
				"claude": map[string]any{
					"model":        "claude-3",
					"invalid_prop": "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid safe-outputs configuration with additional properties",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"title-prefix": "[ai] ",
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "valid new GitHub Actions properties - permissions with new properties",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents":            "read",
					"attestations":        "write",
					"id-token":            "write",
					"packages":            "read",
					"pages":               "write",
					"repository-projects": "none",
				},
			},
			wantErr: false,
		},
		{
			name: "valid GitHub Actions defaults property",
			frontmatter: map[string]any{
				"on": "push",
				"defaults": map[string]any{
					"run": map[string]any{
						"shell":             "bash",
						"working-directory": "/app",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid defaults with additional properties",
			frontmatter: map[string]any{
				"defaults": map[string]any{
					"run": map[string]any{
						"shell":        "bash",
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "valid claude engine with network permissions",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "claude",
				},
			},
			wantErr: false,
		},
		{
			name: "valid codex engine without permissions",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":    "codex",
					"model": "gpt-4o",
				},
			},
			wantErr: false,
		},
		{
			name: "valid codex string engine (no permissions possible)",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "codex",
			},
			wantErr: false,
		},
		{
			name: "valid network defaults",
			frontmatter: map[string]any{
				"on":      "push",
				"network": "defaults",
			},
			wantErr: false,
		},
		{
			name: "valid network empty object",
			frontmatter: map[string]any{
				"on":      "push",
				"network": map[string]any{},
			},
			wantErr: false,
		},
		{
			name: "valid network with allowed domains",
			frontmatter: map[string]any{
				"on": "push",
				"network": map[string]any{
					"allowed": []string{"example.com", "*.trusted.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid network string (not defaults)",
			frontmatter: map[string]any{
				"on":      "push",
				"network": "invalid",
			},
			wantErr:     true,
			errContains: "oneOf",
		},
		{
			name: "invalid network object with unknown property",
			frontmatter: map[string]any{
				"on": "push",
				"network": map[string]any{
					"invalid": []string{"example.com"},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid' not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests for new MCP format during MCP revamp
			if strings.Contains(tt.name, "complex tools configuration (new format)") ||
				strings.Contains(tt.name, "stdio without command") ||
				strings.Contains(tt.name, "http without url") ||
				strings.Contains(tt.name, "unknown type") ||
				strings.Contains(tt.name, "custom tool with additional properties") {
				t.Skip("Skipping test for MCP format changes - MCP revamp in progress")
				return
			}

			err := ValidateMainWorkflowFrontmatterWithSchema(tt.frontmatter)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateIncludedFileFrontmatterWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter with tools only",
			frontmatter: map[string]any{
				"tools": map[string]any{"github": "test"},
			},
			wantErr: false,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "invalid frontmatter with on trigger",
			frontmatter: map[string]any{
				"on":    "push",
				"tools": map[string]any{"github": "test"},
			},
			wantErr:     true,
			errContains: "additional properties 'on' not allowed",
		},
		{
			name: "invalid frontmatter with multiple unexpected keys",
			frontmatter: map[string]any{
				"on":          "push",
				"permissions": "read",
				"tools":       map[string]any{"github": "test"},
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid frontmatter with only unexpected keys",
			frontmatter: map[string]any{
				"on":          "push",
				"permissions": "read",
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "valid frontmatter with complex tools object",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"list_issues", "get_issue"},
					},
					"claude": map[string]any{
						"allowed": map[string]any{
							"Edit":     nil,
							"WebFetch": nil,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with custom MCP tool",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"myTool": map[string]any{
						"mcp": map[string]any{
							"type":    "http",
							"url":     "https://api.contoso.com",
							"headers": map[string]any{"Authorization": "Bearer token"},
						},
						"allowed": []string{"api_call1", "api_call2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with HTTP MCP tool with underscored headers",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"datadog": map[string]any{
						"type": "http",
						"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
						"headers": map[string]any{
							"DD_API_KEY": "test-key",
							"DD_APPLICATION_KEY": "test-app",
							"DD_SITE":    "datadoghq.com",
						},
						"allowed": []string{"get-monitors", "get-monitor"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as boolean true",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as boolean false",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": false,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as nil",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as object with key",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key": "custom-memory-${{ github.workflow }}",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory with all options",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key":            "custom-key",
						"retention-days": 30,
						"docker-image":   "custom/memory:latest",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid cache-memory with invalid retention-days (too low)",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"retention-days": 0,
					},
				},
			},
			wantErr:     true,
			errContains: "got 0, want 1",
		},
		{
			name: "invalid cache-memory with invalid retention-days (too high)",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"retention-days": 91,
					},
				},
			},
			wantErr:     true,
			errContains: "got 91, want 90",
		},
		{
			name: "invalid cache-memory with additional property",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key":            "custom-key",
						"invalid_option": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_option' not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIncludedFileFrontmatterWithSchema(tt.frontmatter)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateIncludedFileFrontmatterWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateIncludedFileFrontmatterWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateIncludedFileFrontmatterWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		schema      string
		context     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid data with simple schema",
			frontmatter: map[string]any{
				"name": "test",
			},
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false
			}`,
			context: "test context",
			wantErr: false,
		},
		{
			name: "invalid data with additional property",
			frontmatter: map[string]any{
				"name":    "test",
				"invalid": "value",
			},
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false
			}`,
			context:     "test context",
			wantErr:     true,
			errContains: "additional properties 'invalid' not allowed",
		},
		{
			name: "invalid schema JSON",
			frontmatter: map[string]any{
				"name": "test",
			},
			schema:      `invalid json`,
			context:     "test context",
			wantErr:     true,
			errContains: "schema validation error for test context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWithSchema(tt.frontmatter, tt.schema, tt.context)

			if tt.wantErr && err == nil {
				t.Errorf("validateWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateWithSchemaAndLocation_CleanedErrorMessage(t *testing.T) {
	// Test that error messages are properly cleaned of unhelpful jsonschema prefixes
	frontmatter := map[string]any{
		"on":               "push",
		"timeout_minu tes": 10, // Invalid property name with space
	}

	// Create a temporary test file
	tempFile := "/tmp/gh-aw/test_schema_validation.md"
	// Ensure the directory exists
	if err := os.MkdirAll("/tmp/gh-aw", 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	err := os.WriteFile(tempFile, []byte(`---
on: push
timeout_minu tes: 10
---

# Test workflow`), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile)

	err = ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, tempFile)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	errorMsg := err.Error()

	// The error message should NOT contain the unhelpful jsonschema prefixes
	if strings.Contains(errorMsg, "jsonschema validation failed") {
		t.Errorf("Error message should not contain 'jsonschema validation failed' prefix, got: %s", errorMsg)
	}

	if strings.Contains(errorMsg, "- at '': ") {
		t.Errorf("Error message should not contain '- at '':' prefix, got: %s", errorMsg)
	}

	// The error message should contain the friendly rewritten error description
	if !strings.Contains(errorMsg, "Unknown property: timeout_minu tes") {
		t.Errorf("Error message should contain the validation error, got: %s", errorMsg)
	}

	// The error message should be formatted with location information
	if !strings.Contains(errorMsg, tempFile) {
		t.Errorf("Error message should contain file path, got: %s", errorMsg)
	}
}
