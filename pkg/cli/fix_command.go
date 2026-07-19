package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var fixLog = logger.New("cli:fix_command")

// FixConfig contains configuration for the fix command
type FixConfig struct {
	WorkflowIDs        []string
	Write              bool
	Verbose            bool
	WorkflowDir        string   // Custom workflow directory
	DisabledCodemodIDs []string // Codemod IDs to skip
}

// RunFix runs the fix command with the given configuration
func RunFix(config FixConfig) error {
	return runFixCommand(config.WorkflowIDs, config.Write, config.Verbose, config.WorkflowDir, config.DisabledCodemodIDs)
}

// NewFixCommand creates the fix command
func NewFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [workflow]...",
		Short: "Auto-fix deprecated agentic workflow fields using codemods (dry-run by default)",
		Long: `Apply automatic codemod-style fixes to agentic workflow files.

This command applies a registry of codemods that automatically update deprecated fields
and migrate to new syntax. Codemods preserve formatting and comments as much as possible.

Use --list-codemods to see all available codemods and their descriptions.

If no workflows are specified, all Markdown files in .github/workflows will be processed.

The command will:
  1. Scan workflow files for deprecated fields.
  2. Apply relevant codemods to fix issues.
  3. Report what was changed in each file.

Without --write (dry-run mode), no files are modified. With --write, the command performs
all steps and additionally:
  4. Write updated files back to disk.
  5. Delete deprecated .github/aw/schemas/agentic-workflow.json file if it exists.
  6. Delete old template files from previous versions if present.
  7. Delete old workflow-specific .agent.md files from .github/agents/ if present.

` + WorkflowIDExplanation,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` fix                     # Check all workflows (dry-run)
  ` + string(constants.CLIExtensionPrefix) + ` fix --write             # Fix all workflows
  ` + string(constants.CLIExtensionPrefix) + ` fix my-workflow         # Check specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` fix my-workflow --write # Fix specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` fix --dir custom/workflows # Fix workflows in custom directory
  ` + string(constants.CLIExtensionPrefix) + ` fix --list-codemods     # List available codemods`,
		RunE: func(cmd *cobra.Command, args []string) error {
			listCodemods, _ := cmd.Flags().GetBool("list-codemods")
			write, _ := cmd.Flags().GetBool("write")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dir, _ := cmd.Flags().GetString("dir")
			disabledCodemods, _ := cmd.Flags().GetStringSlice("disable-codemod")

			if listCodemods {
				return listAvailableCodemods()
			}

			return runFixCommand(args, write, verbose, dir, disabledCodemods)
		},
	}

	cmd.Flags().Bool("write", false, "Write changes to files (without this flag, no changes are made)")
	cmd.Flags().Bool("list-codemods", false, "List all available codemods and exit")
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows)")
	cmd.Flags().StringSlice("disable-codemod", nil, "Disable specific codemod IDs during the fix step (repeatable)")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// listAvailableCodemods lists all available codemods
func listAvailableCodemods() error {
	codemods := GetAllCodemods()

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Available Codemods:"))
	fmt.Fprintln(os.Stderr, "")

	for _, codemod := range codemods {
		fmt.Fprintf(os.Stderr, "  %s\n", console.FormatInfoMessage(codemod.Name))
		fmt.Fprintf(os.Stderr, "    ID: %s\n", codemod.ID)
		if codemod.IntroducedIn != "" {
			fmt.Fprintf(os.Stderr, "    Introduced in: %s\n", codemod.IntroducedIn)
		}
		fmt.Fprintf(os.Stderr, "    %s\n", codemod.Description)
		fmt.Fprintln(os.Stderr, "")
	}

	return nil
}

// runFixCommand runs the fix command on specified or all workflows
func runFixCommand(workflowIDs []string, write bool, verbose bool, workflowDir string, disabledCodemodIDs []string) error {
	fixLog.Printf("Running fix command: workflowIDs=%v, write=%v, verbose=%v, workflowDir=%s, disabledCodemodIDs=%v", workflowIDs, write, verbose, workflowDir, disabledCodemodIDs)

	// Get workflow files to process
	files, err := runFixCommandWorkflowFiles(workflowIDs, verbose, workflowDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflow files found."))
		return nil
	}

	// Load all codemods
	codemods, err := GetCodemods(disabledCodemodIDs)
	if err != nil {
		return err
	}
	fixLog.Printf("Loaded %d codemods", len(codemods))

	// Process each file
	stats := runFixCommandProcessFiles(files, codemods, write, verbose)

	runFixCommandUpdateSupportFiles(write, verbose)
	runFixCommandDeleteDeprecatedSchema(write, verbose)
	runFixCommandPrintSummary(stats, write)

	if stats.totalGuidedErrors > 0 {
		pluralSuffix := "file needs"
		if stats.totalGuidedErrors > 1 {
			pluralSuffix = "files need"
		}
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatErrorMessage(fmt.Sprintf("%d %s a manual fix", stats.totalGuidedErrors, pluralSuffix)))
		return &ExitCodeError{Code: 2}
	}

	if stats.totalProcessingErrors > 0 {
		return &ExitCodeError{Code: 1}
	}

	return nil
}

type runFixCommandStats struct {
	totalFixed            int
	totalFiles            int
	totalGuidedErrors     int
	totalProcessingErrors int
	workflowsNeedingFixes []workflowFixInfo
}

func runFixCommandWorkflowFiles(workflowIDs []string, verbose bool, workflowDir string) ([]string, error) {
	if workflowDir == "" {
		workflowDir = constants.GetWorkflowDir()
		fixLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		workflowDir = filepath.Clean(workflowDir)
		fixLog.Printf("Using custom workflow directory: %s", workflowDir)
	}
	if len(workflowIDs) == 0 {
		return getMarkdownWorkflowFiles(workflowDir)
	}
	var files []string
	for _, workflowID := range workflowIDs {
		file, err := resolveWorkflowFileInDir(workflowID, verbose, workflowDir)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func runFixCommandProcessFiles(files []string, codemods []Codemod, write bool, verbose bool) runFixCommandStats {
	var stats runFixCommandStats
	for _, file := range files {
		fixLog.Printf("Processing file: %s", file)
		fixed, appliedFixes, err := processWorkflowFileWithInfo(file, codemods, write, verbose)
		if err != nil {
			runFixCommandRecordError(&stats, file, err)
			continue
		}
		stats.totalFiles++
		if fixed {
			stats.totalFixed++
			if !write {
				stats.workflowsNeedingFixes = append(stats.workflowsNeedingFixes, workflowFixInfo{File: filepath.Base(file), Fixes: appliedFixes})
			}
		}
	}
	return stats
}

func runFixCommandRecordError(stats *runFixCommandStats, file string, err error) {
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatErrorMessage(fmt.Sprintf("Error processing %s: %v", filepath.Base(file), err)))
	var guidedErr *GuidedError
	if errors.As(err, &guidedErr) {
		stats.totalGuidedErrors++
		stats.totalFiles++
	} else {
		stats.totalProcessingErrors++
	}
}

func runFixCommandUpdateSupportFiles(write bool, verbose bool) {
	// Update prompt and skill files (similar to init command)
	// This ensures the latest templates are always used
	fixLog.Print("Updating prompt and skill files")
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		fixLog.Printf("Failed to update dispatcher skill: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update dispatcher skill: %v", err)))
	}
	if err := ensureAgenticWorkflowsAgent(verbose); err != nil {
		fixLog.Printf("Failed to update agentic workflows custom agent: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to update agentic workflows custom agent: %v", err)))
	}
	if write {
		runFixCommandCleanupGeneratedFiles(verbose)
	}
}

func runFixCommandCleanupGeneratedFiles(verbose bool) {
	fixLog.Print("Cleaning up old template files")
	if err := deleteOldTemplateFiles(verbose); err != nil {
		fixLog.Printf("Failed to delete old template files: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to delete old template files: %v", err)))
	}
	fixLog.Print("Deleting old agent files")
	if err := deleteLegacyAgentFiles(verbose); err != nil {
		fixLog.Printf("Failed to delete old agent files: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to delete old agent files: %v", err)))
	}
}

func runFixCommandDeleteDeprecatedSchema(write bool, verbose bool) {
	schemaPath := filepath.Join(".github", "aw", "schemas", "agentic-workflow.json")
	if !fileutil.FileExists(schemaPath) {
		return
	}
	fixLog.Printf("Found deprecated schema file at %s", schemaPath)
	if !write {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Would delete deprecated .github/aw/schemas/agentic-workflow.json"))
		return
	}
	if err := os.Remove(schemaPath); err != nil {
		fixLog.Printf("Failed to delete schema file: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to delete deprecated schema file: %v", err)))
		return
	}
	fixLog.Print("Deleted deprecated schema file")
	if verbose {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage("Deleted deprecated .github/aw/schemas/agentic-workflow.json"))
	}
}

func runFixCommandPrintSummary(stats runFixCommandStats, write bool) {
	fmt.Fprintln(os.Stderr, "")
	if write {
		runFixCommandPrintWriteSummary(stats)
	} else {
		runFixCommandPrintDryRunSummary(stats)
	}
}

func runFixCommandPrintWriteSummary(stats runFixCommandStats) {
	if stats.totalFixed > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fmt.Sprintf("✓ Fixed %d of %d workflow files", stats.totalFixed, stats.totalFiles)))
	} else if stats.totalGuidedErrors == 0 && stats.totalProcessingErrors == 0 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("✓ No fixes needed"))
	}
}

func runFixCommandPrintDryRunSummary(stats runFixCommandStats) {
	if stats.totalFixed == 0 {
		if stats.totalGuidedErrors == 0 && stats.totalProcessingErrors == 0 {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("✓ No fixes needed"))
		}
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Would fix %d of %d workflow files", stats.totalFixed, stats.totalFiles)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To fix these issues, run:"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  gh aw fix --write")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or fix them individually:"))
	fmt.Fprintln(os.Stderr, "")
	for _, wf := range stats.workflowsNeedingFixes {
		fmt.Fprintf(os.Stderr, "  gh aw fix %s --write\n", strings.TrimSuffix(wf.File, ".md"))
	}
}

// workflowFixInfo tracks workflow files that need fixes
type workflowFixInfo struct {
	File  string
	Fixes []string
}

// processWorkflowFileWithInfo processes a single workflow file and returns detailed fix information
func processWorkflowFileWithInfo(filePath string, codemods []Codemod, write bool, verbose bool) (bool, []string, error) {
	fixLog.Printf("Processing workflow file: %s", filePath)

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	currentContent := originalContent

	// Apply each codemod
	currentContent, appliedCodemods, hasChanges, err := processWorkflowFileWithInfoApplyCodemods(currentContent, codemods)
	if err != nil {
		return false, nil, err
	}

	// If no changes, report and return
	if !hasChanges {
		if verbose {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("  %s - no fixes needed", filepath.Base(filePath))))
		}
		return false, nil, nil
	}

	// Report changes
	fileName := filepath.Base(filePath)
	if write {
		if err := processWorkflowFileWithInfoWrite(filePath, fileName, currentContent, appliedCodemods, verbose); err != nil {
			return false, nil, err
		}
	} else {
		processWorkflowFileWithInfoDryRun(fileName, appliedCodemods)
	}

	return true, appliedCodemods, nil
}

func processWorkflowFileWithInfoApplyCodemods(currentContent string, codemods []Codemod) (string, []string, bool, error) {
	var appliedCodemods []string
	var hasChanges bool
	for _, codemod := range codemods {
		fixLog.Printf("Attempting codemod: %s", codemod.ID)
		currentResult, err := parser.ExtractFrontmatterFromContent(currentContent)
		if err != nil {
			fixLog.Printf("Failed to parse frontmatter for codemod %s: %v", codemod.ID, err)
			continue
		}

		newContent, applied, err := codemod.Apply(currentContent, currentResult.Frontmatter)
		if err != nil {
			fixLog.Printf("Codemod %s failed: %v", codemod.ID, err)
			wrappedErr := fmt.Errorf("codemod %s failed: %w", codemod.ID, err)
			if codemod.Guided {
				return "", nil, false, &GuidedError{Cause: wrappedErr}
			}
			return "", nil, false, wrappedErr
		}

		if applied {
			currentContent = newContent
			appliedCodemods = append(appliedCodemods, codemod.Name)
			hasChanges = true
			fixLog.Printf("Applied codemod: %s", codemod.ID)
		}
	}
	return currentContent, appliedCodemods, hasChanges, nil
}

func processWorkflowFileWithInfoWrite(filePath, fileName, currentContent string, appliedCodemods []string, verbose bool) error {
	// Write the file with owner-only read/write permissions (0600) for security best practices
	if err := os.WriteFile(filePath, []byte(currentContent), constants.FilePermSensitive); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err := scaffoldSerenaSharedWorkflowIfNeeded(filePath, appliedCodemods, currentContent, verbose); err != nil {
		return fmt.Errorf("failed to scaffold shared Serena workflow: %w", err)
	}
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fileName))
	for _, codemodName := range appliedCodemods {
		fmt.Fprintf(os.Stderr, "    • %s\n", codemodName)
	}
	return nil
}

func processWorkflowFileWithInfoDryRun(fileName string, appliedCodemods []string) {
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage("⚠ "+fileName))
	for _, codemodName := range appliedCodemods {
		fmt.Fprintf(os.Stderr, "    • %s\n", codemodName)
	}
}

const scaffoldedSerenaSharedWorkflow = `---
import-schema:
  languages:
    type: array
    items:
      type: string
    required: true
    description: >
      List of programming language identifiers to enable for Serena LSP analysis.
      Supported values include: go, typescript, javascript, python, rust, java,
      ruby, csharp, cpp, c, kotlin, scala, swift, php, and more.

imports:
  - uses: github/gh-aw/.github/workflows/shared/mcp/serena.md@main
    with:
      languages: ${{ github.aw.import-inputs.languages }}
---
`

func scaffoldSerenaSharedWorkflowIfNeeded(filePath string, appliedCodemods []string, content string, verbose bool) error {
	if !wasAnyCodemodApplied(
		appliedCodemods,
		"Migrate tools.serena to shared Serena import",
		"Migrate tools.serena or engine.tools.serena to shared Serena import",
	) {
		return nil
	}
	if !strings.Contains(content, "shared/mcp/serena.md") {
		return nil
	}

	workflowRoot := resolveWorkflowRoot(filePath)
	serenaPath := filepath.Join(workflowRoot, "shared", "mcp", "serena.md")
	if _, err := os.Stat(serenaPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(serenaPath), constants.DirPermPublic); err != nil {
		return err
	}

	if err := os.WriteFile(serenaPath, []byte(scaffoldedSerenaSharedWorkflow), constants.FilePermSensitive); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Scaffolded "+serenaPath))
	}

	return nil
}

func wasCodemodApplied(appliedCodemods []string, codemodName string) bool {
	return slices.Contains(appliedCodemods, codemodName)
}

func wasAnyCodemodApplied(appliedCodemods []string, codemodNames ...string) bool {
	for _, codemodName := range codemodNames {
		if wasCodemodApplied(appliedCodemods, codemodName) {
			return true
		}
	}
	return false
}

func resolveWorkflowRoot(filePath string) string {
	clean := filepath.Clean(filePath)
	needle := filepath.Join(".github", "workflows")
	needleWithSep := needle + string(filepath.Separator)
	if idx := strings.Index(clean, needleWithSep); idx >= 0 {
		return clean[:idx+len(needle)]
	}
	return filepath.Dir(clean)
}
