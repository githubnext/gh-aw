// This file implements the "replay" command, which downloads artifacts for a
// workflow run (reusing the helpers from audit/logs) and renders a unified
// MCP Gateway + AWF Firewall + Agent event timeline directly in the console.
//
// Usage:
//
//	gh aw replay <run-id-or-url>
//
// The output simulates the chronological activity log that would be visible
// while observing a Copilot CLI session, but is produced entirely offline
// from the downloaded artifacts.

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var replayLog = logger.New("cli:replay")

// NewReplayCommand creates the replay command.
func NewReplayCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay <run-id-or-url>",
		Short: "Render unified timeline logs for a workflow run in the console",
		Long: `Download artifacts for a workflow run and render a unified, chronologically
ordered activity timeline in the console.

The timeline merges events from three sources:
  - MCP Gateway logs  (gateway.jsonl / rpc-messages.jsonl)
  - AWF Firewall logs (audit.jsonl)
  - Agent session logs (events.jsonl)

The result simulates what you would see when watching a Copilot CLI session
live, providing a readable, complete log of all agentic activity.

The run argument accepts the same formats as the "audit" command:
  - A numeric run ID                     (e.g., 1234567890)
  - A GitHub Actions run URL             (e.g., https://github.com/owner/repo/actions/runs/1234567890)
  - A GitHub Enterprise run URL

Artifacts are downloaded to the default logs directory and cached; repeated
invocations for the same run ID will read from the local cache without
re-downloading.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` replay 1234567890
  ` + string(constants.CLIExtensionPrefix) + ` replay https://github.com/owner/repo/actions/runs/1234567890
  ` + string(constants.CLIExtensionPrefix) + ` replay 1234567890 --repo owner/repo
  ` + string(constants.CLIExtensionPrefix) + ` replay 1234567890 -o ./my-logs
  ` + string(constants.CLIExtensionPrefix) + ` replay 1234567890 -v`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			outputDir, _ := cmd.Flags().GetString("output")
			repoFlag, _ := cmd.Flags().GetString("repo")

			runIDOrURL := args[0]

			components, err := parser.ParseRunURLExtended(runIDOrURL)
			if err != nil {
				return err
			}

			// Apply --repo flag when owner/repo were not inferred from a URL.
			if repoFlag != "" && components.Owner == "" {
				parts := strings.SplitN(repoFlag, "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid repository format %q: expected 'owner/repo'", repoFlag)
				}
				components.Owner = parts[0]
				components.Repo = parts[1]
			}

			if outputDir == "" {
				outputDir = defaultLogsOutputDir
			}

			return ReplayWorkflowRun(cmd.Context(), components.Number, ReplayOptions{
				Owner:     components.Owner,
				Repo:      components.Repo,
				Hostname:  components.Host,
				OutputDir: outputDir,
				Verbose:   verbose,
			})
		},
	}

	addOutputFlag(cmd, defaultLogsOutputDir)
	addRepoFlag(cmd)
	RegisterDirFlagCompletion(cmd, "output")

	return cmd
}

// ReplayOptions holds configuration for the replay command.
type ReplayOptions struct {
	Owner     string
	Repo      string
	Hostname  string
	OutputDir string
	Verbose   bool
}

// ReplayWorkflowRun downloads artifacts for the given run (if not already cached)
// and renders the unified event timeline to stdout.
func ReplayWorkflowRun(ctx context.Context, runID int64, opts ReplayOptions) error {
	replayLog.Printf("Starting replay for run %d (owner=%s, repo=%s, hostname=%s)", runID, opts.Owner, opts.Repo, opts.Hostname)

	// Auto-detect GHES host from git remote when not explicitly provided.
	hostname := opts.Hostname
	if hostname == "" {
		hostname = getHostFromOriginRemote()
		if hostname != "github.com" {
			replayLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
		}
	}

	runDir := filepath.Join(opts.OutputDir, fmt.Sprintf("run-%d", runID))
	if absDir, err := filepath.Abs(runDir); err == nil {
		runDir = absDir
	}

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Replaying run %d...", runID)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run directory: "+runDir))
	}

	// Download artifacts when the run directory does not yet contain the JSONL
	// log files we need.  We deliberately pass a nil artifact filter so that all
	// artifacts are downloaded — the timeline relies on whichever JSONL files
	// happen to be present; no single one is strictly required.
	if err := downloadRunArtifacts(ctx, runID, runDir, opts.Verbose, opts.Owner, opts.Repo, hostname, nil); err != nil {
		if !errors.Is(err, ErrNoArtifacts) {
			return fmt.Errorf("failed to download artifacts for run %d: %w", runID, err)
		}
		// No artifacts is non-fatal: the run may still have useful events in the
		// workflow logs or the directory may have been populated by a previous run.
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No artifacts attached to this run; timeline may be empty."))
		}
	}

	// Collect and merge events from all available JSONL sources.
	events, err := BuildUnifiedTimeline(runDir, opts.Verbose)
	if err != nil {
		return fmt.Errorf("failed to build timeline for run %d: %w", runID, err)
	}

	if len(events) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No timeline events found for run %d.", runID)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Ensure the workflow has gateway.jsonl, audit.jsonl, or events.jsonl artifacts."))
		return nil
	}

	output := renderUnifiedTimelineStream(events)
	if output != "" {
		fmt.Print(output)
	}

	return nil
}
