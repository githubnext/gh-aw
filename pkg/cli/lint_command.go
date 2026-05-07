package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

var defaultGhAwActionlintIgnorePatterns = []string{
	// gh-aw extends GitHub Actions permissions with copilot-requests.
	`unknown permission scope "copilot-requests"`,
	// gh-aw exposes additional job.workflow_* context properties.
	`property "workflow_(repository|sha|ref|file_path)" is not defined in object type`,
}

// NewLintCommand creates the lint command.
func NewLintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [lock-file-or-directory]...",
		Short: "Lint existing .lock.yml workflows with actionlint only",
		Long: `Lint existing .lock.yml workflow files from disk using actionlint only.

This command does not recompile Markdown workflows and does not run zizmor or poutine.
By default, shellcheck and pyflakes integrations are disabled for generated run scripts.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` lint
  ` + string(constants.CLIExtensionPrefix) + ` lint .github/workflows/foo.lock.yml
  ` + string(constants.CLIExtensionPrefix) + ` lint --dir .github/workflows
  ` + string(constants.CLIExtensionPrefix) + ` lint --shellcheck --pyflakes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowDir, _ := cmd.Flags().GetString("dir")
			includeShellcheck, _ := cmd.Flags().GetBool("shellcheck")
			includePyflakes, _ := cmd.Flags().GetBool("pyflakes")
			verbose, _ := cmd.Flags().GetBool("verbose")

			lockFiles, err := resolveLockFilesForLint(args, workflowDir)
			if err != nil {
				return err
			}

			initActionlintStats()
			defer displayActionlintSummary()

			// Lint should fail on any actionlint error.
			strictActionlint := true
			return runActionlintOnFilesWithOptions(cmd.Context(), lockFiles, verbose, strictActionlint, actionlintRunOptions{
				IncludeShellcheck: includeShellcheck,
				IncludePyflakes:   includePyflakes,
				IgnorePatterns:    defaultGhAwActionlintIgnorePatterns,
			})
		},
	}

	cmd.Flags().StringP("dir", "d", ".github/workflows", "Directory to scan for *.lock.yml files when no arguments are provided")
	cmd.Flags().Bool("shellcheck", false, "Enable shellcheck integration in actionlint")
	cmd.Flags().Bool("pyflakes", false, "Enable pyflakes integration in actionlint")

	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

func resolveLockFilesForLint(inputs []string, workflowDir string) ([]string, error) {
	candidates := inputs
	if len(candidates) == 0 {
		candidates = []string{workflowDir}
	}

	var lockFiles []string
	for _, candidate := range candidates {
		lockFilesFromCandidate, err := expandLintCandidate(candidate)
		if err != nil {
			return nil, err
		}
		lockFiles = append(lockFiles, lockFilesFromCandidate...)
	}

	if len(lockFiles) == 0 {
		return nil, errors.New("no .lock.yml files found to lint")
	}

	slices.Sort(lockFiles)
	lockFiles = slices.Compact(lockFiles)

	return lockFiles, nil
}

func expandLintCandidate(candidate string) ([]string, error) {
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", candidate, err)
	}

	info, err := os.Stat(absCandidate)
	if err != nil {
		return nil, fmt.Errorf("failed to access %q: %w", candidate, err)
	}

	if info.IsDir() {
		lockFiles, err := filepath.Glob(filepath.Join(absCandidate, "*.lock.yml"))
		if err != nil {
			return nil, fmt.Errorf("failed to scan %q for .lock.yml files: %w", candidate, err)
		}
		return lockFiles, nil
	}

	if !strings.HasSuffix(absCandidate, ".lock.yml") {
		return nil, fmt.Errorf("path %q is not a .lock.yml file or directory", candidate)
	}

	return []string{absCandidate}, nil
}
