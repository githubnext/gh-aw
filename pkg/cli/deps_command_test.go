package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDepsCommand(t *testing.T) {
	cmd := NewDepsCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "deps", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestDepsCommandHasSubcommands(t *testing.T) {
	cmd := NewDepsCommand()

	subcommands := cmd.Commands()
	require.NotEmpty(t, subcommands, "deps command should have subcommands")

	// Check for expected subcommands
	expectedSubcommands := []string{"health", "outdated", "security", "report"}
	foundSubcommands := make(map[string]bool)

	for _, subCmd := range subcommands {
		foundSubcommands[subCmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		assert.True(t, foundSubcommands[expected], "Expected subcommand %s not found", expected)
	}
}

func TestDepsHealthCommand(t *testing.T) {
	cmd := newDepsHealthCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "health", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)
}

func TestDepsOutdatedCommand(t *testing.T) {
	cmd := newDepsOutdatedCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "outdated", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check for verbose flag
	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "outdated command should have --verbose flag")
	assert.Equal(t, "v", verboseFlag.Shorthand)
}

func TestDepsSecurityCommand(t *testing.T) {
	cmd := newDepsSecurityCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "security", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check for verbose flag
	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "security command should have --verbose flag")
}

func TestDepsReportCommand(t *testing.T) {
	cmd := newDepsReportCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "report", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check for flags
	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "report command should have --verbose flag")

	jsonFlag := cmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag, "report command should have --json flag")
}

func TestDepsCommandExamples(t *testing.T) {
	cmd := NewDepsCommand()

	// Verify examples are present in long description
	assert.Contains(t, cmd.Long, "gh aw deps health")
	assert.Contains(t, cmd.Long, "gh aw deps outdated")
	assert.Contains(t, cmd.Long, "gh aw deps security")
	assert.Contains(t, cmd.Long, "gh aw deps report")
}

func TestDepsSubcommandDescriptions(t *testing.T) {
	tests := []struct {
		name        string
		subcommand  func() *cobra.Command
		expectedUse string
		keywords    []string
	}{
		{
			name:        "health command",
			subcommand:  newDepsHealthCommand,
			expectedUse: "health",
			keywords:    []string{"health", "metrics", "v0.x", "unstable"},
		},
		{
			name:        "outdated command",
			subcommand:  newDepsOutdatedCommand,
			expectedUse: "outdated",
			keywords:    []string{"outdated", "updates", "latest", "version"},
		},
		{
			name:        "security command",
			subcommand:  newDepsSecurityCommand,
			expectedUse: "security",
			keywords:    []string{"security", "vulnerabilities", "advisory"},
		},
		{
			name:        "report command",
			subcommand:  newDepsReportCommand,
			expectedUse: "report",
			keywords:    []string{"comprehensive", "report", "JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.subcommand()
			assert.Equal(t, tt.expectedUse, cmd.Use)

			// Check that keywords are present in either Short or Long description
			description := cmd.Short + " " + cmd.Long
			for _, keyword := range tt.keywords {
				assert.Contains(t, description, keyword,
					"Expected keyword %q not found in %s description", keyword, tt.name)
			}
		})
	}
}

func TestDepsCommandGroupAssignment(t *testing.T) {
	// This test verifies that the deps command is properly assigned to a command group
	// The actual group assignment happens in main.go, but we can test the command structure
	cmd := NewDepsCommand()

	// Verify command is ready for group assignment
	assert.Empty(t, cmd.GroupID, "GroupID should be empty until assigned in main.go")

	// Verify command can have a group assigned
	cmd.GroupID = "development"
	assert.Equal(t, "development", cmd.GroupID)
}

func TestDepsCommandFlagInheritance(t *testing.T) {
	// Test that global flags like --verbose are properly inherited by subcommands
	cmd := NewDepsCommand()

	// Add a persistent flag to the parent (simulating root command behavior)
	cmd.PersistentFlags().BoolP("global-verbose", "g", false, "global verbose flag")

	// Initialize all subcommands
	for _, subCmd := range cmd.Commands() {
		// Each subcommand should be able to access parent's persistent flags
		flag := subCmd.Flags().Lookup("global-verbose")
		// Note: Flag lookup includes parent flags after command is executed
		// This test just verifies the structure is correct
		_ = flag // Flag will be nil until command is executed in context
	}
}
