// This file provides compiler initialization and configuration for workflow compilation.
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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileCompilerSetupLog = logger.New("cli:compile_compiler_setup")

// resetActionPinsFile resets the action_pins.json file to an empty state
func resetActionPinsFile() error {
	compileCompilerSetupLog.Print("Resetting action_pins.json to empty state")

	// Get the path to action_pins.json relative to the repository root
	// This assumes the command is run from the repository root
	actionPinsPath := filepath.Join("pkg", "workflow", "data", "action_pins.json")

	// Check if file exists
	if _, err := os.Stat(actionPinsPath); os.IsNotExist(err) {
		compileCompilerSetupLog.Printf("action_pins.json does not exist at %s, skipping reset", actionPinsPath)
		return nil
	}

	// Create empty structure matching the schema
	emptyData := map[string]any{
		"entries": map[string]any{},
	}

	// Marshal with pretty printing
	data, err := json.MarshalIndent(emptyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal empty action pins: %w", err)
	}

	// Add trailing newline for prettier compliance
	data = append(data, '\n')

	// Write the file
	if err := os.WriteFile(actionPinsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write action_pins.json: %w", err)
	}

	compileCompilerSetupLog.Printf("Successfully reset %s to empty state", actionPinsPath)
	return nil
}

// createAndConfigureCompiler creates a new compiler instance and configures it
// based on the provided configuration
func createAndConfigureCompiler(config CompileConfig) *workflow.Compiler {
	compileCompilerSetupLog.Printf("Creating compiler with config: verbose=%v, validate=%v, strict=%v, trialMode=%v",
		config.Verbose, config.Validate, config.Strict, config.TrialMode)

	// Handle force refresh action pins - reset the source action_pins.json file
	if config.ForceRefreshActionPins {
		if err := resetActionPinsFile(); err != nil {
			compileCompilerSetupLog.Printf("Warning: failed to reset action_pins.json: %v", err)
		}
	}

	// Create compiler with auto-detected version and action mode
	// Git root is now auto-detected in NewCompiler() for all compiler instances
	compiler := workflow.NewCompiler(
		workflow.WithVerbose(config.Verbose),
		workflow.WithEngineOverride(config.EngineOverride),
		workflow.WithFailFast(config.FailFast),
	)
	compileCompilerSetupLog.Print("Created compiler instance")

	// Configure compiler flags
	configureCompilerFlags(compiler, config)

	// Set up action mode
	setupActionMode(compiler, config.ActionMode, config.ActionTag)

	// Set up actions repository override if specified
	if config.ActionsRepo != "" {
		compiler.SetActionsRepo(config.ActionsRepo)
		compileCompilerSetupLog.Printf("Actions repository overridden: %s (default: %s)", config.ActionsRepo, workflow.GitHubActionsOrgRepo)
	}

	// Set up repository context
	setupRepositoryContext(compiler, config)

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

	// Set force refresh action pins flag
	compiler.SetForceRefreshActionPins(config.ForceRefreshActionPins)
	if config.ForceRefreshActionPins {
		compileCompilerSetupLog.Print("Force refresh action pins enabled: will clear cache and resolve all actions from GitHub API")
	}

	// Set safe update flag: when set via CLI it disables/skips safe update enforcement
	// regardless of the workflow's strict mode setting.
	compiler.SetApprove(config.Approve)
	if config.Approve {
		compileCompilerSetupLog.Print("Safe update changes approved via --approve-updates flag: skipping safe update enforcement for new restricted secrets or unapproved action additions/removals")
	}

	// Set require docker flag: when set, container image validation fails instead of
	// silently skipping when Docker is not available.
	compiler.SetRequireDocker(config.ValidateImages)
	if config.ValidateImages {
		compileCompilerSetupLog.Print("Container image validation requires Docker (--validate-images flag)")
	}

	// Load pre-cached manifests from file (written by MCP server at startup).
	// These take precedence over git HEAD / filesystem reads for safe update enforcement.
	if config.PriorManifestFile != "" {
		if err := loadPriorManifestFile(compiler, config.PriorManifestFile); err != nil {
			compileCompilerSetupLog.Printf("Failed to load prior manifest file %s: %v (safe update will fall back to git HEAD / filesystem)", config.PriorManifestFile, err)
		}
	}
}

// setupActionMode configures the action script inlining mode
func setupActionMode(compiler *workflow.Compiler, actionMode string, actionTag string) {
	compileCompilerSetupLog.Printf("Setting up action mode: %s, actionTag: %s", actionMode, actionTag)

	// If actionTag is specified, it pins the version used in action/release references.
	// When --action-mode action is explicitly set alongside --action-tag, honour the explicit
	// action mode so that the external actions repo (--actions-repo) is also respected.
	// Without an explicit action mode, --action-tag still defaults to release mode (original behaviour).
	if actionTag != "" {
		compiler.SetActionTag(actionTag)
		if actionMode == string(workflow.ActionModeAction) {
			compileCompilerSetupLog.Printf("--action-tag specified (%s) with --action-mode action, using action mode", actionTag)
			compiler.SetActionMode(workflow.ActionModeAction)
			compileCompilerSetupLog.Printf("Action mode set to: action with tag/SHA: %s", actionTag)
		} else {
			compileCompilerSetupLog.Printf("--action-tag specified (%s), overriding to release mode", actionTag)
			compiler.SetActionMode(workflow.ActionModeRelease)
			compileCompilerSetupLog.Printf("Action mode set to: release with tag/SHA: %s", actionTag)
		}
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
func setupRepositoryContext(compiler *workflow.Compiler, config CompileConfig) {
	compileCompilerSetupLog.Print("Setting up repository context")

	// If a schedule seed is explicitly provided, use it directly
	if config.ScheduleSeed != "" {
		// Validate owner/repo format: must contain exactly one '/' with non-empty parts
		parts := strings.SplitN(config.ScheduleSeed, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			compileCompilerSetupLog.Printf("Invalid --schedule-seed value %q: expected 'owner/repo' format", config.ScheduleSeed)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("--schedule-seed %q is not in 'owner/repo' format; ignoring and falling back to git remote detection", config.ScheduleSeed),
			))
		} else {
			compiler.SetRepositorySlug(config.ScheduleSeed)
			compileCompilerSetupLog.Printf("Repository slug overridden via --schedule-seed: %s", config.ScheduleSeed)
			return
		}
	}

	// Set repository slug for schedule scattering
	repoSlug := getRepositorySlugFromRemote()
	if repoSlug != "" {
		compiler.SetRepositorySlug(repoSlug)
		compileCompilerSetupLog.Printf("Repository slug set: %s", repoSlug)
	} else {
		compileCompilerSetupLog.Print("No repository slug found")
	}
}

// loadPriorManifestFile reads a JSON file containing pre-cached manifests and
// registers each entry with the compiler.  The file must contain a JSON object
// mapping lock-file paths to serialised GHAWManifest objects, as written by
// writePriorManifestFile in the MCP server startup path.
func loadPriorManifestFile(compiler *workflow.Compiler, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read prior manifest file: %w", err)
	}

	var raw map[string]*json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal prior manifest file: %w", err)
	}

	manifests := make(map[string]*workflow.GHAWManifest, len(raw))
	for lockFile, msg := range raw {
		if msg == nil {
			// nil entry means "treat as empty manifest" (new workflow with no prior lock file)
			manifests[lockFile] = nil
			continue
		}
		var m workflow.GHAWManifest
		if err := json.Unmarshal(*msg, &m); err != nil {
			compileCompilerSetupLog.Printf("Skipping malformed manifest for %s: %v", lockFile, err)
			continue
		}
		manifests[lockFile] = &m
	}

	compiler.SetPriorManifests(manifests)
	compileCompilerSetupLog.Printf("Loaded %d pre-cached manifest(s) from %s", len(manifests), filePath)
	return nil
}
