package workflow

import (
	"strings"
	"testing"
)

// TestValidateLogLevel tests the ValidateLogLevel function with various inputs
func TestValidateLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid log-level: debug",
			logLevel:  "debug",
			expectErr: false,
		},
		{
			name:      "valid log-level: info",
			logLevel:  "info",
			expectErr: false,
		},
		{
			name:      "valid log-level: warn",
			logLevel:  "warn",
			expectErr: false,
		},
		{
			name:      "valid log-level: error",
			logLevel:  "error",
			expectErr: false,
		},
		{
			name:      "empty log-level (allowed, defaults to info)",
			logLevel:  "",
			expectErr: false,
		},
		{
			name:      "invalid log-level: verbose",
			logLevel:  "verbose",
			expectErr: true,
			errMsg:    "invalid log-level 'verbose'",
		},
		{
			name:      "invalid log-level: trace",
			logLevel:  "trace",
			expectErr: true,
			errMsg:    "invalid log-level 'trace'",
		},
		{
			name:      "invalid log-level: warning",
			logLevel:  "warning",
			expectErr: true,
			errMsg:    "invalid log-level 'warning'",
		},
		{
			name:      "invalid log-level: fatal",
			logLevel:  "fatal",
			expectErr: true,
			errMsg:    "invalid log-level 'fatal'",
		},
		{
			name:      "invalid log-level: DEBUG (uppercase)",
			logLevel:  "DEBUG",
			expectErr: true,
			errMsg:    "invalid log-level 'DEBUG'",
		},
		{
			name:      "invalid log-level: Info (mixed case)",
			logLevel:  "Info",
			expectErr: true,
			errMsg:    "invalid log-level 'Info'",
		},
		{
			name:      "invalid log-level: random string",
			logLevel:  "random",
			expectErr: true,
			errMsg:    "invalid log-level 'random'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogLevel(tt.logLevel)
			if tt.expectErr {
				if err == nil {
					t.Errorf("ValidateLogLevel(%q) expected error but got none", tt.logLevel)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateLogLevel(%q) error = %q, should contain %q", tt.logLevel, err.Error(), tt.errMsg)
				}
				// Check that error message lists all valid options
				if !strings.Contains(err.Error(), "debug") || !strings.Contains(err.Error(), "info") ||
					!strings.Contains(err.Error(), "warn") || !strings.Contains(err.Error(), "error") {
					t.Errorf("ValidateLogLevel(%q) error message should list all valid options (debug, info, warn, error), got: %q", tt.logLevel, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("ValidateLogLevel(%q) unexpected error: %v", tt.logLevel, err)
				}
			}
		})
	}
}

// TestValidateFirewallConfig tests the validateFirewallConfig method
func TestValidateFirewallConfig(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expectErr    bool
		errMsg       string
	}{
		{
			name: "valid log-level: debug",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "debug",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid log-level: info",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "info",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid log-level: warn",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "warn",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid log-level: error",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "error",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "empty log-level (allowed)",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "no firewall config (allowed)",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{},
			},
			expectErr: false,
		},
		{
			name:         "no network permissions (allowed)",
			workflowData: &WorkflowData{},
			expectErr:    false,
		},
		{
			name: "invalid log-level: verbose",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "verbose",
					},
				},
			},
			expectErr: true,
			errMsg:    "invalid log-level 'verbose'",
		},
		{
			name: "invalid log-level: trace",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "trace",
					},
				},
			},
			expectErr: true,
			errMsg:    "invalid log-level 'trace'",
		},
		{
			name: "invalid log-level: DEBUG (uppercase)",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
						Enabled:  true,
						LogLevel: "DEBUG",
					},
				},
			},
			expectErr: true,
			errMsg:    "invalid log-level 'DEBUG'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			err := compiler.validateFirewallConfig(tt.workflowData)
			if tt.expectErr {
				if err == nil {
					t.Errorf("validateFirewallConfig() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateFirewallConfig() error = %q, should contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateFirewallConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateLogLevelErrorMessageQuality tests the error message quality
func TestValidateLogLevelErrorMessageQuality(t *testing.T) {
	err := ValidateLogLevel("verbose")
	if err == nil {
		t.Fatal("Expected error for invalid log-level 'verbose'")
	}

	errMsg := err.Error()

	// Check that error message contains key information
	requiredStrings := []string{
		"verbose",           // The invalid value
		"invalid log-level", // Clear error type
		"debug",             // Valid option 1
		"info",              // Valid option 2
		"warn",              // Valid option 3
		"error",             // Valid option 4
	}

	for _, required := range requiredStrings {
		if !strings.Contains(errMsg, required) {
			t.Errorf("Error message should contain %q, got: %q", required, errMsg)
		}
	}

	// Check that error message is concise and helpful
	if len(errMsg) > 200 {
		t.Errorf("Error message is too long (%d chars): %q", len(errMsg), errMsg)
	}
}

// TestValidateFirewallConfigIntegration tests the integration with workflow compilation
func TestValidateFirewallConfigIntegration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that valid log-level passes through compilation validation
	validWorkflow := &WorkflowData{
		Name: "test-workflow",
		NetworkPermissions: &NetworkPermissions{
				Enabled:  true,
				LogLevel: "debug",
			},
		},
	}

	err := compiler.validateFirewallConfig(validWorkflow)
	if err != nil {
		t.Errorf("Valid firewall config should not produce error: %v", err)
	}

	// Test that invalid log-level is caught during compilation validation
	invalidWorkflow := &WorkflowData{
		Name: "test-workflow",
		NetworkPermissions: &NetworkPermissions{
				Enabled:  true,
				LogLevel: "verbose",
			},
		},
	}

	err = compiler.validateFirewallConfig(invalidWorkflow)
	if err == nil {
		t.Error("Invalid firewall config should produce error")
	} else if !strings.Contains(err.Error(), "verbose") {
		t.Errorf("Error should mention the invalid value 'verbose', got: %v", err)
	}
}
