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

	// GetLogFileForParsing returns the log file path to use for JavaScript parsing in the workflow
	// This may be different from the stdout/stderr log file if the engine produces separate detailed logs
	GetLogFileForParsing() string

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

// GetLogFileForParsing returns the default log file path for parsing
// Engines can override this to use engine-specific log files
func (e *BaseEngine) GetLogFileForParsing() string {
	// Default to agent-stdio.log which contains stdout/stderr
	return "/tmp/gh-aw/agent-stdio.log"
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
