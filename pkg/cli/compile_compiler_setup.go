// Package cli provides compiler initialization and configuration for workflow compilation.
//
// This file contains functions that create and configure the workflow compiler
// instance with various settings like validation, strict mode, trial mode, and
// action mode.
//
// # Organization Rationale
//
// These compiler setup functions are grouped here because they:
//   - Handle compiler instance creation and configuration
//   - Set up compilation flags and modes
//   - Have a clear domain focus (compiler configuration)
//   - Keep the main orchestrator focused on workflow processing
//
// # Key Functions
//
// Compiler Creation:
//   - createAndConfigureCompiler() - Creates compiler with full configuration
//
// Configuration:
//   - configureCompilerFlags() - Sets validation, strict mode, trial mode flags
//   - setupActionMode() - Configures action script inlining mode
//   - setupRepositoryContext() - Sets repository slug for schedule scattering
//
// These functions abstract compiler setup, allowing the main compile
// orchestrator to focus on coordination while these handle configuration.
package cli

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var compileCompilerSetupLog = logger.New("cli:compile_compiler_setup")

// createAndConfigureCompiler creates a new compiler instance and configures it
// based on the provided configuration
func createAndConfigureCompiler(config CompileConfig) *workflow.Compiler {
	compileCompilerSetupLog.Printf("Creating compiler with config: verbose=%v, validate=%v, strict=%v, trialMode=%v",
		config.Verbose, config.Validate, config.Strict, config.TrialMode)

	// Create compiler with verbose flag and AI engine override
	compiler := workflow.NewCompiler(config.Verbose, config.EngineOverride, GetVersion())
	compileCompilerSetupLog.Print("Created compiler instance")

	// Configure compiler flags
	configureCompilerFlags(compiler, config)

	// Set up action mode
	setupActionMode(compiler, config.ActionMode, config.ActionTag)

	// Set up repository context
	setupRepositoryContext(compiler)

	return compiler
}

// configureCompilerFlags sets various compilation flags on the compiler
func configureCompilerFlags(compiler *workflow.Compiler, config CompileConfig) {
	compileCompilerSetupLog.Print("Configuring compiler flags")

	// Set validation based on the validate flag (false by default for compatibility)
	compiler.SetSkipValidation(!config.Validate)
	compileCompilerSetupLog.Printf("Validation enabled: %v", config.Validate)

	// Set noEmit flag to validate without generating lock files
	compiler.SetNoEmit(config.NoEmit)
	if config.NoEmit {
		compileCompilerSetupLog.Print("No-emit mode enabled: validating without generating lock files")
	}

	// Set strict mode if specified
	compiler.SetStrictMode(config.Strict)

	// Set trial mode if specified
	if config.TrialMode {
		compileCompilerSetupLog.Printf("Enabling trial mode: repoSlug=%s", config.TrialLogicalRepoSlug)
		compiler.SetTrialMode(true)
		if config.TrialLogicalRepoSlug != "" {
			compiler.SetTrialLogicalRepoSlug(config.TrialLogicalRepoSlug)
		}
	}

	// Set refresh stop time flag
	compiler.SetRefreshStopTime(config.RefreshStopTime)
	if config.RefreshStopTime {
		compileCompilerSetupLog.Print("Stop time refresh enabled: will regenerate stop-after times")
	}
}

// setupActionMode configures the action script inlining mode
func setupActionMode(compiler *workflow.Compiler, actionMode string, actionTag string) {
	compileCompilerSetupLog.Printf("Setting up action mode: %s, actionTag: %s", actionMode, actionTag)

	// If actionTag is specified, override to release mode
	if actionTag != "" {
		compileCompilerSetupLog.Printf("--action-tag specified (%s), overriding to release mode", actionTag)
		compiler.SetActionMode(workflow.ActionModeRelease)
		compiler.SetActionTag(actionTag)
		compileCompilerSetupLog.Printf("Action mode set to: release with tag/SHA: %s", actionTag)
		return
	}

	if actionMode != "" {
		mode := workflow.ActionMode(actionMode)
		if !mode.IsValid() {
			// This should be caught by validation earlier, but log it
			compileCompilerSetupLog.Printf("Invalid action mode '%s', using auto-detection", actionMode)
			mode = workflow.DetectActionMode(GetVersion())
		}
		compiler.SetActionMode(mode)
		compileCompilerSetupLog.Printf("Action mode set to: %s", mode)
	} else {
		// Use auto-detection with version from binary
		mode := workflow.DetectActionMode(GetVersion())
		compiler.SetActionMode(mode)
		compileCompilerSetupLog.Printf("Action mode auto-detected: %s (version: %s)", mode, GetVersion())
	}
}

// setupRepositoryContext sets the repository slug for schedule scattering
func setupRepositoryContext(compiler *workflow.Compiler) {
	compileCompilerSetupLog.Print("Setting up repository context")

	// Set repository slug for schedule scattering
	repoSlug := getRepositorySlugFromRemote()
	if repoSlug != "" {
		compiler.SetRepositorySlug(repoSlug)
		compileCompilerSetupLog.Printf("Repository slug set: %s", repoSlug)
	} else {
		compileCompilerSetupLog.Print("No repository slug found")
	}
}

// validateActionModeConfig validates the action mode configuration
func validateActionModeConfig(actionMode string) error {
	if actionMode == "" {
		return nil
	}

	mode := workflow.ActionMode(actionMode)
	if !mode.IsValid() {
		return fmt.Errorf("invalid action mode '%s'. Must be 'dev', 'release', or 'script'", actionMode)
	}

	return nil
}
