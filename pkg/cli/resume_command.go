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

type resumeEngineDefinition struct {
	id                 string
	displayName        string
	commandName        string
	commandArgs        func(sessionID string) []string
	sessionStateDir    string
	homeDirName        string
	sessionTargetDir   string
	homeEnvironmentKey string
}

// NewResumeCommand creates the resume command.
func NewResumeCommand() *cobra.Command {
	opts := resumeCommandOptions{}
	cmd := &cobra.Command{
		Use:   "resume <run-id-or-url>",
		Short: "Resume a local agent session from a workflow run",
		Long: `Download the activation and agent artifacts for a GitHub Actions workflow run,
restore saved session files, and launch the matching local CLI continuation command.

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
	engine, err := resolveResumeEngine(awInfo.EngineID)
	if err != nil {
		return fmt.Errorf("resume currently supports a limited set of engines; run %d used engine %q", components.Number, awInfo.EngineID)
	}

	sessionSourceDir := filepath.Join(runDir, "sandbox", "agent", "logs", engine.sessionStateDir)
	sessionID, err := findResumeSessionID(sessionSourceDir)
	if err != nil {
		return err
	}
	engineCommandPath, err := resumeLookPath(engine.commandName)
	if err != nil {
		return fmt.Errorf("%s CLI is required to resume this run; install it and ensure %q is on PATH", engine.displayName, engine.commandName)
	}
	engineHome, err := resolveResumeStateHome(components.Number, engine.homeDirName)
	if err != nil {
		return err
	}
	sessionTargetDir := filepath.Join(engineHome, engine.sessionTargetDir, sessionID)
	if !fileutil.DirExists(sessionTargetDir) {
		if err := copyResumeSession(filepath.Join(sessionSourceDir, sessionID), sessionTargetDir); err != nil {
			return fmt.Errorf("failed to restore %s session %s: %w", engine.displayName, sessionID, err)
		}
	} else {
		resumeCommandLog.Printf("Using existing local %s session: %s", engine.displayName, sessionTargetDir)
	}

	resumeCommandLog.Printf("Launching %s CLI for session %s from run %d", engine.displayName, sessionID, components.Number)
	engineCmd := resumeCommandContext(ctx, engineCommandPath, engine.commandArgs(sessionID)...)
	engineCmd.Stdin = os.Stdin
	engineCmd.Stdout = os.Stdout
	engineCmd.Stderr = os.Stderr
	engineCmd.Env = withResumeHomeEnv(os.Environ(), engine.homeEnvironmentKey, engineHome)
	if err := resumeRunCommand(engineCmd); err != nil {
		return fmt.Errorf("%s CLI exited with an error: %w", engine.displayName, err)
	}
	return nil
}

func resolveResumeEngine(engineID string) (resumeEngineDefinition, error) {
	switch {
	case engineID == "copilot" || strings.HasPrefix(engineID, "copilot-"):
		return resumeEngineDefinition{
			id:                 "copilot",
			displayName:        "Copilot",
			commandName:        "copilot",
			commandArgs:        func(sessionID string) []string { return []string{"--resume=" + sessionID} },
			sessionStateDir:    "copilot-session-state",
			homeDirName:        "copilot-home",
			sessionTargetDir:   "session-state",
			homeEnvironmentKey: "COPILOT_HOME",
		}, nil
	case engineID == "claude" || strings.HasPrefix(engineID, "claude-"):
		return resumeEngineDefinition{
			id:                 "claude",
			displayName:        "Claude",
			commandName:        "claude",
			commandArgs:        func(string) []string { return []string{"--continue"} },
			sessionStateDir:    "claude-session-state",
			homeDirName:        "claude-home",
			sessionTargetDir:   filepath.Join(".claude", "projects"),
			homeEnvironmentKey: "HOME",
		}, nil
	default:
		return resumeEngineDefinition{}, fmt.Errorf("unsupported engine: %s", engineID)
	}
}

func resolveResumeStateHome(runID int64, homeDirName string) (string, error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to locate home directory: %w", err)
		}
		stateHome = filepath.Join(homeDir, ".local", "state")
	}
	return filepath.Join(stateHome, "gh-aw", "resume", fmt.Sprintf("run-%d", runID), homeDirName), nil
}

func withResumeHomeEnv(environment []string, key string, value string) []string {
	filtered := make([]string, 0, len(environment)+1)
	keyPrefix := key + "="
	for _, entry := range environment {
		if strings.HasPrefix(entry, keyPrefix) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, keyPrefix+value)
}

func findResumeSessionID(sessionStateDir string) (string, error) {
	entries, err := os.ReadDir(sessionStateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("workflow run does not contain session state")
		}
		return "", fmt.Errorf("failed to read session state: %w", err)
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
		return "", errors.New("workflow run does not contain a resumable session")
	case 1:
		return sessionIDs[0], nil
	default:
		return "", fmt.Errorf("workflow run contains multiple sessions (%v); unable to select one automatically", sessionIDs)
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
