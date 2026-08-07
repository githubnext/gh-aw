package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var importLog = logger.New("cli:import_command")

// ImportOptions contains configuration for the import command.
type ImportOptions struct {
	WorkflowID string
	ImportPath string
	Verbose    bool
}

// NewImportCommand creates the import command.
func NewImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <workflow> <import-path>",
		Short: "Add an import to an existing agentic workflow",
		Long: `Add an import entry to the imports: field of an existing agentic workflow.

The import path is added to the frontmatter imports list if it is not already present.
The workflow file is updated in place.

` + WorkflowIDExplanation,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` import my-workflow shared/security-notice.md
  ` + string(constants.CLIExtensionPrefix) + ` import my-workflow copilot-setup-steps.yml
  ` + string(constants.CLIExtensionPrefix) + ` import my-workflow github/gh-aw/shared/mcp/tavily.md@main`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			return RunImport(ImportOptions{
				WorkflowID: args[0],
				ImportPath: args[1],
				Verbose:    verbose,
			})
		},
	}
	return cmd
}

// RunImport adds an import to an existing workflow file.
func RunImport(opts ImportOptions) error {
	importLog.Printf("Adding import %q to workflow %q", opts.ImportPath, opts.WorkflowID)

	workflowFile, err := resolveWorkflowFile(opts.WorkflowID, opts.Verbose)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to read workflow file %s: %w", workflowFile, err)
	}

	updated, added, err := addImportToWorkflow(string(content), opts.ImportPath)
	if err != nil {
		return fmt.Errorf("failed to add import to workflow: %w", err)
	}

	if !added {
		fmt.Printf("Import %q is already present in %s\n", opts.ImportPath, workflowFile)
		return nil
	}

	if err := os.WriteFile(workflowFile, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file %s: %w", workflowFile, err)
	}

	fmt.Printf("Added import %q to %s\n", opts.ImportPath, workflowFile)
	return nil
}

// addImportToWorkflow adds importPath to the imports: frontmatter field of content.
// Returns the updated content, whether the import was newly added, and any error.
func addImportToWorkflow(content, importPath string) (string, bool, error) {
	// The parser returns no error but an empty frontmatter map when there is no
	// frontmatter delimiter ("---") in the file.
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return "", false, errors.New("no frontmatter found in workflow file")
	}

	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	imports, existing := result.Frontmatter["imports"]

	var importList []any

	switch v := imports.(type) {
	case []any:
		importList = v
	case []string:
		importList = make([]any, 0, len(v))
		for _, s := range v {
			importList = append(importList, s)
		}
	case nil:
		importList = nil
	default:
		return "", false, fmt.Errorf("unexpected imports field type %T", imports)
	}

	if existing {
		// Check for duplicates
		for _, entry := range importList {
			if str, ok := entry.(string); ok && str == importPath {
				return content, false, nil
			}
		}
	}

	importList = append(importList, importPath)
	result.Frontmatter["imports"] = importList

	updated, err := reconstructWorkflowFileFromMap(result.Frontmatter, result.Markdown)
	if err != nil {
		return "", false, err
	}

	// reconstructWorkflowFileFromMap may append a trailing newline; preserve the
	// original file's trailing-newline behaviour.
	if !strings.HasSuffix(content, "\n") && strings.HasSuffix(updated, "\n") {
		updated = strings.TrimSuffix(updated, "\n")
	}

	return updated, true, nil
}
