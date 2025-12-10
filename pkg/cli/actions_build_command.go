package cli

import (
	"fmt"
	"os"
	"path/filepath"
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
		indexPath := filepath.Join(actionsDir, actionName, "index.js")
		if _, err := os.Stat(indexPath); err == nil {
			if err := os.Remove(indexPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", indexPath, err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Removed %s/index.js", actionName)))
			cleanedCount++
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

// validateActionYml validates an action.yml file
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

	// Check that it's a node20 action
	if !strings.Contains(contentStr, "using: 'node20'") && !strings.Contains(contentStr, "using: \"node20\"") {
		return fmt.Errorf("action must use 'node20' runtime")
	}

	return nil
}

// buildAction builds a single action by bundling its dependencies using GitHub Script mode
func buildAction(actionsDir, actionName string) error {
	actionsBuildLog.Printf("Building action: %s", actionName)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\n📦 Building action: %s", actionName)))

	actionPath := filepath.Join(actionsDir, actionName)
	srcPath := filepath.Join(actionPath, "src", "index.js")
	outputPath := filepath.Join(actionPath, "index.js")

	// Validate action.yml
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Validating action.yml"))
	if err := validateActionYml(actionPath); err != nil {
		return err
	}

	// Check if source file exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", srcPath)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Reading source file"))
	sourceContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get ALL JavaScript sources - the bundler will figure out what's needed
	allSources := workflow.GetJavaScriptSources()
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Loaded %d source files for bundling", len(allSources))))

	// Bundle using GitHub Script mode (inlines dependencies, removes exports)
	// The bundler will recursively bundle all required dependencies
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Bundling with GitHub Script mode (recursive)"))
	bundled, err := workflow.BundleJavaScriptWithMode(string(sourceContent), allSources, "", workflow.RuntimeModeGitHubScript)
	if err != nil {
		return fmt.Errorf("failed to bundle JavaScript: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputPath, []byte(bundled), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ✓ Built %s", outputPath)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Bundled dependencies inline and removed exports"))

	return nil
}

// getActionDependencies returns the list of JavaScript dependencies for an action
// This mapping defines which files from pkg/workflow/js/ are needed for each action
func getActionDependencies(actionName string) []string {
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
		"minimize-comment": {
			"load_agent_output.cjs",
		},
		"close-issue": {
			"close_entity_helpers.cjs",
		},
		"close-pull-request": {
			"close_entity_helpers.cjs",
		},
		"close-discussion": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
		},
		"add-comment": {
			"add_comment_helpers.cjs",
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
		},
		"create-issue": {
			"expiration_helpers.cjs",
			"generate_footer.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
			"staged_preview.cjs",
			"temporary_id.cjs",
		},
		"add-labels": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
		},
		"create-discussion": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
		},
		"update-issue": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
		},
		"update-pull-request": {
			"generate_footer.cjs",
			"get_repository_url.cjs",
			"get_tracker_id.cjs",
			"load_agent_output.cjs",
			"repo_helpers.cjs",
			"sanitize_label_content.cjs",
		},
	}

	if deps, ok := dependencyMap[actionName]; ok {
		return deps
	}
	return []string{}
}
