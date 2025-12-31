// Package cli provides configuration validation for workflow compilation.
//
// This file contains functions that validate compilation configuration before
// the compilation process begins, ensuring that all flags and parameters are
// valid and compatible.
//
// # Organization Rationale
//
// These configuration validation functions are grouped here because they:
//   - Validate pre-compilation configuration
//   - Are independent of compilation logic
//   - Have a clear domain focus (configuration validation)
//   - Enable early error detection before expensive operations
//
// # Key Functions
//
// Configuration Validation:
//   - validateWorkflowDirectory() - Validates workflow directory path
//   - setupWorkflowDirectory() - Sets up and validates workflow directory
//
// These functions abstract configuration validation, allowing the main compile
// orchestrator to focus on coordination while these handle validation logic.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var compileConfigValidatorLog = logger.New("cli:compile_config_validator")

// validateWorkflowDirectory validates that the workflow directory exists
func validateWorkflowDirectory(workflowDir string) error {
	compileConfigValidatorLog.Printf("Validating workflow directory: %s", workflowDir)
	
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		// Get git root for better error message
		gitRoot, gitErr := findGitRoot()
		if gitErr != nil {
			return fmt.Errorf("workflow directory %s does not exist", workflowDir)
		}
		return fmt.Errorf("the %s directory does not exist in git root (%s)", filepath.Base(workflowDir), gitRoot)
	}
	
	compileConfigValidatorLog.Print("Workflow directory exists")
	return nil
}

// setupWorkflowDirectory sets up the workflow directory path, using defaults if needed
// Returns the absolute path to the workflows directory
func setupWorkflowDirectory(workflowDir string, gitRoot string) (string, error) {
	compileConfigValidatorLog.Printf("Setting up workflow directory: dir=%s, gitRoot=%s", workflowDir, gitRoot)
	
	// Use default if not specified
	if workflowDir == "" {
		workflowDir = ".github/workflows"
		compileConfigValidatorLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
		compileConfigValidatorLog.Printf("Using custom workflow directory: %s", workflowDir)
	}
	
	// Build absolute path
	absWorkflowDir := filepath.Join(gitRoot, workflowDir)
	
	// Validate it exists
	if err := validateWorkflowDirectory(absWorkflowDir); err != nil {
		return "", err
	}
	
	compileConfigValidatorLog.Printf("Workflow directory setup complete: %s", absWorkflowDir)
	return absWorkflowDir, nil
}
