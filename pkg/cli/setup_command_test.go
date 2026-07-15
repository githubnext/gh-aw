//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSetupCommand(t *testing.T) {
	cmd := NewSetupCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "setup", cmd.Use)
	assert.Equal(t, "Run reusable auth and repository setup checks", cmd.Short)
	assert.True(t, cmd.Hidden)
	assert.Contains(t, cmd.Long, "Available subcommands:")
	assert.Contains(t, cmd.Example, "gh aw setup repo --repo github/gh-aw --json")
	assert.Contains(t, cmd.Long, "- auth")
	assert.Contains(t, cmd.Long, "- repo")
	assert.True(t, cmd.HasSubCommands())

	var hasAuth, hasRepo bool
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "auth" {
			hasAuth = true
			assert.NotNil(t, subcmd.Flags().Lookup("json"), "auth subcommand should expose --json")
		}
		if subcmd.Name() == "repo" {
			hasRepo = true
			assert.NotNil(t, subcmd.Flags().Lookup("json"), "repo subcommand should expose --json")
		}
	}
	assert.True(t, hasAuth, "should have auth subcommand")
	assert.True(t, hasRepo, "should have repo subcommand")
}

func TestSetupSubcommandsAdvertiseJSONExamples(t *testing.T) {
	authCmd := newSetupAuthSubcommand()
	repoCmd := newSetupRepoSubcommand()

	assert.Contains(t, authCmd.Example, "gh aw setup auth --json")
	assert.Contains(t, authCmd.Long, "setup-oriented commands.")
	assert.Contains(t, repoCmd.Example, "gh aw setup repo --repo github/gh-aw --json")
	assert.NotContains(t, repoCmd.Example, "\t")
}

func TestNewSetupCommandHelp(t *testing.T) {
	cmd := NewSetupCommand()
	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)
}

func TestSetupCommandUnknownSubcommandReturnsError(t *testing.T) {
	cmd := NewSetupCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"unknown-cmd"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, `unknown command "unknown-cmd" for "setup"`, err.Error())
}

func TestNewSetupRepoSubcommandRequiresRepoFlagOnExecute(t *testing.T) {
	cmd := newSetupRepoSubcommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, "required flag(s) \"repo\" not set", err.Error())
}

func TestRunSetupAuthWithRuntime(t *testing.T) {
	called := 0
	err := runSetupAuthWithRuntime(SetupAuthOptions{Ctx: context.Background()}, setupRepositoryRuntime{
		checkAuth: func(context.Context) error {
			called++
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, called)
}

func TestRunSetupAuthWithRuntime_JSONOutput(t *testing.T) {
	output := captureSetupStdout(t, func() error {
		return runSetupAuthWithRuntime(SetupAuthOptions{Ctx: context.Background(), JSON: true}, setupRepositoryRuntime{
			checkAuth: func(context.Context) error { return nil },
		})
	})

	var result SetupAuthResult
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.True(t, result.Authenticated)
}

func TestRunSetupRepositoryCheck_AttachedCheckout(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  repoDir,
	}), setupRepositoryRuntime{
		checkAuth:          func(context.Context) error { return nil },
		ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
		repoExists:         func(context.Context, string) (bool, error) { return true, nil },
		dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
		checkCleanWorktree: func(bool) error { return nil },
	})
	require.NoError(t, err)
}

func TestRunSetupRepositoryCheck_JSONOutput(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	output := captureSetupStdout(t, func() error {
		return runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
			Ctx:              context.Background(),
			Repo:             "octo/platform-ops",
			Dir:              repoDir,
			RequireOwnerType: "org",
			JSON:             true,
		}), setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		})
	})

	var result SetupRepositoryCheckResult
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, "octo/platform-ops", result.Repository)
	assert.Equal(t, repoDir, result.Directory)
	assert.True(t, result.Authenticated)
	assert.True(t, result.RepositoryExists)
	assert.Equal(t, "org", result.OwnerType)
	assert.Equal(t, "org", result.RequiredOwnerType)
	assert.True(t, result.CheckoutAttached)
	assert.False(t, result.CloneNeeded)
	require.NotNil(t, result.CleanWorktree)
	assert.True(t, *result.CleanWorktree)
}

func TestRunSetupRepositoryCheck_EnforcesOwnerTypeRequirement(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:              context.Background(),
		Repo:             "octo/platform-ops",
		Dir:              repoDir,
		RequireOwnerType: "user",
	}), setupRepositoryRuntime{
		checkAuth:          func(context.Context) error { return nil },
		ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
		repoExists:         func(context.Context, string) (bool, error) { return true, nil },
		dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
		checkCleanWorktree: func(bool) error { return nil },
	})
	require.Error(t, err)
	assert.Equal(t, "owner octo is org, but --require-owner-type=user was requested", err.Error())
}

func TestRunSetupRepositoryCheck_RequiresExistingRepository(t *testing.T) {
	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
	}), setupRepositoryRuntime{
		checkAuth:  func(context.Context) error { return nil },
		ownerType:  func(context.Context, string) (string, error) { return "Organization", nil },
		repoExists: func(context.Context, string) (bool, error) { return false, nil },
	})
	require.Error(t, err)
	assert.Equal(t, "repository octo/platform-ops does not exist", err.Error())
}

func TestRunSetupRepositoryCheck_PropagatesCleanWorktreeError(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	wantErr := errors.New("working directory has uncommitted changes, please commit or stash them first")

	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  repoDir,
	}), setupRepositoryRuntime{
		checkAuth: func(context.Context) error { return nil },
		ownerType: func(context.Context, string) (string, error) {
			return "Organization", nil
		},
		repoExists:    func(context.Context, string) (bool, error) { return true, nil },
		dirOriginRepo: func(string) (string, error) { return "octo/platform-ops", nil },
		checkCleanWorktree: func(bool) error {
			return wantErr
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestRunSetupRepositoryCheck_RejectsNonExistentNestedCheckoutPath(t *testing.T) {
	parentRepoDir := initBootstrapGitRepo(t)
	nestedDir := filepath.Join(parentRepoDir, "new-checkout")

	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  nestedDir,
	}), setupRepositoryRuntime{
		checkAuth:  func(context.Context) error { return nil },
		ownerType:  func(context.Context, string) (string, error) { return "Organization", nil },
		repoExists: func(context.Context, string) (bool, error) { return true, nil },
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is inside a different git checkout rooted at")
	assert.Contains(t, err.Error(), parentRepoDir)
}

func TestCreateSetupRepository_UsesSupportedFlags(t *testing.T) {
	fakeBin := t.TempDir()
	argsLog := filepath.Join(fakeBin, "gh-args.log")
	fakeGH := filepath.Join(fakeBin, "gh")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLog + "\"\n" +
		"if [ \"$1\" = \"repo\" ] && [ \"$2\" = \"create\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	require.NoError(t, createSetupRepository(context.Background(), "octo/platform-ops", "private"))

	logData, err := os.ReadFile(argsLog)
	require.NoError(t, err)
	logText := string(logData)
	assert.Contains(t, logText, "repo create octo/platform-ops --private")
	assert.NotContains(t, logText, "--confirm")
	assert.NotContains(t, logText, "--clone=false")
}

func TestCheckSetupRepositoryOwnerType_FallsBackToOrgsEndpoint(t *testing.T) {
	fakeBin := t.TempDir()
	fakeGH := filepath.Join(fakeBin, "gh")
	script := `#!/bin/sh
if [ "$1" = "api" ] && [ "$2" = "users/octo" ]; then
  echo "Not Found" >&2
  exit 1
fi
if [ "$1" = "api" ] && [ "$2" = "orgs/octo" ]; then
  echo "Organization"
  exit 0
fi
exit 1
`
	require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	ownerType, err := checkSetupRepositoryOwnerType(context.Background(), "octo")
	require.NoError(t, err)
	assert.Equal(t, "Organization", ownerType)
}

func TestRunSetupRepositoryCheck_AcceptsCaseInsensitiveSlugMatch(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	err := runSetupRepositoryCheckWithRuntime(normalizeSetupRepositoryCheckOptions(SetupRepositoryCheckOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  repoDir,
	}), setupRepositoryRuntime{
		checkAuth:          func(context.Context) error { return nil },
		ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
		repoExists:         func(context.Context, string) (bool, error) { return true, nil },
		dirOriginRepo:      func(string) (string, error) { return "Octo/Platform-Ops", nil },
		checkCleanWorktree: func(bool) error { return nil },
	})
	require.NoError(t, err)
}

func TestValidateSetupRepositoryCheckOptions_RejectsEmptyRepoComponents(t *testing.T) {
	tests := []SetupRepositoryCheckOptions{
		{Repo: "/repo"},
		{Repo: "owner/"},
	}

	for _, tt := range tests {
		if err := validateSetupRepositoryCheckOptions(tt); err == nil {
			t.Fatalf("expected invalid repo slug error for %q", tt.Repo)
		}
	}
}

func TestSetupCommandSubcommandListingsUseHyphenBullets(t *testing.T) {
	tests := []struct {
		name    string
		longDoc string
	}{
		{name: "setup", longDoc: NewSetupCommand().Long},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.longDoc, "Available subcommands:")
			assert.NotContains(t, tt.longDoc, "  • ")
		})
	}
}

func TestSetupRepoSubcommandUsesNoArgs(t *testing.T) {
	cmd := newSetupRepoSubcommand()
	require.NotNil(t, cmd.Args)
	require.NoError(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
}

func TestSetupAuthSubcommandUsesNoArgs(t *testing.T) {
	cmd := newSetupAuthSubcommand()
	require.NotNil(t, cmd.Args)
	require.NoError(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
}

func TestSetupCommandStructure(t *testing.T) {
	tests := []struct {
		name           string
		expectedUse    string
		commandCreator func() any
	}{
		{
			name:        "setup command exists",
			expectedUse: "setup",
			commandCreator: func() any {
				return NewSetupCommand()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.commandCreator()
			require.NotNil(t, cmd)
		})
	}
}

func captureSetupStdout(t *testing.T, fn func() error) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := fn()
	require.NoError(t, runErr)

	require.NoError(t, w.Close())
	os.Stdout = oldStdout
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}
