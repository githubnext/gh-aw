package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionsBuildLog = logger.New("cli:actions_build")

// ActionsBuildCommand builds all custom GitHub Actions by bundling JavaScript dependencies
func ActionsBuildCommand() error {
	actionsDir := "actions"

	actionsBuildLog.Print("Starting actions build")

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Building all actions with esbuild..."))

	// Check if actions/package.json exists
	packageJSON := filepath.Join(actionsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("actions/package.json not found - run 'npm init' in actions/ directory")
	}

	// Check if node_modules exists, if not run npm install
	nodeModules := filepath.Join(actionsDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("📦 Installing dependencies..."))
		if err := runCommand(actionsDir, "npm", "install"); err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
	}

	// Run the build script
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("🔨 Building actions with esbuild..."))
	if err := runCommand(actionsDir, "npm", "run", "build"); err != nil {
		return fmt.Errorf("failed to build actions: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✨ All actions built successfully"))
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

// runCommand executes a command in the specified directory
func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr // Redirect stdout to stderr to maintain console formatting
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
