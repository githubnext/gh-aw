package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeJobsDebugLog = logger.New("workflow:safe_jobs_debug")

// ExtractScriptConfig holds configuration for extracting scripts from safe-output jobs
type ExtractScriptConfig struct {
	WorkflowName string // Name of the workflow (for directory naming)
	WorkflowPath string // Path to the workflow markdown file
	JobName      string // Name of the safe-output job
	StepIndex    int    // Index of the step within the job
	StepName     string // Name of the step (for file naming)
	Script       string // The JavaScript script content
}

// ExtractScriptFromSafeJob extracts a JavaScript script from a github-script action step
// and writes it to an external .cjs file for easier local debugging and testing.
// The script is still inlined in the compiled YAML, but the external file provides
// a convenient way to debug and modify the script locally.
//
// Returns the absolute path to the written script file, or an error if writing fails.
func (c *Compiler) ExtractScriptFromSafeJob(config ExtractScriptConfig) (string, error) {
	if config.Script == "" {
		return "", nil
	}

	safeJobsDebugLog.Printf("Extracting script for job=%s, step=%s", config.JobName, config.StepName)

	// Determine the output directory based on workflow path
	workflowDir := filepath.Dir(config.WorkflowPath)
	scriptsDir := filepath.Join(workflowDir, ".gh-aw", "scripts", sanitizeForFilename(config.WorkflowName))

	// Create scripts directory if it doesn't exist
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Generate filename: <job-name>_<step-index>_<step-name>.cjs
	// Sanitize names for filesystem
	sanitizedStepName := sanitizeForFilename(config.StepName)
	filename := fmt.Sprintf("%s_%d_%s.cjs", sanitizeForFilename(config.JobName), config.StepIndex, sanitizedStepName)
	scriptPath := filepath.Join(scriptsDir, filename)

	// Add a header comment to the script file for context
	header := fmt.Sprintf(`// GitHub Agentic Workflow Script
// Workflow: %s
// Job: %s
// Step: %s (index %d)
// 
// This file was automatically extracted from the workflow's safe-output job
// for easier local debugging and testing. The script is still inlined in the
// compiled .lock.yml file, but you can use this file to:
//   - Test the script locally with Node.js
//   - Debug with a JavaScript debugger
//   - Get IDE support (syntax highlighting, linting, etc.)
//
// To test locally:
//   node %s
//
// Note: You may need to mock environment variables and the agent output file.

`, config.WorkflowName, config.JobName, config.StepName, config.StepIndex, filename)

	fullScript := header + config.Script

	// Write the script to file
	if err := os.WriteFile(scriptPath, []byte(fullScript), 0644); err != nil {
		return "", fmt.Errorf("failed to write script file: %w", err)
	}

	safeJobsDebugLog.Printf("Wrote script to: %s (%d bytes)", scriptPath, len(fullScript))

	return scriptPath, nil
}

// extractScriptFromStep checks if a step uses github-script action and extracts the script
// to an external file for debugging. Returns nil if extraction succeeds or isn't needed.
func (c *Compiler) extractScriptFromStep(stepMap map[string]any, workflowName, workflowPath, jobName string, stepIndex int) error {
	// Check if this step uses github-script action
	uses, hasUses := stepMap["uses"]
	if !hasUses {
		return nil // Not a uses step, skip
	}

	usesStr, ok := uses.(string)
	if !ok || !strings.Contains(usesStr, "github-script") {
		return nil // Not a github-script action, skip
	}

	// Get the 'with' section
	withSection, hasWith := stepMap["with"]
	if !hasWith {
		return nil // No with section, skip
	}

	withMap, ok := withSection.(map[string]any)
	if !ok {
		return nil // Invalid with section, skip
	}

	// Get the script content
	script, hasScript := withMap["script"]
	if !hasScript {
		return nil // No script, skip
	}

	scriptStr, ok := script.(string)
	if !ok || scriptStr == "" {
		return nil // Invalid or empty script, skip
	}

	// Get step name for the filename
	stepName := "step"
	if name, hasName := stepMap["name"]; hasName {
		if nameStr, ok := name.(string); ok && nameStr != "" {
			stepName = nameStr
		}
	}

	// Extract the script to file
	config := ExtractScriptConfig{
		WorkflowName: workflowName,
		WorkflowPath: workflowPath,
		JobName:      jobName,
		StepIndex:    stepIndex,
		StepName:     stepName,
		Script:       scriptStr,
	}

	scriptPath, err := c.ExtractScriptFromSafeJob(config)
	if err != nil {
		return err
	}

	safeJobsDebugLog.Printf("Extracted script for %s/%s to: %s", jobName, stepName, scriptPath)
	return nil
}

// sanitizeForFilename converts a string to a safe filename by replacing special characters
func sanitizeForFilename(s string) string {
	// Replace spaces and special characters with underscores
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.ToLower(s)

	// Remove any leading/trailing underscores or dots
	s = strings.Trim(s, "_.")

	// Replace multiple consecutive underscores with a single one
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	return s
}
