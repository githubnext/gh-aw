package workflow

import (
	"fmt"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
)

// GitHubActionStep represents the YAML lines for a single step in a GitHub Actions workflow
type GitHubActionStep []string

// CodingAgentEngine represents an AI coding agent that can be used as an engine to execute agentic workflows
type CodingAgentEngine interface {
	// GetID returns the unique identifier for this engine
	GetID() string

	// GetDisplayName returns the human-readable name for this engine
	GetDisplayName() string

	// GetDescription returns a description of this engine's capabilities
	GetDescription() string

	// IsExperimental returns true if this engine is experimental
	IsExperimental() bool

	// SupportsToolsAllowlist returns true if this engine supports MCP tool allow-listing
	SupportsToolsAllowlist() bool

	// SupportsHTTPTransport returns true if this engine supports HTTP transport for MCP servers
	SupportsHTTPTransport() bool

	// SupportsMaxTurns returns true if this engine supports the max-turns feature
	SupportsMaxTurns() bool

	// SupportsWebFetch returns true if this engine has built-in support for the web-fetch tool
	SupportsWebFetch() bool

	// SupportsWebSearch returns true if this engine has built-in support for the web-search tool
	SupportsWebSearch() bool

	// GetDeclaredOutputFiles returns a list of output files that this engine may produce
	// These files will be automatically uploaded as artifacts if they exist
	GetDeclaredOutputFiles() []string

	// GetInstallationSteps returns the GitHub Actions steps needed to install this engine
	GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep

	// GetExecutionSteps returns the GitHub Actions steps for executing this engine
	GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep

	// RenderMCPConfig renders the MCP configuration for this engine to the given YAML builder
	RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData)

	// ParseLogMetrics extracts metrics from engine-specific log content
	ParseLogMetrics(logContent string, verbose bool) LogMetrics

	// GetLogParserScriptId returns the name of the JavaScript script to parse logs for this engine
	GetLogParserScriptId() string

	// GetErrorPatterns returns regex patterns for extracting error messages from logs
	GetErrorPatterns() []ErrorPattern

	// GetVersionCommand returns the command to get the version of the agent (e.g., "copilot --version")
	// Returns empty string if the engine does not support version reporting
	GetVersionCommand() string

	// HasDefaultConcurrency returns true if this engine should have default concurrency mode enabled
	// Default concurrency mode applies gh-aw-{engine-id} pattern when no custom concurrency is configured
	HasDefaultConcurrency() bool
}

// ErrorPattern represents a regex pattern for extracting error information from logs
type ErrorPattern struct {
	// Pattern is the regular expression to match log lines
	Pattern string `json:"pattern"`
	// LevelGroup is the capture group index (1-based) that contains the error level (error, warning, etc.)
	// If 0, the level will be inferred from the pattern name or content
	LevelGroup int `json:"level_group"`
	// MessageGroup is the capture group index (1-based) that contains the error message
	// If 0, the entire match will be used as the message
	MessageGroup int `json:"message_group"`
	// Description is a human-readable description of what this pattern matches
	Description string `json:"description"`
}

// BaseEngine provides common functionality for agentic engines
type BaseEngine struct {
	id                     string
	displayName            string
	description            string
	experimental           bool
	supportsToolsAllowlist bool
	supportsHTTPTransport  bool
	supportsMaxTurns       bool
	supportsWebFetch       bool
	supportsWebSearch      bool
	hasDefaultConcurrency  bool
}

func (e *BaseEngine) GetID() string {
	return e.id
}

func (e *BaseEngine) GetDisplayName() string {
	return e.displayName
}

func (e *BaseEngine) GetDescription() string {
	return e.description
}

func (e *BaseEngine) IsExperimental() bool {
	return e.experimental
}

func (e *BaseEngine) SupportsToolsAllowlist() bool {
	return e.supportsToolsAllowlist
}

func (e *BaseEngine) SupportsHTTPTransport() bool {
	return e.supportsHTTPTransport
}

func (e *BaseEngine) SupportsMaxTurns() bool {
	return e.supportsMaxTurns
}

func (e *BaseEngine) SupportsWebFetch() bool {
	return e.supportsWebFetch
}

func (e *BaseEngine) SupportsWebSearch() bool {
	return e.supportsWebSearch
}

// GetDeclaredOutputFiles returns an empty list by default (engines can override)
func (e *BaseEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetErrorPatterns returns an empty list by default (engines can override)
func (e *BaseEngine) GetErrorPatterns() []ErrorPattern {
	return []ErrorPattern{}
}

// GetVersionCommand returns empty string by default (engines can override)
func (e *BaseEngine) GetVersionCommand() string {
	return ""
}

// HasDefaultConcurrency returns the configured value for default concurrency mode
func (e *BaseEngine) HasDefaultConcurrency() bool {
	return e.hasDefaultConcurrency
}

// EngineRegistry manages available agentic engines
type EngineRegistry struct {
	engines map[string]CodingAgentEngine
}

var (
	globalRegistry   *EngineRegistry
	registryInitOnce sync.Once
)

// NewEngineRegistry creates a new engine registry with built-in engines
func NewEngineRegistry() *EngineRegistry {
	registry := &EngineRegistry{
		engines: make(map[string]CodingAgentEngine),
	}

	// Register built-in engines
	registry.Register(NewClaudeEngine())
	registry.Register(NewCodexEngine())
	registry.Register(NewCopilotEngine())
	registry.Register(NewCustomEngine())

	return registry
}

// GetGlobalEngineRegistry returns the singleton engine registry
func GetGlobalEngineRegistry() *EngineRegistry {
	registryInitOnce.Do(func() {
		globalRegistry = NewEngineRegistry()
	})
	return globalRegistry
}

// Register adds an engine to the registry
func (r *EngineRegistry) Register(engine CodingAgentEngine) {
	r.engines[engine.GetID()] = engine
}

// GetEngine retrieves an engine by ID
func (r *EngineRegistry) GetEngine(id string) (CodingAgentEngine, error) {
	engine, exists := r.engines[id]
	if !exists {
		return nil, fmt.Errorf("unknown engine: %s", id)
	}
	return engine, nil
}

// GetSupportedEngines returns a list of all supported engine IDs
func (r *EngineRegistry) GetSupportedEngines() []string {
	var engines []string
	for id := range r.engines {
		engines = append(engines, id)
	}
	return engines
}

// IsValidEngine checks if an engine ID is valid
func (r *EngineRegistry) IsValidEngine(id string) bool {
	_, exists := r.engines[id]
	return exists
}

// GetDefaultEngine returns the default engine (Copilot)
func (r *EngineRegistry) GetDefaultEngine() CodingAgentEngine {
	return r.engines["copilot"]
}

// GetEngineByPrefix returns an engine that matches the given prefix
// This is useful for backward compatibility with strings like "codex-experimental"
func (r *EngineRegistry) GetEngineByPrefix(prefix string) (CodingAgentEngine, error) {
	for id, engine := range r.engines {
		if strings.HasPrefix(prefix, id) {
			return engine, nil
		}
	}
	return nil, fmt.Errorf("no engine found matching prefix: %s", prefix)
}

// GetAllEngines returns all registered engines
func (r *EngineRegistry) GetAllEngines() []CodingAgentEngine {
	var engines []CodingAgentEngine
	for _, engine := range r.engines {
		engines = append(engines, engine)
	}
	return engines
}

// GetCopilotAgentPlaywrightTools returns the list of playwright tools available in the copilot agent
// This matches the tools available in the copilot agent MCP server configuration
// This is a shared function used by all engines for consistent playwright tool configuration
func GetCopilotAgentPlaywrightTools() []any {
	tools := []string{
		"browser_click",
		"browser_close",
		"browser_console_messages",
		"browser_drag",
		"browser_evaluate",
		"browser_file_upload",
		"browser_fill_form",
		"browser_handle_dialog",
		"browser_hover",
		"browser_install",
		"browser_navigate",
		"browser_navigate_back",
		"browser_network_requests",
		"browser_press_key",
		"browser_resize",
		"browser_select_option",
		"browser_snapshot",
		"browser_tabs",
		"browser_take_screenshot",
		"browser_type",
		"browser_wait_for",
	}

	// Convert []string to []any for compatibility with the configuration system
	result := make([]any, len(tools))
	for i, tool := range tools {
		result[i] = tool
	}
	return result
}

// ConvertStepToYAML converts a step map to YAML string with proper indentation
// This is a shared utility function used by all engines and the compiler
func ConvertStepToYAML(stepMap map[string]any) (string, error) {
	// Use OrderMapFields to get ordered MapSlice
	orderedStep := OrderMapFields(stepMap, constants.PriorityStepFields)

	// Wrap in array for step list format and marshal with proper options
	yamlBytes, err := yaml.MarshalWithOptions([]yaml.MapSlice{orderedStep},
		yaml.Indent(2),                        // Use 2-space indentation
		yaml.UseLiteralStyleIfMultiline(true), // Use literal block scalars for multiline strings
	)
	if err != nil {
		return "", fmt.Errorf("failed to marshal step to YAML: %w", err)
	}

	// Convert to string and adjust base indentation to match GitHub Actions format
	yamlStr := string(yamlBytes)

	// Add 6 spaces to the beginning of each line to match GitHub Actions step indentation
	lines := strings.Split(strings.TrimSpace(yamlStr), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
		} else {
			result.WriteString("      " + line + "\n")
		}
	}

	return result.String(), nil
}

// generateLogCaptureStep creates a shared log capture step for any engine
// This reduces code duplication across engines and ensures consistency
func generateLogCaptureStep(logFile string) GitHubActionStep {
	logCaptureLines := []string{
		"      - name: Print agent log",
		"        if: always()",
		"        run: |",
		"          touch " + logFile,
		"          echo \"## Agent Log\" >> $GITHUB_STEP_SUMMARY",
		"          echo '```markdown' >> $GITHUB_STEP_SUMMARY",
		fmt.Sprintf("          cat %s >> $GITHUB_STEP_SUMMARY", logFile),
		"          echo '```' >> $GITHUB_STEP_SUMMARY",
	}
	return GitHubActionStep(logCaptureLines)
}

// ProcessCustomSteps processes custom steps from engine config if they exist
// This reduces code duplication across engines by providing a shared implementation
// for handling custom steps defined in the engine configuration
func ProcessCustomSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := ConvertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

	return steps
}

// AddSafeOutputsEnvToMap adds safe-outputs environment variables to a map
// This is used by engines that build environment as a map (Codex, Copilot)
func AddSafeOutputsEnvToMap(env map[string]string, workflowData *WorkflowData, useQuoting bool) {
	if workflowData.SafeOutputs == nil {
		return
	}

	env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
	}
	if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
		env["GITHUB_AW_TARGET_REPO"] = workflowData.TrialTargetRepo
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		if useQuoting {
			env["GITHUB_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", workflowData.SafeOutputs.UploadAssets.BranchName)
			env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ","))
		} else {
			env["GITHUB_AW_ASSETS_BRANCH"] = workflowData.SafeOutputs.UploadAssets.BranchName
			env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] = strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")
		}
		env["GITHUB_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB)
	}
}

// AddMaxTurnsEnvToMap adds max-turns environment variable to a map if configured
// This is used by engines that support the max-turns configuration (Copilot, Custom)
func AddMaxTurnsEnvToMap(env map[string]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GITHUB_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}
}

// AddCustomEngineEnvToMap adds custom environment variables from engine config to a map
// This is used by all engines that support custom environment variables
func AddCustomEngineEnvToMap(env map[string]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}
}

// AddSafeOutputsEnvToLines adds safe-outputs environment variables to step lines
// This is used by engines that build environment as step lines (Claude)
func AddSafeOutputsEnvToLines(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	*stepLines = append(*stepLines, "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}")

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		*stepLines = append(*stepLines, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"")
	}
	if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_TARGET_REPO: %q", workflowData.TrialTargetRepo))
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_BRANCH: %q", workflowData.SafeOutputs.UploadAssets.BranchName))
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_MAX_SIZE_KB: %d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB))
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_ALLOWED_EXTS: %q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")))
	}
}

// AddMaxTurnsEnvToLines adds max-turns environment variable to step lines if configured
// This is used by engines that build environment as step lines (Claude)
func AddMaxTurnsEnvToLines(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_MAX_TURNS: %s", workflowData.EngineConfig.MaxTurns))
	}
}

// AddCustomEngineEnvToLines adds custom environment variables from engine config to step lines
// This is used by engines that build environment as step lines (Claude)
func AddCustomEngineEnvToLines(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			*stepLines = append(*stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}
}

// AddSafeOutputsEnvToAnyMap adds safe-outputs environment variables to a map[string]any
// This is used by the custom engine which uses map[string]any for environment variables
func AddSafeOutputsEnvToAnyMap(env map[string]any, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
	}
	if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
		env["GITHUB_AW_TARGET_REPO"] = workflowData.TrialTargetRepo
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		env["GITHUB_AW_ASSETS_BRANCH"] = workflowData.SafeOutputs.UploadAssets.BranchName
		env["GITHUB_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB)
		env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] = strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")
	}
}

// AddMaxTurnsEnvToAnyMap adds max-turns environment variable to a map[string]any if configured
// This is used by the custom engine which uses map[string]any for environment variables
func AddMaxTurnsEnvToAnyMap(env map[string]any, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GITHUB_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}
}

// AddCustomEngineEnvToAnyMap adds custom environment variables from engine config to a map[string]any
// This is used by the custom engine which uses map[string]any for environment variables
func AddCustomEngineEnvToAnyMap(env map[string]any, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}
}
