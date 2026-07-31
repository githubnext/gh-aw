package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResumeCommand(t *testing.T) {
	cmd := NewResumeCommand()

	assert.Equal(t, "resume <run-id-or-url>", cmd.Use)
	assert.Equal(t, "Resume a local agent session from a workflow run", cmd.Short)
	require.NotNil(t, cmd.Flags().Lookup("dir"))
	require.NotNil(t, cmd.Flags().Lookup("repo"))
	require.NotNil(t, cmd.Flags().Lookup("verbose"))
	require.NoError(t, cmd.Args(cmd, []string{"123"}))
	require.Error(t, cmd.Args(cmd, nil))
	require.Error(t, cmd.Args(cmd, []string{"123", "456"}))
}

func TestFindResumeSessionID(t *testing.T) {
	t.Run("rejects missing sessions", func(t *testing.T) {
		_, err := findResumeSessionID(t.TempDir())

		require.EqualError(t, err, "workflow run does not contain a resumable session")
	})

	t.Run("returns only session", func(t *testing.T) {
		sessionStateDir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(sessionStateDir, "session-123"), constants.DirPermSensitive))

		sessionID, err := findResumeSessionID(sessionStateDir)

		require.NoError(t, err)
		assert.Equal(t, "session-123", sessionID)
	})

	t.Run("rejects multiple sessions", func(t *testing.T) {
		sessionStateDir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(sessionStateDir, "session-b"), constants.DirPermSensitive))
		require.NoError(t, os.Mkdir(filepath.Join(sessionStateDir, "session-a"), constants.DirPermSensitive))

		_, err := findResumeSessionID(sessionStateDir)

		require.EqualError(t, err, "workflow run contains multiple sessions ([session-a session-b]); unable to select one automatically")
	})
}

func TestCopyResumeSession(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "session")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "checkpoints"), constants.DirPermSensitive))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "session.db"), []byte("session"), constants.FilePermSensitive))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "checkpoints", "one.json"), []byte("{}"), constants.FilePermSensitive))

	require.NoError(t, copyResumeSession(sourceDir, targetDir))

	sessionData, err := os.ReadFile(filepath.Join(targetDir, "session.db"))
	require.NoError(t, err)
	assert.Equal(t, "session", string(sessionData))
	assert.FileExists(t, filepath.Join(targetDir, "checkpoints", "one.json"))
}

func TestRunResumeCommand(t *testing.T) {
	originalDownload := resumeDownloadRunArtifacts
	originalLookPath := resumeLookPath
	originalCommandContext := resumeCommandContext
	originalRunCommand := resumeRunCommand
	t.Cleanup(func() {
		resumeDownloadRunArtifacts = originalDownload
		resumeLookPath = originalLookPath
		resumeCommandContext = originalCommandContext
		resumeRunCommand = originalRunCommand
	})

	outputDir := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	copilotHome := filepath.Join(stateHome, "gh-aw", "resume", "run-123", "copilot-home")

	resumeDownloadRunArtifacts = func(_ context.Context, opts downloadArtifactsOptions) error {
		require.Equal(t, int64(123), opts.runID)
		require.Equal(t, []string{"activation", "agent"}, opts.artifactFilter)
		require.NoError(t, os.MkdirAll(
			filepath.Join(opts.outputDir, "sandbox", "agent", "logs", "copilot-session-state", "session-123"),
			constants.DirPermSensitive,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(opts.outputDir, "aw_info.json"),
			[]byte(`{"engine_id":"copilot"}`),
			constants.FilePermSensitive,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(opts.outputDir, "sandbox", "agent", "logs", "copilot-session-state", "session-123", "session.db"),
			[]byte("session"),
			constants.FilePermSensitive,
		))
		return nil
	}
	resumeLookPath = func(file string) (string, error) {
		require.Equal(t, "copilot", file)
		return os.Args[0], nil
	}
	resumeCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		require.Equal(t, os.Args[0], name)
		require.Equal(t, []string{"--resume=session-123"}, args)
		return exec.CommandContext(ctx, name, args...)
	}
	resumeRunCommand = func(cmd *exec.Cmd) error {
		assert.Equal(t, os.Stdin, cmd.Stdin)
		assert.Equal(t, os.Stdout, cmd.Stdout)
		assert.Equal(t, os.Stderr, cmd.Stderr)
		assert.Contains(t, cmd.Env, "COPILOT_HOME="+copilotHome)
		return nil
	}

	err := runResumeCommand(context.Background(), "123", resumeCommandOptions{outputDir: outputDir})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(copilotHome, "session-state", "session-123", "session.db"))
}

func TestRunResumeCommandRejectsNonCopilotEngine(t *testing.T) {
	originalDownload := resumeDownloadRunArtifacts
	t.Cleanup(func() {
		resumeDownloadRunArtifacts = originalDownload
	})

	outputDir := t.TempDir()
	resumeDownloadRunArtifacts = func(_ context.Context, opts downloadArtifactsOptions) error {
		require.NoError(t, os.MkdirAll(opts.outputDir, constants.DirPermSensitive))
		return os.WriteFile(
			filepath.Join(opts.outputDir, "aw_info.json"),
			[]byte(`{"engine_id":"codex"}`),
			constants.FilePermSensitive,
		)
	}

	err := runResumeCommand(context.Background(), "123", resumeCommandOptions{outputDir: outputDir})

	require.EqualError(t, err, `resume currently supports a limited set of engines; run 123 used engine "codex"`)
}

func TestWithResumeHomeEnv(t *testing.T) {
	environment := withResumeHomeEnv(
		[]string{"PATH=/bin", "COPILOT_HOME=/old/home"},
		"COPILOT_HOME",
		"/isolated/home",
	)

	assert.Equal(t, []string{"PATH=/bin", "COPILOT_HOME=/isolated/home"}, environment)
}

func TestRunResumeCommandClaudeEngine(t *testing.T) {
	originalDownload := resumeDownloadRunArtifacts
	originalLookPath := resumeLookPath
	originalCommandContext := resumeCommandContext
	originalRunCommand := resumeRunCommand
	t.Cleanup(func() {
		resumeDownloadRunArtifacts = originalDownload
		resumeLookPath = originalLookPath
		resumeCommandContext = originalCommandContext
		resumeRunCommand = originalRunCommand
	})

	outputDir := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	claudeHome := filepath.Join(stateHome, "gh-aw", "resume", "run-123", "claude-home")

	resumeDownloadRunArtifacts = func(_ context.Context, opts downloadArtifactsOptions) error {
		require.Equal(t, []string{"activation", "agent"}, opts.artifactFilter)
		require.NoError(t, os.MkdirAll(
			filepath.Join(opts.outputDir, "sandbox", "agent", "logs", "claude-session-state", "session-123"),
			constants.DirPermSensitive,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(opts.outputDir, "aw_info.json"),
			[]byte(`{"engine_id":"claude"}`),
			constants.FilePermSensitive,
		))
		return os.WriteFile(
			filepath.Join(opts.outputDir, "sandbox", "agent", "logs", "claude-session-state", "session-123", "state.json"),
			[]byte("{}"),
			constants.FilePermSensitive,
		)
	}
	resumeLookPath = func(file string) (string, error) {
		require.Equal(t, "claude", file)
		return os.Args[0], nil
	}
	resumeCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		require.Equal(t, os.Args[0], name)
		require.Equal(t, []string{"--continue"}, args)
		return exec.CommandContext(ctx, name, args...)
	}
	resumeRunCommand = func(cmd *exec.Cmd) error {
		assert.Contains(t, cmd.Env, "HOME="+claudeHome)
		return nil
	}

	err := runResumeCommand(context.Background(), "123", resumeCommandOptions{outputDir: outputDir})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(claudeHome, ".claude", "projects", "session-123", "state.json"))
}
