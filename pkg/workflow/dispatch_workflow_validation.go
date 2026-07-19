package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/goccy/go-yaml"
)

var dispatchWorkflowValidationLog = logger.New("workflow:dispatch_workflow_validation")

// validateDispatchWorkflow validates that the dispatch-workflow configuration is correct
func (c *Compiler) validateDispatchWorkflow(data *WorkflowData, workflowPath string) error {
	dispatchWorkflowValidationLog.Print("Starting dispatch-workflow validation")

	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		dispatchWorkflowValidationLog.Print("No dispatch-workflow configuration found")
		return nil
	}

	config := data.SafeOutputs.DispatchWorkflow

	if len(config.Workflows) == 0 {
		return errors.New("dispatch-workflow: must specify at least one workflow in the list\n\nExample configuration in workflow frontmatter:\nsafe-outputs:\n  dispatch-workflow:\n    workflows: [workflow-name-1, workflow-name-2]\n\nWorkflow names should match the filename without the .md extension")
	}

	if c.shouldSkipLocalDispatchWorkflowValidation(config.TargetRepoSlug) {
		dispatchWorkflowValidationLog.Printf("Skipping local dispatch-workflow validation because target-repo is cross-repo: %q", config.TargetRepoSlug)
		return nil
	}

	currentWorkflowName := getCurrentWorkflowName(workflowPath)
	dispatchWorkflowValidationLog.Printf("Current workflow name: %s", currentWorkflowName)
	collector := NewErrorCollector(c.failFast)

	for _, workflowName := range config.Workflows {
		if err := validateDispatchWorkflowTarget(workflowName, currentWorkflowName, workflowPath, collector); err != nil {
			return err
		}
	}

	dispatchWorkflowValidationLog.Printf("Dispatch workflow validation completed: error_count=%d, total_workflows=%d", collector.Count(), len(config.Workflows))

	return collector.FormattedError("dispatch-workflow")
}

func validateDispatchWorkflowTarget(workflowName, currentWorkflowName, workflowPath string, collector *ErrorCollector) error {
	dispatchWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)
	if workflowName == currentWorkflowName {
		selfRefErr := fmt.Errorf("dispatch-workflow: self-reference not allowed (workflow '%s' cannot dispatch itself)\n\nA workflow cannot trigger itself to prevent infinite loops.\nIf you need recurring execution, use a schedule trigger or workflow_dispatch instead", workflowName)
		return collector.Add(selfRefErr)
	}
	fileResult, err := findWorkflowFile(workflowName, workflowPath)
	if err != nil {
		return collector.Add(fmt.Errorf("dispatch-workflow: error finding workflow '%s': %w", workflowName, err))
	}
	if !fileResult.mdExists && !fileResult.lockExists && !fileResult.ymlExists {
		return collector.Add(dispatchWorkflowNotFoundError(workflowName, workflowPath))
	}
	workflowContent, workflowFile, handled, err := readDispatchWorkflowFile(workflowName, fileResult)
	if err != nil {
		return collector.Add(err)
	}
	if handled {
		return nil
	}
	return validateDispatchWorkflowContent(workflowName, workflowFile, workflowContent, collector)
}

func dispatchWorkflowNotFoundError(workflowName, workflowPath string) error {
	currentDir := filepath.Dir(workflowPath)
	githubDir := filepath.Dir(currentDir)
	repoRoot := filepath.Dir(githubDir)
	workflowsDir := filepath.Join(repoRoot, constants.GetWorkflowDir())
	return fmt.Errorf("dispatch-workflow: workflow '%s' not found in %s\n\nChecked for: %s.md, %s.lock.yml, %s.yml\n\nTo fix:\n1. Verify the workflow file exists in %s/\n2. Ensure the filename matches exactly (case-sensitive)\n3. Use the filename without extension in your configuration", workflowName, workflowsDir, workflowName, workflowName, workflowName, workflowsDir)
}

func readDispatchWorkflowFile(workflowName string, fileResult *findWorkflowFileResult) ([]byte, string, bool, error) {
	if fileResult.lockExists {
		content, err := os.ReadFile(fileResult.lockPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
		return content, fileResult.lockPath, false, dispatchWorkflowReadError(fileResult.lockPath, err)
	}
	if fileResult.ymlExists {
		content, err := os.ReadFile(fileResult.ymlPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
		return content, fileResult.ymlPath, false, dispatchWorkflowReadError(fileResult.ymlPath, err)
	}
	mdHasDispatch, checkErr := mdHasWorkflowDispatch(fileResult.mdPath)
	if checkErr != nil {
		return nil, "", true, fmt.Errorf("dispatch-workflow: failed to read workflow source %s: %w", fileResult.mdPath, checkErr)
	}
	if !mdHasDispatch {
		return nil, "", true, fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
	}
	dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found .md source at %s with workflow_dispatch trigger)", workflowName, fileResult.mdPath)
	return nil, "", true, nil
}

func dispatchWorkflowReadError(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", path, err)
}

func validateDispatchWorkflowContent(workflowName, workflowFile string, workflowContent []byte, collector *ErrorCollector) error {
	var workflow map[string]any
	if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
		return collector.Add(fmt.Errorf("dispatch-workflow: failed to parse workflow file %s: %w", workflowFile, err))
	}
	onSection, hasOn := workflow["on"]
	if !hasOn {
		return collector.Add(fmt.Errorf("dispatch-workflow: workflow '%s' does not have an 'on' trigger section", workflowName))
	}
	if !containsWorkflowDispatch(onSection) {
		return collector.Add(fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName))
	}
	dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found in %s)", workflowName, workflowFile)
	return nil
}

func (c *Compiler) shouldSkipLocalDispatchWorkflowValidation(targetRepoSlug string) bool {
	trimmed := strings.TrimSpace(targetRepoSlug)
	if trimmed == "" {
		return false
	}

	normalized := strings.ReplaceAll(trimmed, " ", "")
	if normalized == "${{github.repository}}" {
		return false
	}

	if strings.Contains(normalized, "${{") || strings.Contains(normalized, "}}") {
		return false
	}

	targetOwner, targetRepo, ok := parseRepoSlugLiteral(trimmed)
	if !ok {
		return false
	}

	currentOwner, currentRepo, ok := parseRepoSlugLiteral(strings.TrimSpace(c.GetRepositorySlug()))
	if ok && strings.EqualFold(targetOwner, currentOwner) && strings.EqualFold(targetRepo, currentRepo) {
		return false
	}

	return true
}

func parseRepoSlugLiteral(slug string) (string, string, bool) {
	// Reject any whitespace to keep target-repo parsing strict and unambiguous.
	if slug == "" || strings.ContainsAny(slug, " \t\r\n") {
		return "", "", false
	}

	owner, repo, err := repoutil.SplitRepoSlug(slug)
	if err != nil {
		return "", "", false
	}

	return owner, repo, true
}

// extractWorkflowDispatchInputs parses a workflow file and extracts the workflow_dispatch inputs schema
// Returns a map of input definitions that can be used to generate MCP tool schemas
func extractWorkflowDispatchInputs(workflowPath string) (map[string]any, error) {
	dispatchWorkflowValidationLog.Printf("Extracting workflow_dispatch inputs from: %s", workflowPath)
	return extractInputsFromYAML(workflowPath, "workflow_dispatch")
}

// containsWorkflowDispatch reports whether the given 'on:' section value includes
// a workflow_dispatch trigger.  It handles the three GitHub Actions forms:
//   - string:     "on: workflow_dispatch"
//   - []any:      "on: [push, workflow_dispatch]"
//   - map[string]any: "on:\n  workflow_dispatch: ..."
func containsWorkflowDispatch(onSection any) bool {
	return containsTrigger(onSection, "workflow_dispatch")
}
