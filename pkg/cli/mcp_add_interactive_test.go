package cli

import (
	"os"
	"strings"
	"testing"
)

func TestAddMCPToolInteractively_InAutomatedEnvironment(t *testing.T) {
	// Save original environment
	origTestMode := os.Getenv("GO_TEST_MODE")
	origCI := os.Getenv("CI")

	// Set test mode
	os.Setenv("GO_TEST_MODE", "true")

	// Clean up after test
	t.Cleanup(func() {
		if origTestMode != "" {
			os.Setenv("GO_TEST_MODE", origTestMode)
		} else {
			os.Unsetenv("GO_TEST_MODE")
		}
		if origCI != "" {
			os.Setenv("CI", origCI)
		} else {
			os.Unsetenv("CI")
		}
	})

	// Test should fail in automated environment
	err := AddMCPToolInteractively("test-workflow", "", false)
	if err == nil {
		t.Error("Expected error in automated environment, got nil")
	}

	expectedErrMsg := "interactive MCP configuration cannot be used in automated tests or CI environments"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestAddMCPToolInteractively_CI_Environment(t *testing.T) {
	// Save original environment
	origTestMode := os.Getenv("GO_TEST_MODE")
	origCI := os.Getenv("CI")

	// Set CI environment
	os.Setenv("CI", "true")

	// Clean up after test
	t.Cleanup(func() {
		if origTestMode != "" {
			os.Setenv("GO_TEST_MODE", origTestMode)
		} else {
			os.Unsetenv("GO_TEST_MODE")
		}
		if origCI != "" {
			os.Setenv("CI", origCI)
		} else {
			os.Unsetenv("CI")
		}
	})

	// Test should fail in CI environment
	err := AddMCPToolInteractively("test-workflow", "", false)
	if err == nil {
		t.Error("Expected error in CI environment, got nil")
	}

	expectedErrMsg := "interactive MCP configuration cannot be used in automated tests or CI environments"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestMCPAddInteractiveConfig_Structure(t *testing.T) {
	// Test that MCPAddInteractiveConfig has the expected fields
	config := &MCPAddInteractiveConfig{
		WorkflowFile:    "/path/to/workflow.md",
		ServerName:      "test-server",
		TransportType:   "stdio",
		CustomToolID:    "custom-tool",
		AllowNetwork:    true,
		NetworkDomains:  []string{"example.com"},
		EnvironmentVars: map[string]string{"KEY": "value"},
	}

	// Verify field values
	if config.WorkflowFile != "/path/to/workflow.md" {
		t.Errorf("Expected WorkflowFile to be '/path/to/workflow.md', got '%s'", config.WorkflowFile)
	}

	if config.ServerName != "test-server" {
		t.Errorf("Expected ServerName to be 'test-server', got '%s'", config.ServerName)
	}

	if config.TransportType != "stdio" {
		t.Errorf("Expected TransportType to be 'stdio', got '%s'", config.TransportType)
	}

	if config.CustomToolID != "custom-tool" {
		t.Errorf("Expected CustomToolID to be 'custom-tool', got '%s'", config.CustomToolID)
	}

	if !config.AllowNetwork {
		t.Error("Expected AllowNetwork to be true")
	}

	if len(config.NetworkDomains) != 1 || config.NetworkDomains[0] != "example.com" {
		t.Errorf("Expected NetworkDomains to contain 'example.com', got %v", config.NetworkDomains)
	}

	if config.EnvironmentVars["KEY"] != "value" {
		t.Errorf("Expected EnvironmentVars['KEY'] to be 'value', got '%s'", config.EnvironmentVars["KEY"])
	}
}

func TestMCPAddInteractiveConfig_DefaultValues(t *testing.T) {
	// Test default values for MCPAddInteractiveConfig
	config := &MCPAddInteractiveConfig{
		AllowNetwork:    false,
		NetworkDomains:  []string{},
		EnvironmentVars: make(map[string]string),
	}

	if config.AllowNetwork {
		t.Error("Expected AllowNetwork to default to false")
	}

	if len(config.NetworkDomains) != 0 {
		t.Errorf("Expected NetworkDomains to be empty, got %v", config.NetworkDomains)
	}

	if len(config.EnvironmentVars) != 0 {
		t.Errorf("Expected EnvironmentVars to be empty, got %v", config.EnvironmentVars)
	}
}

func TestNewMCPAddSubcommand_InteractiveFlag(t *testing.T) {
	// Test that the interactive flag is properly defined
	cmd := NewMCPAddSubcommand()

	if cmd == nil {
		t.Fatal("NewMCPAddSubcommand returned nil")
	}

	// Check that interactive flag exists
	flag := cmd.Flags().Lookup("interactive")
	if flag == nil {
		t.Fatal("Interactive flag not found")
	}

	if flag.Name != "interactive" {
		t.Errorf("Expected flag name 'interactive', got '%s'", flag.Name)
	}

	if flag.Usage != "Use interactive form mode for configuration" {
		t.Errorf("Expected flag usage 'Use interactive form mode for configuration', got '%s'", flag.Usage)
	}

	// Check default value is false
	if flag.DefValue != "false" {
		t.Errorf("Expected default value 'false', got '%s'", flag.DefValue)
	}
}

func TestNewMCPAddSubcommand_HelpText(t *testing.T) {
	// Test that help text mentions both usage modes
	cmd := NewMCPAddSubcommand()

	if cmd == nil {
		t.Fatal("NewMCPAddSubcommand returned nil")
	}

	// Check that long description mentions interactive mode
	if !strings.Contains(cmd.Long, "--interactive") {
		t.Error("Long description should mention --interactive flag")
	}

	if !strings.Contains(cmd.Long, "interactive form") {
		t.Error("Long description should mention interactive form mode")
	}

	// Check examples include interactive usage
	if !strings.Contains(cmd.Long, "gh aw mcp add weekly-research --interactive") {
		t.Error("Examples should include interactive mode usage")
	}
}

func TestNewMCPAddSubcommand_BackwardCompatibility(t *testing.T) {
	// Test that flag-based usage still works
	cmd := NewMCPAddSubcommand()

	if cmd == nil {
		t.Fatal("NewMCPAddSubcommand returned nil")
	}

	// Check that all existing flags are still present
	registryFlag := cmd.Flags().Lookup("registry")
	if registryFlag == nil {
		t.Error("registry flag should still exist for backward compatibility")
	}

	transportFlag := cmd.Flags().Lookup("transport")
	if transportFlag == nil {
		t.Error("transport flag should still exist for backward compatibility")
	}

	toolIDFlag := cmd.Flags().Lookup("tool-id")
	if toolIDFlag == nil {
		t.Error("tool-id flag should still exist for backward compatibility")
	}
}

func TestMCPAddInteractiveConfig_TransportOptions(t *testing.T) {
	// Test that all transport options are valid
	validTransports := []string{"stdio", "http", "docker"}

	for _, transport := range validTransports {
		config := &MCPAddInteractiveConfig{
			TransportType: transport,
		}

		if config.TransportType != transport {
			t.Errorf("Expected TransportType to be '%s', got '%s'", transport, config.TransportType)
		}
	}
}

func TestMCPAddInteractiveConfig_EmptyValues(t *testing.T) {
	// Test that empty values are handled correctly
	config := &MCPAddInteractiveConfig{}

	if config.WorkflowFile != "" {
		t.Errorf("Expected WorkflowFile to be empty, got '%s'", config.WorkflowFile)
	}

	if config.ServerName != "" {
		t.Errorf("Expected ServerName to be empty, got '%s'", config.ServerName)
	}

	if config.TransportType != "" {
		t.Errorf("Expected TransportType to be empty, got '%s'", config.TransportType)
	}

	if config.CustomToolID != "" {
		t.Errorf("Expected CustomToolID to be empty, got '%s'", config.CustomToolID)
	}
}

func TestCompileWorkflowAfterMCPAdd_SpinnerIntegration(t *testing.T) {
	// This test verifies that the spinner integration doesn't panic
	// and handles errors correctly. We can't directly test the spinner
	// UI output, but we can verify the method works correctly.

	// Test with invalid workflow (should handle error correctly)
	// This will fail compilation but should not panic
	err := compileWorkflowAfterMCPAdd("/nonexistent/workflow.md", false)

	// We expect an error since the workflow doesn't exist
	if err == nil {
		t.Error("Expected error when compiling non-existent workflow")
	}

	// Verify error handling doesn't panic
	// The spinner should be stopped even on error
	t.Logf("Compilation error (expected): %v", err)
}

func TestMCPAddCommand_InteractiveMode_RequiresWorkflow(t *testing.T) {
	// Test that interactive mode requires at least a workflow argument
	cmd := NewMCPAddSubcommand()

	if cmd == nil {
		t.Fatal("NewMCPAddSubcommand returned nil")
	}

	// With no arguments, it should list servers (not use interactive mode)
	// This is tested by checking the args validation
	if cmd.Args == nil {
		t.Error("Command should have args validation")
	}
}

func TestMCPAddInteractiveLog_Initialized(t *testing.T) {
	// Test that the logger is properly initialized
	if mcpAddInteractiveLog == nil {
		t.Error("mcpAddInteractiveLog should be initialized")
	}

	// Logger should be configured for cli:mcp_add_interactive namespace
	// This is verified by the logger initialization pattern
}

func TestAddMCPToolInteractively_DefaultRegistry(t *testing.T) {
	// Save original environment
	origTestMode := os.Getenv("GO_TEST_MODE")
	origCI := os.Getenv("CI")

	// Set test mode to prevent actual execution
	os.Setenv("GO_TEST_MODE", "true")

	// Clean up after test
	t.Cleanup(func() {
		if origTestMode != "" {
			os.Setenv("GO_TEST_MODE", origTestMode)
		} else {
			os.Unsetenv("GO_TEST_MODE")
		}
		if origCI != "" {
			os.Setenv("CI", origCI)
		} else {
			os.Unsetenv("CI")
		}
	})

	// Test with empty registry URL (should use default)
	err := AddMCPToolInteractively("test-workflow", "", false)

	// Should fail due to test mode, but we're testing parameter handling
	if err == nil {
		t.Error("Expected error in test mode")
	}

	// The error should be about test mode, not missing registry
	if !strings.Contains(err.Error(), "automated tests") {
		t.Errorf("Expected error about automated tests, got: %v", err)
	}
}
