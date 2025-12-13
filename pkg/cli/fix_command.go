package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var fixLog = logger.New("cli:fix_command")

// NewFixCommand creates the fix command
func NewFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [workflow-id]...",
		Short: "Apply automatic codemod-style fixes to agentic workflow files",
		Long: `Apply automatic codemod-style fixes to agentic workflow Markdown files.

This command applies a registry of codemods that automatically update deprecated fields
and migrate to new syntax. Codemods preserve formatting and comments as much as possible.

Available codemods:
  • timeout-minutes-migration: Replaces 'timeout_minutes' with 'timeout-minutes'
  • network-firewall-migration: Replaces 'network.firewall' with 'sandbox.agent: false'

If no workflows are specified, all Markdown files in .github/workflows will be processed.

The command will:
  1. Scan workflow files for deprecated fields
  2. Apply relevant codemods to fix issues
  3. Report what was changed in each file
  4. Write updated files back to disk (with --write flag)

Examples:
  ` + constants.CLIExtensionPrefix + ` fix                     # Check all workflows (dry-run)
  ` + constants.CLIExtensionPrefix + ` fix --write             # Fix all workflows
  ` + constants.CLIExtensionPrefix + ` fix my-workflow         # Check specific workflow
  ` + constants.CLIExtensionPrefix + ` fix my-workflow --write # Fix specific workflow
  ` + constants.CLIExtensionPrefix + ` fix --list-codemods     # List available codemods`,
		RunE: func(cmd *cobra.Command, args []string) error {
			listCodemods, _ := cmd.Flags().GetBool("list-codemods")
			write, _ := cmd.Flags().GetBool("write")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if listCodemods {
				return listAvailableCodemods()
			}

			return runFixCommand(args, write, verbose)
		},
	}

	cmd.Flags().Bool("write", false, "Write changes to files (default is dry-run)")
	cmd.Flags().Bool("list-codemods", false, "List all available codemods and exit")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames

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
		fmt.Fprintf(os.Stderr, "    %s\n", codemod.Description)
		fmt.Fprintln(os.Stderr, "")
	}

	return nil
}

// runFixCommand runs the fix command on specified or all workflows
func runFixCommand(workflowIDs []string, write bool, verbose bool) error {
	fixLog.Printf("Running fix command: workflowIDs=%v, write=%v, verbose=%v", workflowIDs, write, verbose)

	// Get workflow files to process
	var files []string
	var err error

	if len(workflowIDs) > 0 {
		// Process specific workflows
		for _, workflowID := range workflowIDs {
			file, err := resolveWorkflowFile(workflowID, verbose)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
	} else {
		// Process all workflows in .github/workflows
		files, err = getMarkdownWorkflowFiles()
		if err != nil {
			return err
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflow files found."))
		return nil
	}

	// Load all codemods
	codemods := GetAllCodemods()
	fixLog.Printf("Loaded %d codemods", len(codemods))

	// Process each file
	var totalFixed int
	var totalFiles int

	for _, file := range files {
		fixLog.Printf("Processing file: %s", file)
		
		fixed, err := processWorkflowFile(file, codemods, write, verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatErrorMessage(fmt.Sprintf("Error processing %s: %v", filepath.Base(file), err)))
			continue
		}

		totalFiles++
		if fixed {
			totalFixed++
		}
	}

	// Print summary
	fmt.Fprintln(os.Stderr, "")
	if write {
		if totalFixed > 0 {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fmt.Sprintf("✓ Fixed %d of %d workflow files", totalFixed, totalFiles)))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("✓ No fixes needed"))
		}
	} else {
		if totalFixed > 0 {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Would fix %d of %d workflow files", totalFixed, totalFiles)))
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run with --write to apply changes"))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("✓ No fixes needed"))
		}
	}

	return nil
}

// processWorkflowFile processes a single workflow file with all codemods
func processWorkflowFile(filePath string, codemods []Codemod, write bool, verbose bool) (bool, error) {
	fixLog.Printf("Processing workflow file: %s", filePath)

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	currentContent := originalContent

	// Track what was applied
	var appliedCodemods []string
	var hasChanges bool

	// Apply each codemod
	for _, codemod := range codemods {
		fixLog.Printf("Attempting codemod: %s", codemod.ID)

		// Re-parse frontmatter for each codemod to get fresh state
		currentResult, err := parser.ExtractFrontmatterFromContent(currentContent)
		if err != nil {
			fixLog.Printf("Failed to parse frontmatter for codemod %s: %v", codemod.ID, err)
			continue
		}

		newContent, applied, err := codemod.Apply(currentContent, currentResult.Frontmatter)
		if err != nil {
			fixLog.Printf("Codemod %s failed: %v", codemod.ID, err)
			return false, fmt.Errorf("codemod %s failed: %w", codemod.ID, err)
		}

		if applied {
			currentContent = newContent
			appliedCodemods = append(appliedCodemods, codemod.Name)
			hasChanges = true
			fixLog.Printf("Applied codemod: %s", codemod.ID)
		}
	}

	// If no changes, report and return
	if !hasChanges {
		if verbose {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("  %s - no fixes needed", filepath.Base(filePath))))
		}
		return false, nil
	}

	// Report changes
	fileName := filepath.Base(filePath)
	if write {
		// Write the file
		if err := os.WriteFile(filePath, []byte(currentContent), 0644); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fmt.Sprintf("✓ %s", fileName)))
		for _, codemodName := range appliedCodemods {
			fmt.Fprintf(os.Stderr, "    • %s\n", codemodName)
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(fmt.Sprintf("⚠ %s", fileName)))
		for _, codemodName := range appliedCodemods {
			fmt.Fprintf(os.Stderr, "    • %s\n", codemodName)
		}
	}

	return true, nil
}
