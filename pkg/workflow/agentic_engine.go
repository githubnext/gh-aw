package workflow

import (
	"fmt"
	"strings"
	"sync"
)

// GitHubActionStep represents the YAML lines for a single step in a GitHub Actions workflow
type GitHubActionStep []string

// AgenticEngine represents an AI engine that can be used to execute agentic workflows
type AgenticEngine interface {
	// GetID returns the unique identifier for this engine
	GetID() string

	// GetDisplayName returns the human-readable name for this engine
	GetDisplayName() string

	// GetDescription returns a description of this engine's capabilities
	GetDescription() string

	// IsExperimental returns true if this engine is experimental
	IsExperimental() bool

	// SupportsToolsWhitelist returns true if this engine supports MCP tool allow-listing
	SupportsToolsWhitelist() bool

	// SupportsHTTPTransport returns true if this engine supports HTTP transport for MCP servers
	SupportsHTTPTransport() bool

	// SupportsMaxTurns returns true if this engine supports the max-turns feature
	SupportsMaxTurns() bool

	// GetDeclaredOutputFiles returns a list of output files that this engine may produce
	// These files will be automatically uploaded as artifacts if they exist
	GetDeclaredOutputFiles() []string

	// GetInstallationSteps returns the GitHub Actions steps needed to install this engine
	GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep

	// GetExecutionConfig returns the configuration for executing this engine
	GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig, hasOutput bool) ExecutionConfig

	// RenderMCPConfig renders the MCP configuration for this engine to the given YAML builder
	RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string)

	// ParseLogMetrics extracts metrics from engine-specific log content
	ParseLogMetrics(logContent string, verbose bool) LogMetrics
}

// ExecutionConfig contains the configuration for executing an agentic engine
type ExecutionConfig struct {
	// StepName is the name of the GitHub Actions step
	StepName string

	// Command is the shell command to execute (for CLI-based engines)
	Command string

	// Action is the GitHub Action to use (for action-based engines)
	Action string

	// Inputs are the inputs to pass to the action
	Inputs map[string]string

	// Environment variables needed for execution
	Environment map[string]string
}

// BaseEngine provides common functionality for agentic engines
type BaseEngine struct {
	id                     string
	displayName            string
	description            string
	experimental           bool
	supportsToolsWhitelist bool
	supportsHTTPTransport  bool
	supportsMaxTurns       bool
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

func (e *BaseEngine) SupportsToolsWhitelist() bool {
	return e.supportsToolsWhitelist
}

func (e *BaseEngine) SupportsHTTPTransport() bool {
	return e.supportsHTTPTransport
}

func (e *BaseEngine) SupportsMaxTurns() bool {
	return e.supportsMaxTurns
}

// GetDeclaredOutputFiles returns an empty list by default (engines can override)
func (e *BaseEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// EngineRegistry manages available agentic engines
type EngineRegistry struct {
	engines map[string]AgenticEngine
}

var (
	globalRegistry   *EngineRegistry
	registryInitOnce sync.Once
)

// NewEngineRegistry creates a new engine registry with built-in engines
func NewEngineRegistry() *EngineRegistry {
	registry := &EngineRegistry{
		engines: make(map[string]AgenticEngine),
	}

	// Register built-in engines
	registry.Register(NewClaudeEngine())
	registry.Register(NewCodexEngine())

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
func (r *EngineRegistry) Register(engine AgenticEngine) {
	r.engines[engine.GetID()] = engine
}

// GetEngine retrieves an engine by ID
func (r *EngineRegistry) GetEngine(id string) (AgenticEngine, error) {
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

// GetDefaultEngine returns the default engine (Claude)
func (r *EngineRegistry) GetDefaultEngine() AgenticEngine {
	return r.engines["claude"]
}

// GetEngineByPrefix returns an engine that matches the given prefix
// This is useful for backward compatibility with strings like "codex-experimental"
func (r *EngineRegistry) GetEngineByPrefix(prefix string) (AgenticEngine, error) {
	for id, engine := range r.engines {
		if strings.HasPrefix(prefix, id) {
			return engine, nil
		}
	}
	return nil, fmt.Errorf("no engine found matching prefix: %s", prefix)
}

// GetAllEngines returns all registered engines
func (r *EngineRegistry) GetAllEngines() []AgenticEngine {
	var engines []AgenticEngine
	for _, engine := range r.engines {
		engines = append(engines, engine)
	}
	return engines
}
