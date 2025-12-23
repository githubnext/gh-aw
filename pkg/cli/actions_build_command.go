package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var actionsBuildLog = logger.New("cli:actions_build")

// ActionsBuildCommand builds all custom GitHub Actions by bundling JavaScript dependencies
func ActionsBuildCommand() error {
	actionsDir := "actions"

	actionsBuildLog.Print("Starting actions build")

	// Get list of action directories
	actionDirs, err := getActionDirectories(actionsDir)
	if err != nil {
		return err
	}

	if len(actionDirs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No action directories found in actions/"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Building all actions..."))

	// Build each action
	for _, actionName := range actionDirs {
		if err := buildAction(actionsDir, actionName); err != nil {
			return fmt.Errorf("failed to build action %s: %w", actionName, err)
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✨ All actions built successfully (%d actions)", len(actionDirs))))
	return nil
}

// ActionsValidateCommand validates all action.yml files
func ActionsValidateCommand() error {
	actionsDir := "actions"

	actionsBuildLog.Print("Starting actions validation")

	// Get list of action directories
	actionDirs, err := getActionDirectories(actionsDir)
	if err != nil {
		return err
	}

	if len(actionDirs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No action directories found in actions/"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("✅ Validating all actions"))

	allValid := true
	for _, actionName := range actionDirs {
		actionPath := filepath.Join(actionsDir, actionName)
		if err := validateActionYml(actionPath); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("✗ %s/action.yml: %s", actionName, err.Error())))
			allValid = false
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ %s/action.yml is valid", actionName)))
		}
	}

	if !allValid {
		return fmt.Errorf("validation failed for one or more actions")
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✨ All actions valid"))
	return nil
}

// ActionsCleanCommand removes generated index.js files from all actions
func ActionsCleanCommand() error {
	actionsDir := "actions"

	actionsBuildLog.Print("Starting actions cleanup")

	// Get list of action directories
	actionDirs, err := getActionDirectories(actionsDir)
	if err != nil {
		return err
	}

	if len(actionDirs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No action directories found in actions/"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("🧹 Cleaning generated action files"))

	cleanedCount := 0
	for _, actionName := range actionDirs {
		// Clean index.js for actions that use it (except setup-safe-outputs and setup)
		if actionName != "setup-safe-outputs" && actionName != "setup" {
			indexPath := filepath.Join(actionsDir, actionName, "index.js")
			if _, err := os.Stat(indexPath); err == nil {
				if err := os.Remove(indexPath); err != nil {
					return fmt.Errorf("failed to remove %s: %w", indexPath, err)
				}
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Removed %s/index.js", actionName)))
				cleanedCount++
			}
		}

		// Clean js/ directory for setup-safe-outputs
		if actionName == "setup-safe-outputs" {
			jsDir := filepath.Join(actionsDir, actionName, "js")
			if _, err := os.Stat(jsDir); err == nil {
				if err := os.RemoveAll(jsDir); err != nil {
					return fmt.Errorf("failed to remove %s: %w", jsDir, err)
				}
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Removed %s/js/", actionName)))
				cleanedCount++
			}
		}

		// Clean js/ and sh/ directories for setup action
		if actionName == "setup" {
			jsDir := filepath.Join(actionsDir, actionName, "js")
			if _, err := os.Stat(jsDir); err == nil {
				if err := os.RemoveAll(jsDir); err != nil {
					return fmt.Errorf("failed to remove %s: %w", jsDir, err)
				}
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Removed %s/js/", actionName)))
				cleanedCount++
			}

			shDir := filepath.Join(actionsDir, actionName, "sh")
			if _, err := os.Stat(shDir); err == nil {
				if err := os.RemoveAll(shDir); err != nil {
					return fmt.Errorf("failed to remove %s: %w", shDir, err)
				}
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Removed %s/sh/", actionName)))
				cleanedCount++
			}
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✨ Cleanup complete (%d files removed)", cleanedCount)))
	return nil
}

// getActionDirectories returns a sorted list of action directory names
func getActionDirectories(actionsDir string) ([]string, error) {
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("actions/ directory does not exist")
	}

	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read actions directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	sort.Strings(dirs)
	return dirs, nil
}

// validateActionYml validates that an action.yml file exists and contains required fields.
//
// This validation function is co-located with the actions build command because:
//   - It's specific to GitHub Actions custom action structure
//   - It's only called during the actions build process
//   - It validates action metadata before bundling JavaScript
//
// The function validates:
//   - action.yml file exists in the action directory
//   - Required fields are present (name, description, runs)
//   - Basic action metadata structure is valid
//
// This follows the principle that domain-specific validation belongs in domain files.
func validateActionYml(actionPath string) error {
	ymlPath := filepath.Join(actionPath, "action.yml")

	if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
		return fmt.Errorf("action.yml not found")
	}

	content, err := os.ReadFile(ymlPath)
	if err != nil {
		return fmt.Errorf("failed to read action.yml: %w", err)
	}

	contentStr := string(content)

	// Check required fields
	requiredFields := []string{"name:", "description:", "runs:"}
	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			return fmt.Errorf("missing required field '%s'", strings.TrimSuffix(field, ":"))
		}
	}

	// Check that it's either a node20 or composite action
	isNode20 := strings.Contains(contentStr, "using: 'node20'") || strings.Contains(contentStr, "using: \"node20\"")
	isComposite := strings.Contains(contentStr, "using: 'composite'") || strings.Contains(contentStr, "using: \"composite\"")

	if !isNode20 && !isComposite {
		return fmt.Errorf("action must use either 'node20' or 'composite' runtime")
	}

	return nil
}

// buildAction builds a single action by bundling its dependencies
func buildAction(actionsDir, actionName string) error {
	actionsBuildLog.Printf("Building action: %s", actionName)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\n📦 Building action: %s", actionName)))

	actionPath := filepath.Join(actionsDir, actionName)

	// Validate action.yml
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Validating action.yml"))
	if err := validateActionYml(actionPath); err != nil {
		return err
	}

	// Special handling for setup-safe-outputs: copy files instead of embedding
	if actionName == "setup-safe-outputs" {
		return buildSetupSafeOutputsAction(actionsDir, actionName)
	}

	// Special handling for setup: build shell script with embedded files
	if actionName == "setup" {
		return buildSetupAction(actionsDir, actionName)
	}

	srcPath := filepath.Join(actionPath, "src", "index.js")
	outputPath := filepath.Join(actionPath, "index.js")

	// Check if source file exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", srcPath)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Reading source file"))
	sourceContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get dependencies for this action
	dependencies := getActionDependencies(actionName)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Found %d dependencies", len(dependencies))))

	// Get all JavaScript sources
	sources := workflow.GetJavaScriptSources()

	// Read dependency files
	files := make(map[string]string)
	for _, dep := range dependencies {
		if content, ok := sources[dep]; ok {
			files[dep] = content
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    - %s", dep)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("    ⚠ Warning: Could not find %s", dep)))
		}
	}

	// Generate FILES object with embedded content
	filesJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal files: %w", err)
	}

	// Indent the JSON for proper embedding
	indentedJSON := strings.ReplaceAll(string(filesJSON), "\n", "\n  ")
	indentedJSON = "  " + strings.TrimPrefix(indentedJSON, " ")

	// Replace the FILES placeholder in source
	// Match: const FILES = { ... };
	filesRegex := regexp.MustCompile(`(?s)const FILES = \{[^}]*\};`)
	outputContent := filesRegex.ReplaceAllString(string(sourceContent), fmt.Sprintf("const FILES = %s;", strings.TrimSpace(indentedJSON)))

	// Write output file
	if err := os.WriteFile(outputPath, []byte(outputContent), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Built %s", outputPath)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Embedded %d files", len(files))))

	return nil
}

// buildSetupSafeOutputsAction builds the setup-safe-outputs action by copying JavaScript files
func buildSetupSafeOutputsAction(actionsDir, actionName string) error {
	actionPath := filepath.Join(actionsDir, actionName)
	jsDir := filepath.Join(actionPath, "js")

	// Get dependencies for this action
	dependencies := getActionDependencies(actionName)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Found %d dependencies", len(dependencies))))

	// Get all JavaScript sources
	sources := workflow.GetJavaScriptSources()

	// Create js directory if it doesn't exist
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		return fmt.Errorf("failed to create js directory: %w", err)
	}

	// Copy each dependency file to the js directory
	copiedCount := 0
	for _, dep := range dependencies {
		if content, ok := sources[dep]; ok {
			destPath := filepath.Join(jsDir, dep)
			if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dep, err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    - %s", dep)))
			copiedCount++
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("    ⚠ Warning: Could not find %s", dep)))
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Copied %d files to js/", copiedCount)))

	return nil
}

// buildSetupAction builds the setup action by copying JavaScript files to js/ directory
// and shell scripts to sh/ directory
func buildSetupAction(actionsDir, actionName string) error {
	actionPath := filepath.Join(actionsDir, actionName)
	jsDir := filepath.Join(actionPath, "js")
	shDir := filepath.Join(actionPath, "sh")

	// Get dependencies for this action
	dependencies := getActionDependencies(actionName)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Found %d JavaScript dependencies", len(dependencies))))

	// Get all JavaScript sources
	sources := workflow.GetJavaScriptSources()

	// Create js directory if it doesn't exist
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		return fmt.Errorf("failed to create js directory: %w", err)
	}

	// Copy each dependency file to the js directory
	copiedCount := 0
	for _, dep := range dependencies {
		if content, ok := sources[dep]; ok {
			destPath := filepath.Join(jsDir, dep)
			if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dep, err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    - %s", dep)))
			copiedCount++
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("    ⚠ Warning: Could not find %s", dep)))
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Copied %d files to js/", copiedCount)))

	// Get bundled shell scripts
	shellScripts := workflow.GetBundledShellScripts()
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Found %d shell scripts", len(shellScripts))))

	// Create sh directory if it doesn't exist
	if err := os.MkdirAll(shDir, 0755); err != nil {
		return fmt.Errorf("failed to create sh directory: %w", err)
	}

	// Copy each shell script to the sh directory
	shCopiedCount := 0
	for filename, content := range shellScripts {
		destPath := filepath.Join(shDir, filename)
		// Shell scripts should be executable (0755)
		if err := os.WriteFile(destPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("    - %s", filename)))
		shCopiedCount++
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Copied %d shell scripts to sh/", shCopiedCount)))

	return nil
}

// getActionDependencies returns the list of JavaScript dependencies for an action
// This mapping defines which files from pkg/workflow/js/ are needed for each action
func getActionDependencies(actionName string) []string {
	// For setup, use the dynamic script discovery
	// This ensures all .cjs files are included automatically
	if actionName == "setup" {
		return workflow.GetAllScriptFilenames()
	}

	// Static dependencies for other actions
	dependencyMap := map[string][]string{
		"setup-safe-outputs": {
			"safe_outputs_mcp_server.cjs",
			"safe_outputs_bootstrap.cjs",
			"safe_outputs_tools_loader.cjs",
			"safe_outputs_config.cjs",
			"safe_outputs_handlers.cjs",
			"safe_outputs_tools.json",
			"mcp_server_core.cjs",
			"mcp_logger.cjs",
			"messages.cjs",
		},
		"setup-safe-inputs": {
			"safe_inputs_mcp_server.cjs",
			"safe_inputs_bootstrap.cjs",
			"safe_inputs_config_loader.cjs",
			"safe_inputs_tool_factory.cjs",
			"safe_inputs_validation.cjs",
			"mcp_server_core.cjs",
			"mcp_logger.cjs",
		},
		"noop": {
			"load_agent_output.cjs",
		},
		"minimize_comment": {
			"load_agent_output.cjs",
		},
		"close_issue": {
			"close_entity_helpers.cjs",
		},
		"close_pull_request": {
			"close_entity_helpers.cjs",
		},
		"close_discussion": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
		},
	}

	if deps, ok := dependencyMap[actionName]; ok {
		return deps
	}
	return []string{}
}
