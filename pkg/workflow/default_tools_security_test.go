package workflow

import (
	"reflect"
	"strings"
	"testing"
)

// TestDefaultGitHubToolsAreReadOnly validates that all default GitHub tools are read-only operations.
// This test enforces the security policy that default tools must not include write operations.
func TestDefaultGitHubToolsAreReadOnly(t *testing.T) {
	// Get the default tools by calling applyDefaultTools with empty input
	compiler := &Compiler{}
	tools := compiler.applyDefaultTools(nil, nil)

	// Extract the github tool configuration
	githubTool, exists := tools["github"]
	if !exists {
		t.Fatal("GitHub tool not found in default tools")
	}

	githubConfig, ok := githubTool.(map[string]any)
	if !ok {
		t.Fatal("GitHub tool is not a map")
	}

	allowed, exists := githubConfig["allowed"]
	if !exists {
		t.Fatal("GitHub tool does not have 'allowed' field")
	}

	allowedSlice, ok := allowed.([]any)
	if !ok {
		t.Fatal("GitHub tool 'allowed' field is not a slice")
	}

	// Convert to string slice for easier testing
	var defaultTools []string
	for _, tool := range allowedSlice {
		if toolStr, ok := tool.(string); ok {
			defaultTools = append(defaultTools, toolStr)
		}
	}

	// Define write operation patterns
	writeOperationPrefixes := []string{
		"create_", "add_", "update_", "delete_", "remove_", "set_", "patch_",
		"modify_", "edit_", "post_", "put_", "cancel_", "close_", "reopen_",
	}

	// Check each default tool
	for _, tool := range defaultTools {
		for _, prefix := range writeOperationPrefixes {
			if strings.HasPrefix(tool, prefix) {
				t.Errorf("SECURITY VIOLATION: Default tool '%s' appears to be a write operation (prefix: %s). Default tools must be read-only only.", tool, prefix)
			}
		}

		// Also validate that tools follow expected read-only patterns
		if !isReadOnlyOperation(tool) {
			t.Errorf("Tool '%s' does not follow expected read-only patterns (get_, list_, search_, download_)", tool)
		}
	}
}

// TestValidateDefaultToolsReadOnlyFunction tests the validation function itself
func TestValidateDefaultToolsReadOnlyFunction(t *testing.T) {
	tests := []struct {
		name        string
		tools       []string
		shouldPanic bool
		description string
	}{
		{
			name:        "all read-only tools should pass",
			tools:       []string{"get_issue", "list_issues", "search_repositories", "download_artifact"},
			shouldPanic: false,
			description: "Standard read-only tools should not trigger validation",
		},
		{
			name:        "create operation should panic",
			tools:       []string{"get_issue", "create_issue"},
			shouldPanic: true,
			description: "create_ prefix should trigger security validation",
		},
		{
			name:        "update operation should panic",
			tools:       []string{"update_issue", "get_issue"},
			shouldPanic: true,
			description: "update_ prefix should trigger security validation",
		},
		{
			name:        "add operation should panic",
			tools:       []string{"get_issue", "add_issue_comment"},
			shouldPanic: true,
			description: "add_ prefix should trigger security validation",
		},
		{
			name:        "delete operation should panic",
			tools:       []string{"delete_issue", "list_issues"},
			shouldPanic: true,
			description: "delete_ prefix should trigger security validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.shouldPanic && r == nil {
					t.Errorf("Expected panic for %s, but function did not panic", tt.description)
				} else if !tt.shouldPanic && r != nil {
					t.Errorf("Unexpected panic for %s: %v", tt.description, r)
				}
			}()

			validateDefaultToolsReadOnly(tt.tools)
		})
	}
}

// TestCurrentDefaultToolsMatchReadOnlyPattern verifies all current default tools follow read-only patterns
func TestCurrentDefaultToolsMatchReadOnlyPattern(t *testing.T) {
	// These are the actual tools defined in applyDefaultTools as of the time this test was written
	// This test ensures they all follow read-only patterns
	currentDefaultTools := []string{
		// actions
		"download_workflow_run_artifact",
		"get_job_logs",
		"get_workflow_run",
		"get_workflow_run_logs",
		"get_workflow_run_usage",
		"list_workflow_jobs",
		"list_workflow_run_artifacts",
		"list_workflow_runs",
		"list_workflows",
		// code security
		"get_code_scanning_alert",
		"list_code_scanning_alerts",
		// context
		"get_me",
		// dependabot
		"get_dependabot_alert",
		"list_dependabot_alerts",
		// discussions
		"get_discussion",
		"get_discussion_comments",
		"list_discussion_categories",
		"list_discussions",
		// issues
		"get_issue",
		"get_issue_comments",
		"list_issues",
		"search_issues",
		// notifications
		"get_notification_details",
		"list_notifications",
		// organizations
		"search_orgs",
		// prs
		"get_pull_request",
		"get_pull_request_comments",
		"get_pull_request_diff",
		"get_pull_request_files",
		"get_pull_request_reviews",
		"get_pull_request_status",
		"list_pull_requests",
		"search_pull_requests",
		// repos
		"get_commit",
		"get_file_contents",
		"get_tag",
		"list_branches",
		"list_commits",
		"list_tags",
		"search_code",
		"search_repositories",
		// secret protection
		"get_secret_scanning_alert",
		"list_secret_scanning_alerts",
		// users
		"search_users",
	}

	for _, tool := range currentDefaultTools {
		if !isReadOnlyOperation(tool) {
			t.Errorf("Current default tool '%s' does not follow read-only patterns", tool)
		}
	}
}

// isReadOnlyOperation checks if a tool name follows read-only operation patterns
func isReadOnlyOperation(tool string) bool {
	readOnlyPrefixes := []string{
		"get_", "list_", "search_", "download_",
	}

	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(tool, prefix) {
			return true
		}
	}

	return false
}

// TestDefaultToolsPolicy documents the expected policy for default tools
func TestDefaultToolsPolicy(t *testing.T) {
	t.Log("Default GitHub tools policy:")
	t.Log("1. All default tools MUST be read-only operations")
	t.Log("2. Write operations (create_, update_, delete_, add_, remove_) are PROHIBITED")
	t.Log("3. Default tools should follow patterns: get_, list_, search_, download_")
	t.Log("4. Users must explicitly configure write operations in their workflow's allowed tools")
	t.Log("5. This ensures minimal permissions by default (principle of least privilege)")
}

// TestApplyDefaultToolsIdempotent ensures that applying default tools multiple times doesn't change the result
func TestApplyDefaultToolsIdempotent(t *testing.T) {
	compiler := &Compiler{}

	// Apply default tools once
	tools1 := compiler.applyDefaultTools(nil, nil)

	// Apply default tools again with the result
	tools2 := compiler.applyDefaultTools(tools1, nil)

	// Should be the same
	if !reflect.DeepEqual(tools1, tools2) {
		t.Error("Applying default tools should be idempotent")
	}
}
