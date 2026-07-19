package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/spf13/cobra"
)

var listWorkflowsLog = logger.New("cli:list_workflows")

// WorkflowListItem represents a single workflow for list output
type WorkflowListItem struct {
	Workflow string   `json:"workflow" console:"header:workflow"`
	EngineID string   `json:"engine_id" console:"header:engine"`
	Compiled string   `json:"compiled" console:"header:compiled"`
	Labels   []string `json:"labels,omitempty" console:"-"`
	On       any      `json:"on,omitempty" console:"-"`
}

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [pattern]",
		Short: "List agentic workflows in the repository",
		Long: `List all agentic workflows in a repository without checking their status.

Displays a simplified table with workflow name, AI engine, and compilation status.
Unlike 'status', this command does not check GitHub workflow state or time remaining.

The optional pattern argument filters workflows by name (case-insensitive substring match).
It accepts workflow IDs (basename without .md) or full filenames.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` list                              # List all workflows in current repo
  ` + string(constants.CLIExtensionPrefix) + ` list --repo github/gh-aw          # List workflows from github/gh-aw repo
  ` + string(constants.CLIExtensionPrefix) + ` list --repo org/repo --path workflows  # List from custom path
  ` + string(constants.CLIExtensionPrefix) + ` list --dir custom/workflows        # List from custom local directory
  ` + string(constants.CLIExtensionPrefix) + ` list ci-                           # List workflows with 'ci-' in name
  ` + string(constants.CLIExtensionPrefix) + ` list --repo github/gh-aw ci-      # List workflows from github/gh-aw with 'ci-' in name
  ` + string(constants.CLIExtensionPrefix) + ` list --json                        # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` list --label automation            # List workflows with 'automation' label`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			repo, _ := cmd.Flags().GetString("repo")
			path, _ := cmd.Flags().GetString("path")
			dir, _ := cmd.Flags().GetString("dir")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			labelFilter, _ := cmd.Flags().GetString("label")

			// --dir overrides the local workflow directory when no remote repo is specified.
			// When --repo is set, --path is used for the remote repository path instead.
			if dir != "" && repo == "" {
				path = dir
			}
			return RunListWorkflows(cmd.Context(), repo, path, pattern, verbose, jsonFlag, labelFilter)
		},
	}

	addRepoFlag(cmd)
	addJSONFlag(cmd)
	cmd.Flags().String("label", "", "Filter workflows by label")
	cmd.Flags().String("path", constants.GetWorkflowDir(), "Path to workflows directory in the remote repository (used with --repo)")
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows; ignored when --repo is set)")

	// Register completions for list command
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}

// RunListWorkflows lists workflows without checking GitHub status
func RunListWorkflows(ctx context.Context, repo, path, pattern string, verbose bool, jsonOutput bool, labelFilter string) error {
	listWorkflowsLog.Printf("Listing workflows: repo=%s, path=%s, pattern=%s, jsonOutput=%v, labelFilter=%s", repo, path, pattern, jsonOutput, labelFilter)

	mdFiles, isRemote, err := runListWorkflowsFiles(ctx, repo, path, pattern, verbose, jsonOutput)
	if err != nil {
		listWorkflowsLog.Printf("Failed to get markdown workflow files: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	listWorkflowsLog.Printf("Found %d markdown workflow files", len(mdFiles))
	if len(mdFiles) == 0 {
		if jsonOutput {
			// Output empty array for JSON
			output := []WorkflowListItem{}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Fprintln(os.Stdout, string(jsonBytes))
			return nil
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflow files found."))
		return nil
	}

	if verbose && !jsonOutput {
		fmt.Fprintf(os.Stderr, "Found %d markdown workflow files\n", len(mdFiles))
	}

	// Build workflow list
	workflows := runListWorkflowsItems(mdFiles, pattern, labelFilter, isRemote)

	// Output results
	return runListWorkflowsOutput(workflows, jsonOutput)
}

func runListWorkflowsFiles(ctx context.Context, repo, path, pattern string, verbose bool, jsonOutput bool) ([]string, bool, error) {
	if repo != "" {
		if verbose && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Listing workflow files from %s\n", repo)
		}
		mdFiles, err := getRemoteWorkflowFiles(ctx, repo, path, verbose, jsonOutput)
		return mdFiles, true, err
	}
	if verbose && !jsonOutput {
		fmt.Fprintf(os.Stderr, "Listing workflow files\n")
		if pattern != "" {
			fmt.Fprintf(os.Stderr, "Filtering by pattern: %s\n", pattern)
		}
	}
	mdFiles, err := getMarkdownWorkflowFiles(path)
	return mdFiles, false, err
}

func runListWorkflowsItems(mdFiles []string, pattern, labelFilter string, isRemote bool) []WorkflowListItem {
	var workflows []WorkflowListItem
	importCache := parser.NewImportCache("")
	for _, file := range mdFiles {
		name := extractWorkflowNameFromPath(file)
		if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			continue
		}
		if isRemote {
			workflows = append(workflows, runListWorkflowsRemoteItem(name))
			continue
		}
		item, labels := runListWorkflowsLocalItem(file, name, importCache)
		if labelFilter != "" && !runListWorkflowsHasLabel(labels, labelFilter) {
			continue
		}
		workflows = append(workflows, item)
	}
	return workflows
}

func runListWorkflowsRemoteItem(name string) WorkflowListItem {
	return WorkflowListItem{
		Workflow: name,
		EngineID: "N/A", // Skip fetching to avoid slow API/git calls
		Compiled: "N/A", // Cannot determine for remote repos
		Labels:   nil,
		On:       nil,
	}
}

func runListWorkflowsLocalItem(file, name string, importCache *parser.ImportCache) (WorkflowListItem, []string) {
	agent := extractEngineIDFromFile(file)
	lockFile := stringutil.MarkdownToLockFile(file)
	compiled := "N/A"
	if _, err := os.Stat(lockFile); err == nil {
		compiled = isCompiledUpToDateWithCache(file, lockFile, importCache)
	}
	onField, labels := runListWorkflowsFrontmatter(file)
	return WorkflowListItem{
		Workflow: name,
		EngineID: agent,
		Compiled: compiled,
		Labels:   labels,
		On:       onField,
	}, labels
}

func runListWorkflowsFrontmatter(file string) (any, []string) {
	var onField any
	var labels []string
	if content, err := os.ReadFile(file); err == nil {
		if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil && result.Frontmatter != nil {
			onField = result.Frontmatter["on"]
			if labelsField, ok := result.Frontmatter["labels"]; ok {
				if labelsArray, ok := labelsField.([]any); ok {
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labels = append(labels, labelStr)
						}
					}
				}
			}
		}
	}
	return onField, labels
}

func runListWorkflowsHasLabel(labels []string, labelFilter string) bool {
	for _, label := range labels {
		if strings.EqualFold(label, labelFilter) {
			return true
		}
	}
	return false
}

func runListWorkflowsOutput(workflows []WorkflowListItem, jsonOutput bool) error {
	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(workflows, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return nil
	}
	if len(workflows) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found 1 workflow"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d workflows", len(workflows))))
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(workflows))
	return nil
}

// getRemoteWorkflowFiles fetches the list of workflow files from a remote repository
func getRemoteWorkflowFiles(ctx context.Context, repoSpec, workflowPath string, verbose bool, jsonOutput bool) ([]string, error) {
	listWorkflowsLog.Printf("Fetching remote workflow files: repoSpec=%s, path=%s", repoSpec, workflowPath)
	// Parse repo spec: owner/repo[@ref]
	var owner, repo, ref string
	parts := strings.SplitN(repoSpec, "@", 2)
	repoPart := parts[0]
	if len(parts) == 2 {
		ref = parts[1]
	} else {
		ref = "main" // default to main branch
	}

	// Parse owner/repo
	repoParts := strings.Split(repoPart, "/")
	if len(repoParts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s (expected owner/repo or owner/repo@ref)", repoSpec)
	}
	owner = repoParts[0]
	repo = repoParts[1]

	if verbose && !jsonOutput {
		fmt.Fprintf(os.Stderr, "Fetching workflow files from %s/%s@%s (path: %s)\n", owner, repo, ref, workflowPath)
	}

	// Use the parser package to list workflow files
	listWorkflowsLog.Printf("Listing remote workflow files: owner=%s, repo=%s, ref=%s, path=%s", owner, repo, ref, workflowPath)
	files, err := parser.ListWorkflowFiles(ctx, owner, repo, ref, workflowPath)
	if err != nil {
		listWorkflowsLog.Printf("Failed to list remote workflow files: %v", err)
		return nil, fmt.Errorf("failed to list workflow files from %s/%s: %w", owner, repo, err)
	}

	listWorkflowsLog.Printf("Found %d remote workflow files", len(files))
	return files, nil
}
