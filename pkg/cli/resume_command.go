package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var resumeCommandLog = logger.New("cli:resume_command")

var (
	resumeDownloadRunArtifacts = downloadRunArtifacts
	resumeLookPath             = exec.LookPath
	resumeCommandContext       = exec.CommandContext
	resumeRunCommand           = func(cmd *exec.Cmd) error { return cmd.Run() }
)

type resumeCommandOptions struct {
	outputDir string
	repo      string
	verbose   bool
}

// NewResumeCommand creates the resume command.
func NewResumeCommand() *cobra.Command {
	opts := resumeCommandOptions{}
	cmd := &cobra.Command{
		Use:   "resume <run-id-or-url>",
		Short: "Resume a Copilot CLI session from a workflow run",
		Long: `Download the activation and agent artifacts for a GitHub Actions workflow run,
restore its Copilot CLI session files, and launch Copilot with that session.

This command restores session data only. It does not recreate the GitHub Actions job
environment, start MCP servers, or replay the workflow run.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResumeCommand(cmd.Context(), args[0], opts)
		},
	}
	cmd.Flags().StringVarP(&opts.outputDir, "dir", "d", defaultLogsOutputDir, "Directory used to cache downloaded workflow run artifacts")
	cmd.Flags().StringVarP(&opts.repo, "repo", "r", "", "Target repository (owner/repo format). Defaults to the current repository")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed progress information")
	return cmd
}

func runResumeCommand(ctx context.Context, runIDOrURL string, opts resumeCommandOptions) error {
	components, err := parser.ParseRunURLExtended(runIDOrURL)
	if err != nil {
		return err
	}
	if err := applyAuditRepoFlag(opts.repo, components); err != nil {
		return err
	}

	runDir := filepath.Join(opts.outputDir, fmt.Sprintf("run-%d", components.Number))
	if err := ensureLogsGitignoreWithWarning(opts.verbose); err != nil {
		return err
	}
	if err := resumeDownloadRunArtifacts(ctx, downloadArtifactsOptions{
		runID:          components.Number,
		outputDir:      runDir,
		verbose:        opts.verbose,
		owner:          components.Owner,
		repo:           components.Repo,
		hostname:       components.Host,
		artifactFilter: ResolveArtifactFilter([]string{string(ArtifactSetActivation), string(ArtifactSetAgent)}),
	}); err != nil {
		return err
	}

	awInfo, err := parseAwInfo(filepath.Join(runDir, "aw_info.json"), opts.verbose)
	if err != nil {
		return fmt.Errorf("workflow run does not contain readable activation metadata: %w", err)
	}
	if awInfo.EngineID != "copilot" {
		return fmt.Errorf("resume currently supports Copilot CLI runs only; run %d used engine %q", components.Number, awInfo.EngineID)
	}

	sessionSourceDir := filepath.Join(runDir, "sandbox", "agent", "logs", "copilot-session-state")
	sessionID, err := findResumeSessionID(sessionSourceDir)
	if err != nil {
		return err
	}
	copilotPath, err := resumeLookPath("copilot")
	if err != nil {
		return errors.New("copilot CLI is required to resume this run; install it and ensure 'copilot' is on PATH")
	}
	copilotHome, err := resolveResumeCopilotHome(components.Number)
	if err != nil {
		return err
	}
	sessionTargetDir := filepath.Join(copilotHome, "session-state", sessionID)
	if !fileutil.DirExists(sessionTargetDir) {
		if err := copyResumeSession(filepath.Join(sessionSourceDir, sessionID), sessionTargetDir); err != nil {
			return fmt.Errorf("failed to restore Copilot session %s: %w", sessionID, err)
		}
	} else {
		resumeCommandLog.Printf("Using existing local Copilot session: %s", sessionTargetDir)
	}

	resumeCommandLog.Printf("Launching Copilot CLI for session %s from run %d", sessionID, components.Number)
	copilotCmd := resumeCommandContext(ctx, copilotPath, "--resume="+sessionID)
	copilotCmd.Stdin = os.Stdin
	copilotCmd.Stdout = os.Stdout
	copilotCmd.Stderr = os.Stderr
	copilotCmd.Env = withResumeCopilotHome(os.Environ(), copilotHome)
	if err := resumeRunCommand(copilotCmd); err != nil {
		return fmt.Errorf("copilot CLI exited with an error: %w", err)
	}
	return nil
}

func resolveResumeCopilotHome(runID int64) (string, error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to locate home directory: %w", err)
		}
		stateHome = filepath.Join(homeDir, ".local", "state")
	}
	return filepath.Join(stateHome, "gh-aw", "resume", fmt.Sprintf("run-%d", runID), "copilot-home"), nil
}

func withResumeCopilotHome(environment []string, copilotHome string) []string {
	filtered := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if strings.HasPrefix(entry, "COPILOT_HOME=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, "COPILOT_HOME="+copilotHome)
}

func findResumeSessionID(sessionStateDir string) (string, error) {
	entries, err := os.ReadDir(sessionStateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("workflow run does not contain Copilot session state")
		}
		return "", fmt.Errorf("failed to read Copilot session state: %w", err)
	}
	var sessionIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			sessionIDs = append(sessionIDs, entry.Name())
		}
	}
	sort.Strings(sessionIDs)
	switch len(sessionIDs) {
	case 0:
		return "", errors.New("workflow run does not contain a Copilot session")
	case 1:
		return sessionIDs[0], nil
	default:
		return "", fmt.Errorf("workflow run contains multiple Copilot sessions (%v); unable to select one automatically", sessionIDs)
	}
}

func copyResumeSession(sourceDir, targetDir string) error {
	targetParent := filepath.Dir(targetDir)
	if err := os.MkdirAll(targetParent, constants.DirPermSensitive); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(targetParent, "."+filepath.Base(targetDir)+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := copyResumeSessionContents(sourceDir, tempDir); err != nil {
		return err
	}
	return os.Rename(tempDir, targetDir)
}

func copyResumeSessionContents(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, constants.DirPermSensitive)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermSensitive); err != nil {
			return err
		}
		if err := fileutil.CopyFile(path, targetPath); err != nil {
			return err
		}
		return os.Chmod(targetPath, constants.FilePermSensitive)
	})
}
