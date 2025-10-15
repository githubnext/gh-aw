package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// getPackagesDir returns the global packages directory path
func getPackagesDir() (string, error) {
	// Use global directory under user's home
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".aw", "packages"), nil
}

func getWorkflowsDir() string {
	return ".github/workflows"
}

// readWorkflowFile reads a workflow file from either filesystem
func readWorkflowFile(filePath string, workflowsDir string) ([]byte, string, error) {
	// Using local filesystem
	fullPath := filepath.Join(workflowsDir, filePath)
	if !strings.HasPrefix(fullPath, workflowsDir) {
		// If filePath is already absolute
		fullPath = filePath
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
	// Start spinner for network operation (only if not in verbose mode)
	spinner := console.NewSpinner("Fetching GitHub workflow status...")
	if !verbose {
		spinner.Start()
	}

	args := []string{"workflow", "list", "--all", "--json", "id,name,path,state"}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()

	// Stop spinner
	if !verbose {
		spinner.Stop()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute gh workflow list command: %w", err)
	}

	// Check if output is empty
	if len(output) == 0 {
		return nil, fmt.Errorf("gh workflow list returned empty output - check if repository has workflows and gh CLI is authenticated")
	}

	// Validate JSON before unmarshaling
	if !json.Valid(output) {
		return nil, fmt.Errorf("gh workflow list returned invalid JSON - this may be due to network issues or authentication problems")
	}

	var workflows []GitHubWorkflow
	if err := json.Unmarshal(output, &workflows); err != nil {
		return nil, fmt.Errorf("failed to parse workflow data: %w", err)
	}

	workflowMap := make(map[string]*GitHubWorkflow)
	for i, workflow := range workflows {
		name := extractWorkflowNameFromPath(workflow.Path)
		workflowMap[name] = &workflows[i]
	}

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

	return nil, fmt.Errorf("workflow '%s' not found on GitHub", workflowIdOrName)
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
	cmd := exec.Command("gh", args...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to restore workflow '%s' to disabled state: %v", workflowIdOrName, err)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Restored workflow to disabled state: %s", workflowIdOrName)))
	}
}
