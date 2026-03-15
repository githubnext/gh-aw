package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var workflowsLog = logger.New("cli:workflows")

func getWorkflowsDir() string {
	return ".github/workflows"
}

// readWorkflowFile reads a workflow file from either filesystem
func readWorkflowFile(filePath string, workflowsDir string) ([]byte, string, error) {
	// Using local filesystem
	var fullPath string

	// Check if filePath is already an absolute path
	if filepath.IsAbs(filePath) {
		fullPath = filePath
	} else {
		// Join relative path with workflowsDir
		fullPath = filepath.Join(workflowsDir, filePath)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read workflow file %s: %w", fullPath, err)
	}
	return content, fullPath, nil
}

// GitHubWorkflow represents a workflow from GitHub API
type GitHubWorkflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

// fetchGitHubWorkflows fetches workflow information from GitHub
func fetchGitHubWorkflows(repoOverride string, verbose bool) (map[string]*GitHubWorkflow, error) {
	workflowsLog.Printf("Fetching GitHub workflows: repoOverride=%s", repoOverride)

	// Start spinner for network operation (only if not in verbose mode)
	spinner := console.NewSpinner("Fetching GitHub workflow status...")
	if !verbose {
		spinner.Start()
	}

	args := []string{"workflow", "list", "--all", "--json", "id,name,path,state"}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()

	if err != nil {
		// Stop spinner on error
		if !verbose {
			spinner.Stop()
		}

		// Extract detailed error information including exit code and stderr
		var exitCode int
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
			workflowsLog.Printf("gh workflow list command failed with exit code %d. Command: gh %v", exitCode, args)
			workflowsLog.Printf("stderr output: %s", stderr)

			return nil, fmt.Errorf("failed to execute gh workflow list command (exit code %d): %w. stderr: %s", exitCode, err, stderr)
		}

		// If not an ExitError, log what we can
		workflowsLog.Printf("gh workflow list command failed with error (not ExitError): %v. Command: gh %v", err, args)
		return nil, fmt.Errorf("failed to execute gh workflow list command: %w", err)
	}

	// Check if output is empty
	if len(output) == 0 {
		if !verbose {
			spinner.Stop()
		}
		return nil, errors.New("gh workflow list returned empty output - check if repository has workflows and gh CLI is authenticated")
	}

	// Validate JSON before unmarshaling
	if !json.Valid(output) {
		if !verbose {
			spinner.Stop()
		}
		return nil, errors.New("gh workflow list returned invalid JSON - this may be due to network issues or authentication problems")
	}

	var workflows []GitHubWorkflow
	if err := json.Unmarshal(output, &workflows); err != nil {
		if !verbose {
			spinner.Stop()
		}
		return nil, fmt.Errorf("failed to parse workflow data: %w", err)
	}

	workflowMap := make(map[string]*GitHubWorkflow)
	for i, workflow := range workflows {
		name := extractWorkflowNameFromPath(workflow.Path)
		workflowMap[name] = &workflows[i]
	}

	// Count user workflows (those with .md files)
	mdFiles, _ := getMarkdownWorkflowFiles("")
	mdWorkflowNames := make(map[string]bool)
	for _, file := range mdFiles {
		name := extractWorkflowNameFromPath(file)
		mdWorkflowNames[name] = true
	}

	var userWorkflowCount int
	for name := range workflowMap {
		if mdWorkflowNames[name] {
			userWorkflowCount++
		}
	}

	// Stop spinner with success message showing only user workflow count
	if !verbose {
		if userWorkflowCount == 1 {
			spinner.StopWithMessage("✓ Fetched 1 workflow")
		} else {
			spinner.StopWithMessage(fmt.Sprintf("✓ Fetched %d workflows", userWorkflowCount))
		}
	}

	workflowsLog.Printf("Fetched %d GitHub workflows (%d with .md files)", len(workflowMap), userWorkflowCount)
	return workflowMap, nil
}

// extractWorkflowNameFromPath extracts workflow name from path
func extractWorkflowNameFromPath(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.TrimSuffix(name, ".lock")
}

// getWorkflowStatus gets the status of a single workflow by name
func getWorkflowStatus(workflowIdOrName string, repoOverride string, verbose bool) (*GitHubWorkflow, error) {
	workflowsLog.Printf("Getting workflow status: workflow=%s", workflowIdOrName)

	// Extract workflow name for lookup
	filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")

	// Get all GitHub workflows
	githubWorkflows, err := fetchGitHubWorkflows(repoOverride, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub workflows: %w", err)
	}

	// Find the workflow
	if workflow, exists := githubWorkflows[filename]; exists {
		return workflow, nil
	}

	suggestions := []string{
		fmt.Sprintf("Run '%s status' to see all available workflows", string(constants.CLIExtensionPrefix)),
		"Check if the workflow has been compiled and pushed to GitHub",
		"Verify the workflow name matches the compiled .lock.yml file",
	}
	return nil, errors.New(console.FormatErrorWithSuggestions(
		fmt.Sprintf("workflow '%s' not found on GitHub", workflowIdOrName),
		suggestions,
	))
}

// restoreWorkflowState restores a workflow to disabled state if it was previously disabled
func restoreWorkflowState(workflowIdOrName string, workflowID int64, repoOverride string, verbose bool) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Restoring workflow '%s' to disabled state...", workflowIdOrName)))
	}

	args := []string{"workflow", "disable", strconv.FormatInt(workflowID, 10)}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	cmd := workflow.ExecGH(args...)
	if err := cmd.Run(); err != nil {
		// Extract detailed error information including exit code and stderr
		var exitCode int
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
			workflowsLog.Printf("gh workflow disable command failed with exit code %d. Command: gh %v", exitCode, args)
			workflowsLog.Printf("stderr output: %s", stderr)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to restore workflow '%s' to disabled state (exit code %d): %v. stderr: %s", workflowIdOrName, exitCode, err, stderr)))
		} else {
			workflowsLog.Printf("gh workflow disable command failed with error (not ExitError): %v. Command: gh %v", err, args)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to restore workflow '%s' to disabled state: %v", workflowIdOrName, err)))
		}
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Restored workflow to disabled state: "+workflowIdOrName))
	}
}

// getAvailableWorkflowNames returns a list of available workflow names (without .md extension)
func getAvailableWorkflowNames() []string {
	mdFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return nil
	}

	return sliceutil.Map(mdFiles, func(file string) string {
		return strings.TrimSuffix(filepath.Base(file), ".md")
	})
}

// suggestWorkflowNames returns up to 3 similar workflow names using fuzzy matching
// The target can be a workflow name or filename (with or without .md extension)
func suggestWorkflowNames(target string) []string {
	availableNames := getAvailableWorkflowNames()
	if len(availableNames) == 0 {
		return nil
	}

	// Normalize target: strip .md extension and get basename if it's a path
	normalizedTarget := strings.TrimSuffix(filepath.Base(target), ".md")

	// Use the existing FindClosestMatches function from parser package
	return parser.FindClosestMatches(normalizedTarget, availableNames, 3)
}

// isWorkflowFile returns true if the file should be treated as a workflow file.
// README.md files are excluded as they are documentation, not workflows.
func isWorkflowFile(filename string) bool {
	base := strings.ToLower(filepath.Base(filename))
	return base != "readme.md"
}

// filterWorkflowFiles filters out non-workflow files from a list of markdown files.
func filterWorkflowFiles(files []string) []string {
	return sliceutil.Filter(files, isWorkflowFile)
}

// getMarkdownWorkflowFiles discovers markdown workflow files in the specified directory
func getMarkdownWorkflowFiles(workflowDir string) ([]string, error) {
	// Use provided directory or default
	workflowsDir := workflowDir
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no %s directory found", workflowsDir)
	}

	// Find all markdown files in workflow directory
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow files: %w", err)
	}

	// Filter out README.md files
	mdFiles = filterWorkflowFiles(mdFiles)

	return mdFiles, nil
}

// fastParseTitle scans markdown content for the first H1 header, skipping an
// optional frontmatter block, without performing a full YAML parse.
//
// Frontmatter is recognised only when "---" appears on the very first line
// (matching the behaviour of ExtractFrontmatterFromContent). Returns the H1
// title text, or ("", nil) when no H1 header is present. Returns an error if
// frontmatter is opened but never closed.
func fastParseTitle(content string) (string, error) {
	firstLine := true
	inFrontmatter := false
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if firstLine {
			firstLine = false
			if trimmed == "---" {
				inFrontmatter = true
				continue
			}
		} else if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:]), nil
		}
	}

	// Unclosed frontmatter is an error (consistent with ExtractFrontmatterFromContent).
	if inFrontmatter {
		return "", errors.New("frontmatter not properly closed")
	}

	return "", nil
}

// extractWorkflowNameFromFile extracts the workflow name from a file's H1 header
func extractWorkflowNameFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	title, err := fastParseTitle(string(content))
	if err != nil {
		return "", err
	}
	if title != "" {
		return title, nil
	}

	// No H1 header found, generate default name from filename
	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	baseName = strings.ReplaceAll(baseName, "-", " ")

	// Capitalize first letter of each word
	words := strings.Fields(baseName)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " "), nil
}

// extractEngineIDFromFile extracts the engine ID from a workflow file's frontmatter
func extractEngineIDFromFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "" // Return empty string if file cannot be read
	}

	// Parse frontmatter
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return "" // Return empty string if frontmatter cannot be parsed
	}

	// Use the workflow package's extractEngineConfig to handle both string and object formats
	compiler := &workflow.Compiler{}
	engineSetting, engineConfig := compiler.ExtractEngineConfig(result.Frontmatter)

	// If engine is specified, return the ID from the config
	if engineConfig != nil && engineConfig.ID != "" {
		return engineConfig.ID
	}

	// If we have an engine setting string, return it
	if engineSetting != "" {
		return engineSetting
	}

	return "copilot" // Default engine
}

// normalizeWorkflowID extracts the workflow ID from a workflow identifier.
// It handles both workflow IDs ("my-workflow") and full paths (".github/workflows/my-workflow.md").
// Returns the workflow ID without .md or .lock.yml extension.
//
// Note: unlike stringutil.NormalizeWorkflowName (which operates on bare names),
// this function also handles file system paths by extracting the basename first.
func normalizeWorkflowID(workflowIDOrPath string) string {
	// Get the base filename if it's a path
	basename := filepath.Base(workflowIDOrPath)

	// Remove .md or .lock.yml extension if present
	return stringutil.NormalizeWorkflowName(basename)
}

// resolveWorkflowFile resolves a file or workflow name to an actual file path
// Note: This function only looks for local workflows, not packages
func resolveWorkflowFile(fileOrWorkflowName string, verbose bool) (string, error) {
	return resolveWorkflowFileInDir(fileOrWorkflowName, verbose, "")
}

func resolveWorkflowFileInDir(fileOrWorkflowName string, verbose bool, workflowDir string) (string, error) {
	// First, try to use it as a direct file path
	if _, err := os.Stat(fileOrWorkflowName); err == nil {
		workflowsLog.Printf("Found workflow file at path: %s", fileOrWorkflowName)
		console.LogVerbose(verbose, "Found workflow file at path: "+fileOrWorkflowName)
		// Return absolute path
		absPath, err := filepath.Abs(fileOrWorkflowName)
		if err != nil {
			return fileOrWorkflowName, nil // fallback to original path
		}
		return absPath, nil
	}

	// If it's not a direct file path, try to resolve it as a workflow name
	workflowsLog.Printf("File not found at %s, trying to resolve as workflow name", fileOrWorkflowName)

	// Add .md extension if not present
	workflowPath := fileOrWorkflowName
	if !strings.HasSuffix(workflowPath, ".md") {
		workflowPath += ".md"
	}

	workflowsLog.Printf("Looking for workflow file: %s", workflowPath)

	// Use provided directory or default
	workflowsDir := workflowDir
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	// Try to find the workflow in local sources only (not packages)
	_, path, err := readWorkflowFile(workflowPath, workflowsDir)
	if err != nil {
		suggestions := []string{
			fmt.Sprintf("Run '%s status' to see all available workflows", string(constants.CLIExtensionPrefix)),
			fmt.Sprintf("Create a new workflow with '%s new %s'", string(constants.CLIExtensionPrefix), fileOrWorkflowName),
			"Check for typos in the workflow name",
		}

		// Add fuzzy match suggestions
		similarNames := suggestWorkflowNames(fileOrWorkflowName)
		if len(similarNames) > 0 {
			suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
		}

		return "", errors.New(console.FormatErrorWithSuggestions(
			fmt.Sprintf("workflow '%s' not found in local .github/workflows", fileOrWorkflowName),
			suggestions,
		))
	}

	workflowsLog.Print("Found workflow in local .github/workflows")

	// Return absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // fallback to original path
	}
	return absPath, nil
}
