package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var autoUpdateWorkflowLog = logger.New("workflow:auto_update_workflow")

// AutoUpdateWorkflowFileName is the filename for the generated auto-upgrade workflow.
const AutoUpdateWorkflowFileName = "agentic-auto-upgrade.yml"

// autoUpdateWorkflowIdentifier is the stable identifier used to scatter the
// FUZZY:WEEKLY cron schedule. It is combined with the repo slug to ensure
// that different repositories scatter to different time slots.
const autoUpdateWorkflowIdentifier = "agentic-auto-upgrade"

// GenerateAutoUpdateWorkflowOptions configures an auto-update workflow generation run.
type GenerateAutoUpdateWorkflowOptions struct {
	// Context is used for action reference resolution in non-dev modes.
	// When nil, context.Background() is used.
	Context context.Context
	// WorkflowDir is the directory where the workflow file will be written.
	WorkflowDir string
	// Enabled indicates whether auto-updates are enabled in the repo config.
	Enabled bool
	// RepoSlug is the owner/repo slug used to deterministically scatter the
	// weekly cron schedule across different repositories. Pass an empty string
	// when the slug is not available; scattering will still succeed using only
	// the workflow identifier as seed.
	RepoSlug string
	// SetupActionRef is the resolved reference for the gh-aw actions/setup action.
	// For example: "./actions/setup" (dev mode) or "github/gh-aw/actions/setup@<sha>" (release mode).
	// When empty, "./actions/setup" is used as a fallback.
	SetupActionRef string
	// GitHubScriptPin is the pinned reference for actions/github-script.
	// When empty, getActionPin("actions/github-script") is used as a fallback.
	GitHubScriptPin string
	// ActionMode controls how CLI install steps and command prefixes are generated.
	// Defaults to ActionModeDev when empty.
	ActionMode ActionMode
	// Version is the gh-aw version used by generateInstallCLISteps in non-dev modes.
	Version string
	// ActionTag optionally overrides the setup-cli version tag in non-dev modes.
	ActionTag string
	// Resolver optionally resolves setup-cli action tags to SHA-pinned refs.
	Resolver SHAResolver
}

// GenerateAutoUpdateWorkflow generates or removes the agentic-auto-upgrade.yml workflow
// based on whether auto-updates are enabled in the repository's aw.json.
//
// When enabled, it generates a workflow that runs on a fuzzy weekly schedule
// and inlines the upgrade operation JavaScript to check for and report available
// workflow upgrades via a GitHub issue.
//
// When disabled (or when maintenance is disabled), any existing agentic-auto-upgrade.yml
// is deleted.
func GenerateAutoUpdateWorkflow(opts GenerateAutoUpdateWorkflowOptions) error {
	outputFile := filepath.Join(opts.WorkflowDir, AutoUpdateWorkflowFileName)

	if !opts.Enabled {
		autoUpdateWorkflowLog.Print("Auto-updates not enabled, removing agentic-auto-upgrade.yml if present")
		if _, err := os.Stat(outputFile); err == nil {
			autoUpdateWorkflowLog.Printf("Deleting existing auto-upgrade workflow: %s", outputFile)
			if err := os.Remove(outputFile); err != nil {
				return fmt.Errorf("failed to delete auto-upgrade workflow: %w", err)
			}
			autoUpdateWorkflowLog.Print("Auto-upgrade workflow deleted successfully")
		}
		return nil
	}

	seed := buildAutoUpdateSeed(opts.RepoSlug)
	cronSchedule, err := parser.ScatterSchedule("FUZZY:WEEKLY", seed)
	if err != nil {
		return fmt.Errorf("failed to scatter FUZZY:WEEKLY schedule for auto-update workflow: %w", err)
	}
	autoUpdateWorkflowLog.Printf("Scattered FUZZY:WEEKLY to %q for seed %q", cronSchedule, seed)

	setupActionRef := opts.SetupActionRef
	if setupActionRef == "" {
		setupActionRef = "./actions/setup"
	}
	githubScriptPin := opts.GitHubScriptPin
	if githubScriptPin == "" {
		githubScriptPin = getActionPin("actions/github-script")
	}

	actionMode := opts.ActionMode
	if actionMode == "" {
		actionMode = ActionModeDev
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	content := buildAutoUpdateWorkflowYAML(
		cronSchedule,
		setupActionRef,
		githubScriptPin,
		generateInstallCLISteps(ctx, actionMode, opts.Version, opts.ActionTag, opts.Resolver),
		getCLICmdPrefix(actionMode),
	)

	autoUpdateWorkflowLog.Printf("Writing auto-update workflow to %s", outputFile)
	if err := fileutil.EnsureParentDir(outputFile, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create auto-update workflow directory: %w", err)
	}
	if err := os.WriteFile(outputFile, []byte(content), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write auto-update workflow: %w", err)
	}

	autoUpdateWorkflowLog.Print("Auto-update workflow generated successfully")
	return nil
}

// buildAutoUpdateSeed returns the deterministic seed string used to scatter the
// FUZZY:WEEKLY cron schedule. It combines the repo slug with the fixed workflow
// identifier so that repositories scatter to distinct time slots.
func buildAutoUpdateSeed(repoSlug string) string {
	if repoSlug != "" {
		return repoSlug + "/" + autoUpdateWorkflowIdentifier
	}
	return autoUpdateWorkflowIdentifier
}

// buildAutoUpdateWorkflowYAML generates the YAML content for agentic-auto-upgrade.yml.
func buildAutoUpdateWorkflowYAML(
	cronSchedule, setupActionRef, githubScriptPin, installCLISteps, cliCmdPrefix string,
) string {
	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

The workflow is generated when auto_upgrade is set to true in aw.json.
The weekly schedule is deterministically scattered based on the repository slug.`

	header := GenerateWorkflowHeader("", "pkg/workflow/auto_update_workflow.go", customInstructions)

	return header + `name: Agentic Auto-Upgrade

on:
  schedule:
    - cron: "` + cronSchedule + `"  # Weekly (auto-upgrade)
  workflow_dispatch:

permissions:
  contents: read
  issues: write

jobs:
  auto-upgrade:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `

` + installCLISteps + `      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Run upgrade notification
        uses: ` + githubScriptPin + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_OPERATION: upgrade
          GH_AW_CMD_PREFIX: ` + cliCmdPrefix + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { mainNotifyIssue } = require('${{ runner.temp }}/gh-aw/actions/run_operation_update_upgrade.cjs');
            await mainNotifyIssue();
`
}
