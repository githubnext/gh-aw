//go:build !integration

package cli

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDoctorCommand(t *testing.T) {
	cmd := NewDoctorCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "doctor", cmd.Use)
	assert.Equal(t, "Run diagnostics to verify CLI authentication and repository setup", cmd.Short)
	assert.False(t, cmd.Hidden)
	assert.NotNil(t, cmd.Flags().Lookup("json"), "should expose --json flag")
	assert.NotNil(t, cmd.Flags().Lookup("repo"), "should expose --repo flag")
	assert.NotNil(t, cmd.Flags().Lookup("dir"), "should expose --dir flag")
	assert.NotNil(t, cmd.Flags().Lookup("require-owner-type"), "should expose --require-owner-type flag")
	assert.False(t, cmd.HasSubCommands())
}

func TestDoctorCommandUsesNoArgs(t *testing.T) {
	cmd := NewDoctorCommand()
	require.NotNil(t, cmd.Args)
	require.NoError(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
}

func TestDoctorCommandAdvertisesJSONExample(t *testing.T) {
	cmd := NewDoctorCommand()
	assert.Contains(t, cmd.Example, "gh aw doctor --json")
	assert.Contains(t, cmd.Example, "gh aw doctor --repo github/gh-aw --json")
}

func TestDoctorCommandExampleHasNoTabs(t *testing.T) {
	cmd := NewDoctorCommand()
	assert.NotContains(t, cmd.Example, "\t")
}

func TestDoctorCommandRequireOwnerTypeDefault(t *testing.T) {
	cmd := NewDoctorCommand()
	require.Equal(t, "any", cmd.Flags().Lookup("require-owner-type").DefValue)
}

func TestDoctorCommandInheritsVerboseFlagFromRoot(t *testing.T) {
	cmd := NewDoctorCommand()
	assert.Nil(t, cmd.Flags().Lookup("verbose"))

	root := &cobra.Command{Use: "aw"}
	var verbose bool
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing detailed information")
	root.AddCommand(cmd)

	inherited := cmd.InheritedFlags().Lookup("verbose")
	require.NotNil(t, inherited)
	assert.Equal(t, "false", inherited.DefValue)
}

func TestDoctorCommandRunsAuthOnlyWhenRepoNotProvided(t *testing.T) {
	origAuth := runDoctorSetupAuth
	origRepo := runDoctorSetupRepositoryCheck
	t.Cleanup(func() {
		runDoctorSetupAuth = origAuth
		runDoctorSetupRepositoryCheck = origRepo
	})

	calledAuth := false
	var gotJSON bool
	runDoctorSetupAuth = func(opts SetupAuthOptions) error {
		calledAuth = true
		gotJSON = opts.JSON
		return nil
	}
	runDoctorSetupRepositoryCheck = func(opts SetupRepositoryCheckOptions) error {
		return errors.New("unexpected repository check")
	}

	cmd := NewDoctorCommand()
	cmd.SetArgs([]string{"--json"})
	require.NoError(t, cmd.Execute())
	assert.True(t, calledAuth)
	assert.True(t, gotJSON)
}

func TestDoctorCommandRunsRepositoryCheckWhenRepoProvided(t *testing.T) {
	origAuth := runDoctorSetupAuth
	origRepo := runDoctorSetupRepositoryCheck
	t.Cleanup(func() {
		runDoctorSetupAuth = origAuth
		runDoctorSetupRepositoryCheck = origRepo
	})

	var got SetupRepositoryCheckOptions
	runDoctorSetupAuth = func(opts SetupAuthOptions) error {
		return errors.New("unexpected auth-only check")
	}
	runDoctorSetupRepositoryCheck = func(opts SetupRepositoryCheckOptions) error {
		got = opts
		return nil
	}

	root := &cobra.Command{Use: "aw"}
	var verbose bool
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing detailed information")
	root.AddCommand(NewDoctorCommand())
	root.SetArgs([]string{"doctor", "--repo", "github/gh-aw", "--dir", ".", "--require-owner-type", "org", "--json", "--verbose"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "github/gh-aw", got.Repo)
	assert.Equal(t, ".", got.Dir)
	assert.Equal(t, "org", got.RequireOwnerType)
	assert.True(t, got.JSON)
	assert.True(t, got.Verbose)
}

func TestDoctorCommandUsesDefaultRequireOwnerTypeWhenOmitted(t *testing.T) {
	origAuth := runDoctorSetupAuth
	origRepo := runDoctorSetupRepositoryCheck
	t.Cleanup(func() {
		runDoctorSetupAuth = origAuth
		runDoctorSetupRepositoryCheck = origRepo
	})

	var got SetupRepositoryCheckOptions
	runDoctorSetupAuth = func(opts SetupAuthOptions) error {
		return errors.New("unexpected auth-only check")
	}
	runDoctorSetupRepositoryCheck = func(opts SetupRepositoryCheckOptions) error {
		got = opts
		return nil
	}

	root := &cobra.Command{Use: "aw"}
	root.AddCommand(NewDoctorCommand())
	root.SetArgs([]string{"doctor", "--repo", "github/gh-aw"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "any", got.RequireOwnerType)
}

func TestDoctorCommandRejectsRepoOnlyFlagsWithoutRepo(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "dir only", args: []string{"doctor", "--dir", "."}},
		{name: "require-owner-type only", args: []string{"doctor", "--require-owner-type", "org"}},
		{name: "dir and require-owner-type", args: []string{"doctor", "--dir", ".", "--require-owner-type", "org"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origAuth := runDoctorSetupAuth
			origRepo := runDoctorSetupRepositoryCheck
			t.Cleanup(func() {
				runDoctorSetupAuth = origAuth
				runDoctorSetupRepositoryCheck = origRepo
			})

			runDoctorSetupAuth = func(opts SetupAuthOptions) error {
				return errors.New("should not run auth path with invalid flag combination")
			}
			runDoctorSetupRepositoryCheck = func(opts SetupRepositoryCheckOptions) error {
				return errors.New("should not run repo path without --repo")
			}

			root := &cobra.Command{Use: "aw"}
			var verbose bool
			root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing detailed information")
			root.AddCommand(NewDoctorCommand())
			root.SetArgs(tc.args)

			err := root.Execute()
			require.Error(t, err)
			assert.Equal(t, "--dir and --require-owner-type require --repo", err.Error())
		})
	}
}

func TestDoctorCommandAllowsVerboseWithoutRepo(t *testing.T) {
	origAuth := runDoctorSetupAuth
	origRepo := runDoctorSetupRepositoryCheck
	t.Cleanup(func() {
		runDoctorSetupAuth = origAuth
		runDoctorSetupRepositoryCheck = origRepo
	})

	authCalled := false
	runDoctorSetupAuth = func(opts SetupAuthOptions) error {
		authCalled = true
		return nil
	}
	runDoctorSetupRepositoryCheck = func(opts SetupRepositoryCheckOptions) error {
		return errors.New("should not run repo path without --repo")
	}

	root := &cobra.Command{Use: "aw"}
	var verbose bool
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing detailed information")
	root.AddCommand(NewDoctorCommand())
	root.SetArgs([]string{"doctor", "--verbose"})

	require.NoError(t, root.Execute())
	assert.True(t, authCalled)
	assert.True(t, verbose)
}
