package cli

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
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
			verbose := importCommandVerbose(cmd)
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

	info, err := os.Stat(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to stat workflow file %s: %w", workflowFile, err)
	}
	perm := info.Mode().Perm()

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

	if err := os.WriteFile(workflowFile, []byte(updated), perm); err != nil {
		return fmt.Errorf("failed to write workflow file %s: %w", workflowFile, err)
	}

	fmt.Printf("Added import %q to %s\n", opts.ImportPath, workflowFile)
	return nil
}

// addImportToWorkflow adds importPath to the imports: frontmatter field of content.
// Returns the updated content, whether the import was newly added, and any error.
func addImportToWorkflow(content, importPath string) (string, bool, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	if result.FrontmatterStart == 0 {
		return "", false, errors.New("no frontmatter found in workflow file")
	}

	updatedImports, added, err := appendImportValue(result.Frontmatter["imports"], importPath)
	if err != nil {
		return "", false, err
	}
	if !added {
		return content, false, nil
	}

	updated, err := updateImportsFieldInFrontmatterRaw(content, result.FrontmatterLines, updatedImports)
	if err != nil {
		return "", false, err
	}
	return updated, true, nil
}

func importCommandVerbose(cmd *cobra.Command) bool {
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	return verbose
}

func appendImportValue(imports any, importPath string) (any, bool, error) {
	switch value := imports.(type) {
	case nil:
		return []any{importPath}, true, nil
	case []any:
		if importListContains(value, importPath) {
			return value, false, nil
		}
		return append(append([]any(nil), value...), importPath), true, nil
	case map[string]any:
		updated := make(map[string]any, len(value)+1)
		maps.Copy(updated, value)

		switch aw := updated["aw"].(type) {
		case nil:
			updated["aw"] = []any{importPath}
			return updated, true, nil
		case []any:
			if importListContains(aw, importPath) {
				return value, false, nil
			}
			updated["aw"] = append(append([]any(nil), aw...), importPath)
			return updated, true, nil
		default:
			return nil, false, fmt.Errorf("unexpected imports.aw field type %T", updated["aw"])
		}
	default:
		return nil, false, fmt.Errorf("unexpected imports field type %T", imports)
	}
}

func importListContains(imports []any, importPath string) bool {
	for _, entry := range imports {
		if importEntryMatches(entry, importPath) {
			return true
		}
	}
	return false
}

func importEntryMatches(entry any, importPath string) bool {
	switch value := entry.(type) {
	case string:
		return value == importPath
	case map[string]any:
		if uses, ok := value["uses"].(string); ok && uses == importPath {
			return true
		}
		if path, ok := value["path"].(string); ok && path == importPath {
			return true
		}
	}
	return false
}

func updateImportsFieldInFrontmatterRaw(content string, frontmatterLines []string, imports any) (string, error) {
	renderedImports, err := renderImportsFieldLines(imports, usesCRLFLineEndings(content))
	if err != nil {
		return "", err
	}

	updatedFrontmatterLines := make([]string, 0, len(frontmatterLines)+len(renderedImports))
	fieldUpdated := false
	skipChildren := false
	fieldIndentLevel := 0

	for _, line := range frontmatterLines {
		if skipChildren {
			if strings.TrimSpace(line) == "" {
				continue
			}
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent > fieldIndentLevel {
				continue
			}
			skipChildren = false
		}

		if !fieldUpdated && isTopLevelFieldLine(line, "imports") {
			if comment := inlineYAMLComment(line); comment != "" {
				renderedImports[0] += " " + comment
			}
			updatedFrontmatterLines = append(updatedFrontmatterLines, renderedImports...)
			fieldUpdated = true
			fieldIndentLevel = len(line) - len(strings.TrimLeft(line, " \t"))
			skipChildren = true
			continue
		}

		updatedFrontmatterLines = append(updatedFrontmatterLines, line)
	}

	if !fieldUpdated {
		updatedFrontmatterLines = append(updatedFrontmatterLines, renderedImports...)
	}

	return replaceFrontmatterLines(content, updatedFrontmatterLines)
}

func renderImportsFieldLines(imports any, useCRLF bool) ([]string, error) {
	rendered, err := yaml.MarshalWithOptions(map[string]any{
		"imports": imports,
	}, append(append([]yaml.EncodeOption{}, workflow.DefaultMarshalOptions...), yaml.IndentSequence(true))...)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal imports field: %w", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(rendered), "\n"), "\n")
	if useCRLF {
		for i := range lines {
			lines[i] += "\r"
		}
	}

	return lines, nil
}

func replaceFrontmatterLines(content string, frontmatterLines []string) (string, error) {
	lines := strings.Split(content, "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("no frontmatter found in workflow file")
	}

	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatterEnd = i
			break
		}
	}
	if frontmatterEnd == -1 {
		return "", errors.New("frontmatter closing delimiter not found")
	}

	updatedLines := make([]string, 0, len(lines)-frontmatterEnd+len(frontmatterLines)+1)
	updatedLines = append(updatedLines, lines[0])
	updatedLines = append(updatedLines, frontmatterLines...)
	updatedLines = append(updatedLines, lines[frontmatterEnd:]...)
	return strings.Join(updatedLines, "\n"), nil
}

func inlineYAMLComment(line string) string {
	colonIndex := strings.IndexByte(line, ':')
	if colonIndex == -1 {
		return ""
	}

	for i := colonIndex + 1; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		if line[i-1] != ' ' && line[i-1] != '\t' {
			continue
		}
		return strings.TrimSpace(line[i:])
	}

	return ""
}

func usesCRLFLineEndings(content string) bool {
	return strings.Contains(content, "\r\n")
}
