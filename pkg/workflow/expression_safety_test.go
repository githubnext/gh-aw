package workflow

import (
	"strings"
	"testing"
)

func TestValidateExpressionSafety(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectError    bool
		expectedErrors []string
	}{
		{
			name:        "no_expressions",
			content:     "This is a simple markdown with no expressions",
			expectError: false,
		},
		{
			name:        "allowed_github_workflow",
			content:     "The workflow name is ${{ github.workflow }}",
			expectError: false,
		},
		{
			name:        "allowed_github_repository",
			content:     "Repository: ${{ github.repository }}",
			expectError: false,
		},
		{
			name:        "allowed_github_run_id",
			content:     "Run ID: ${{ github.run_id }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_issue_number",
			content:     "Issue number: ${{ github.event.issue.number }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_discussion_number",
			content:     "Discussion number: ${{ github.event.discussion.number }}",
			expectError: false,
		},
		{
			name:        "allowed_needs_task_outputs_text",
			content:     "Task output: ${{ needs.activation.outputs.text }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_inputs",
			content:     "User input: ${{ github.event.inputs.name }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_inputs_underscore",
			content:     "Branch input: ${{ github.event.inputs.target_branch }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_inputs_hyphen",
			content:     "Deploy input: ${{ github.event.inputs.deploy-environment }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_conclusion",
			content:     "Workflow conclusion: ${{ github.event.workflow_run.conclusion }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_html_url",
			content:     "Run URL: ${{ github.event.workflow_run.html_url }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_head_sha",
			content:     "Head SHA: ${{ github.event.workflow_run.head_sha }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_run_number",
			content:     "Run number: ${{ github.event.workflow_run.run_number }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_event",
			content:     "Triggering event: ${{ github.event.workflow_run.event }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_workflow_run_status",
			content:     "Run status: ${{ github.event.workflow_run.status }}",
			expectError: false,
		},
		{
			name:        "allowed_github_event_repository_default_branch",
			content:     "Default branch: ${{ github.event.repository.default_branch }}",
			expectError: false,
		},
		{
			name:        "multiple_allowed_expressions",
			content:     "Workflow: ${{ github.workflow }}, Repository: ${{ github.repository }}, Output: ${{ needs.activation.outputs.text }}",
			expectError: false,
		},
		{
			name:           "unauthorized_github_token",
			content:        "Using token: ${{ secrets.GITHUB_TOKEN }}",
			expectError:    true,
			expectedErrors: []string{"secrets.GITHUB_TOKEN"},
		},
		{
			name:        "authorized_github_actor",
			content:     "Actor: ${{ github.actor }}",
			expectError: false,
		},
		{
			name:        "authorized_env_variable",
			content:     "Environment: ${{ env.MY_VAR }}",
			expectError: false,
		},
		{
			name:        "unauthorized_steps_output",
			content:     "Step output: ${{ steps.my-step.outputs.result }}",
			expectError: false,
			// Note: steps outputs are allowed, but this is a test case to ensure it
			expectedErrors: []string{"steps.my-step.outputs.result"},
		},
		{
			name:           "mixed_authorized_and_unauthorized",
			content:        "Valid: ${{ github.workflow }}, Invalid: ${{ secrets.API_KEY }}",
			expectError:    true,
			expectedErrors: []string{"secrets.API_KEY"},
		},
		{
			name:           "multiple_unauthorized_expressions",
			content:        "Token: ${{ secrets.GITHUB_TOKEN }}, Valid: ${{ github.actor }}, Env: ${{ env.TEST }}",
			expectError:    true,
			expectedErrors: []string{"secrets.GITHUB_TOKEN"},
		},
		{
			name:        "expressions_with_whitespace",
			content:     "Spaced: ${{   github.workflow   }}, Normal: ${{github.repository}}",
			expectError: false,
		},
		{
			name:           "expressions_with_unauthorized_whitespace",
			content:        "Invalid spaced: ${{   secrets.TOKEN   }}",
			expectError:    true,
			expectedErrors: []string{"secrets.TOKEN"},
		},
		{
			name:        "expressions_in_code_blocks",
			content:     "Code example: `${{ github.workflow }}` and ```${{ github.repository }}```",
			expectError: false,
		},
		{
			name:           "unauthorized_in_code_blocks",
			content:        "Code example: `${{ secrets.TOKEN }}` should still be caught",
			expectError:    true,
			expectedErrors: []string{"secrets.TOKEN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpressionSafety(tt.content)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil {
				// Check that all expected unauthorized expressions are mentioned in the error
				errorMsg := err.Error()
				for _, expectedError := range tt.expectedErrors {
					if !strings.Contains(errorMsg, expectedError) {
						t.Errorf("Expected error message to contain '%s', but got: %s", expectedError, errorMsg)
					}
				}
			}
		})
	}
}

func TestValidateExpressionSafetyEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		description string
	}{
		{
			name:        "empty_expression",
			content:     "Empty: ${{ }}",
			expectError: true,
			description: "Empty expressions should be considered unauthorized",
		},
		{
			name:        "malformed_expression_single_brace",
			content:     "Malformed: ${ github.workflow }",
			expectError: false,
			description: "Malformed expressions (single brace) should be ignored",
		},
		{
			name:        "malformed_expression_no_closing",
			content:     "Malformed: ${{ github.workflow",
			expectError: false,
			description: "Malformed expressions (no closing) should be ignored",
		},
		{
			name:        "nested_expressions",
			content:     "Nested: ${{ ${{ github.workflow }} }}",
			expectError: true,
			description: "Nested expressions should be caught",
		},
		{
			name:        "expression_with_functions",
			content:     "Function: ${{ toJson(github.workflow) }}",
			expectError: true,
			description: "Expressions with functions should be unauthorized unless the base expression is allowed",
		},
		{
			name:        "multiline_expression",
			content:     "Multi:\n${{ github.workflow\n}}",
			expectError: true,
			description: "Should NOT handle expressions spanning multiple lines - though this is unusual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpressionSafety(tt.content)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.description, err)
			}
		})
	}
}

// TestValidateExpressionSafetyWithParser tests the new parser functionality in expression safety
func TestValidateExpressionSafetyWithParser(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "allowed AND expression",
			content: `${{ github.workflow && github.repository }}`,
			wantErr: false,
		},
		{
			name:    "allowed OR expression",
			content: `${{ github.workflow || github.repository }}`,
			wantErr: false,
		},
		{
			name:    "allowed NOT expression",
			content: `${{ !github.workflow }}`,
			wantErr: false,
		},
		{
			name:    "complex allowed expression with parentheses",
			content: `${{ (github.workflow && github.repository) || github.run_id }}`,
			wantErr: false,
		},
		{
			name:        "mixed allowed and unauthorized",
			content:     `${{ github.workflow && secrets.TOKEN }}`,
			wantErr:     true,
			errContains: "secrets.TOKEN",
		},
		{
			name:        "unauthorized in complex expression",
			content:     `${{ (github.workflow || secrets.TOKEN) && github.repository }}`,
			wantErr:     true,
			errContains: "secrets.TOKEN",
		},
		{
			name:    "nested complex allowed expression",
			content: `${{ ((github.workflow && github.repository) || github.run_id) && github.actor }}`,
			wantErr: false,
		},
		{
			name:        "NOT with unauthorized expression",
			content:     `${{ !secrets.TOKEN }}`,
			wantErr:     true,
			errContains: "secrets.TOKEN",
		},
		{
			name:    "unparseable but allowed literal",
			content: `${{ github.workflow }}`,
			wantErr: false,
		},
		{
			name:        "unparseable and unauthorized literal",
			content:     `${{ secrets.INVALID_TOKEN }}`,
			wantErr:     true,
			errContains: "secrets.INVALID_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpressionSafety(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateExpressionSafety() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateExpressionSafety() error = %v, should contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateExpressionSafety() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUnauthorizedExpressionErrorFormat(t *testing.T) {
// Test that the error message is properly formatted with line breaks
content := "Test: ${{ github.inputs.test }}"
err := validateExpressionSafety(content)

if err == nil {
t.Fatal("Expected error for unauthorized expression")
}

errMsg := err.Error()

// Check that the error message contains the unauthorized expression with bullet point
if !strings.Contains(errMsg, "  - github.inputs.test") {
t.Errorf("Error message should contain bullet-pointed unauthorized expression, got: %s", errMsg)
}

// Check that the error message contains allowed expressions with bullet points
if !strings.Contains(errMsg, "  - github.event.issue.number") {
t.Errorf("Error message should contain bullet-pointed allowed expressions, got: %s", errMsg)
}

// Check that the error message contains pattern-based allowed expressions
if !strings.Contains(errMsg, "  - needs.*") {
t.Errorf("Error message should contain pattern-based allowed expressions, got: %s", errMsg)
}

// Check that the error has the proper structure with newlines
if !strings.Contains(errMsg, "unauthorized expressions:\n") {
t.Errorf("Error message should have 'unauthorized expressions:' followed by newline, got: %s", errMsg)
}

if !strings.Contains(errMsg, "\nallowed:\n") {
t.Errorf("Error message should have 'allowed:' on its own line, got: %s", errMsg)
}
}
