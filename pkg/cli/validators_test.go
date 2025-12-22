package cli

import (
	"strings"
	"testing"
)

func TestValidateWorkflowName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple name",
			input:       "my-workflow",
			expectError: false,
		},
		{
			name:        "valid with underscores",
			input:       "my_workflow",
			expectError: false,
		},
		{
			name:        "valid alphanumeric",
			input:       "workflow123",
			expectError: false,
		},
		{
			name:        "valid mixed",
			input:       "my-workflow_v2",
			expectError: false,
		},
		{
			name:        "valid uppercase",
			input:       "MyWorkflow",
			expectError: false,
		},
		{
			name:        "valid all hyphens and underscores",
			input:       "my-workflow_test-123",
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "workflow name cannot be empty",
		},
		{
			name:        "invalid with spaces",
			input:       "my workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with special chars",
			input:       "my@workflow!",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with dots",
			input:       "my.workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with slashes",
			input:       "my/workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with parentheses",
			input:       "my(workflow)",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with brackets",
			input:       "my[workflow]",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with dollar sign",
			input:       "my$workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with percent sign",
			input:       "my%workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with hash",
			input:       "my#workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with asterisk",
			input:       "my*workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with ampersand",
			input:       "my&workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with plus",
			input:       "my+workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
		{
			name:        "invalid with equals",
			input:       "my=workflow",
			expectError: true,
			errorMsg:    "workflow name must contain only alphanumeric characters, hyphens, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowName(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateWorkflowName(%q) expected error but got nil", tt.input)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateWorkflowName(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWorkflowName(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

func TestValidateWorkflowName_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "single character",
			input:       "a",
			expectError: false,
		},
		{
			name:        "single number",
			input:       "1",
			expectError: false,
		},
		{
			name:        "single hyphen",
			input:       "-",
			expectError: false,
		},
		{
			name:        "single underscore",
			input:       "_",
			expectError: false,
		},
		{
			name:        "very long valid name",
			input:       strings.Repeat("a", 100),
			expectError: false,
		},
		{
			name:        "starts with hyphen",
			input:       "-workflow",
			expectError: false,
		},
		{
			name:        "ends with hyphen",
			input:       "workflow-",
			expectError: false,
		},
		{
			name:        "starts with underscore",
			input:       "_workflow",
			expectError: false,
		},
		{
			name:        "ends with underscore",
			input:       "workflow_",
			expectError: false,
		},
		{
			name:        "starts with number",
			input:       "123workflow",
			expectError: false,
		},
		{
			name:        "multiple consecutive hyphens",
			input:       "my--workflow",
			expectError: false,
		},
		{
			name:        "multiple consecutive underscores",
			input:       "my__workflow",
			expectError: false,
		},
		{
			name:        "tab character",
			input:       "my\tworkflow",
			expectError: true,
		},
		{
			name:        "newline character",
			input:       "my\nworkflow",
			expectError: true,
		},
		{
			name:        "carriage return",
			input:       "my\rworkflow",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowName(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("ValidateWorkflowName(%q) expected error but got nil", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateWorkflowName(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}
