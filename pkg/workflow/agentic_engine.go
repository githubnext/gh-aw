package workflow

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var agenticEngineLog = logger.New("workflow:agentic_engine")

// GitHubActionStep represents the YAML lines for a single step in a GitHub Actions workflow
type GitHubActionStep []string

// Interface Segregation Architecture
//
// The agentic engine interfaces follow the Interface Segregation Principle (ISP) to avoid
// forcing implementations to depend on methods they don't use. The architecture uses interface
// composition to provide flexibility while maintaining backward compatibility.
//
// Core Principles:
// 1. Focused interfaces with single responsibilities
// 2. Composition over monolithic interfaces
// 3. Backward compatibility via composite interface
// 4. Optional capabilities via interface type assertions
//
// Interface Hierarchy:
//
//   Engine (core identity - required by all)
//   ├── GetID()
//   ├── GetDisplayName()
//   ├── GetDescription()
//   └── IsExperimental()
//
//   CapabilityProvider (feature detection - optional)
//   ├── SupportsToolsAllowlist()
//   ├── SupportsMaxTurns()
//   ├── SupportsWebSearch()
//   ├── SupportsMaxContinuations()
//   ├── SupportsNativeAgentFile()
//   └── SupportsBareMode()
//
//   WorkflowExecutor (compilation - required)
//   ├── GetDeclaredOutputFiles()
//   ├── GetInstallationSteps()
//   └── GetExecutionSteps()
//
//   MCPConfigProvider (MCP servers - optional)
//   └── RenderMCPConfig()
//
//   LogParser (log analysis - optional)
//   ├── ParseLogMetrics()
//   ├── GetLogParserScriptId()
//   └── GetLogFileForParsing()
//
//   SecurityProvider (security features - optional)
//   ├── GetDefaultDetectionModel()
//   └── GetRequiredSecretNames()
//
//   CodingAgentEngine (composite - backward compatibility)
//   └── Composes all above interfaces
//
// Usage Patterns:
//
// 1. For code that only needs identity information:
//    func processEngine(e Engine) { ... }
//
// 2. For code that needs capability checks:
//    func checkCapabilities(cp CapabilityProvider) { ... }
//
// 3. For backward compatibility (existing code):
//    func compile(engine CodingAgentEngine) { ... }
//
// Implementation:
//
// All engines embed BaseEngine which provides default implementations for all methods.
// Engines can override specific methods to provide custom behavior.
//
// Example:
//   type MyEngine struct {
//       BaseEngine
//   }
//
//   func (e *MyEngine) GetInstallationSteps(...) []GitHubActionStep {
//       // Custom implementation
//   }

// Engine represents the core identity of an AI coding agent
// All engines must implement this interface to provide basic identification
type Engine interface {
	// GetID returns the unique identifier for this engine
	GetID() string

	// GetDisplayName returns the human-readable name for this engine
	GetDisplayName() string

	// GetDescription returns a description of this engine's capabilities
	GetDescription() string

	// IsExperimental returns true if this engine is experimental
	IsExperimental() bool
}

// CapabilityProvider detects what capabilities an engine supports
// Engines can optionally implement this to indicate feature support
type CapabilityProvider interface {
	// SupportsToolsAllowlist returns true if this engine supports MCP tool allow-listing
	SupportsToolsAllowlist() bool

	// SupportsMaxTurns returns true if this engine supports the max-turns feature
	SupportsMaxTurns() bool

	// SupportsWebSearch returns true if this engine has built-in support for the web-search tool
	SupportsWebSearch() bool

	// SupportsMaxContinuations returns true if this engine supports the max-continuations feature
	// When true, max-continuations > 1 enables autopilot/multi-run mode for the engine
	SupportsMaxContinuations() bool

	// SupportsNativeAgentFile returns true if this engine handles agent-file imports natively
	// in its own execution steps (reading the file, stripping frontmatter, and prepending the
	// content to the prompt at runtime).  When false, the compiler is responsible for including
	// the agent file content in prompt.txt during the activation job so that the engine just
	// reads the standard /tmp/gh-aw/aw-prompts/prompt.txt as usual.
	SupportsNativeAgentFile() bool

	// SupportsBareMode returns true if this engine supports the bare mode feature
	// (engine.bare: true), which suppresses automatic loading of context and custom
	// instructions. When false, specifying bare: true emits a warning and has no effect.
	SupportsBareMode() bool
}

// WorkflowExecutor handles workflow compilation and execution
// All engines must implement this to generate GitHub Actions steps
type WorkflowExecutor interface {
	// GetDeclaredOutputFiles returns a list of output files that this engine may produce
	// These files will be automatically uploaded as artifacts if they exist
	GetDeclaredOutputFiles() []string

	// GetInstallationSteps returns the GitHub Actions steps needed to install this engine
	GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep

	// GetSecretValidationStep returns the step that validates required secrets are available.
	// This step is added to the activation job before context variable validation.
	// Returns an empty GitHubActionStep if no secret validation is needed.
	GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep

	// GetExecutionSteps returns the GitHub Actions steps for executing this engine
	GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep

	// GetFirewallLogsCollectionStep returns steps that collect firewall-related log files
	// before secret redaction runs. Engines that copy session or firewall state files should
	// override this; the default implementation returns an empty slice.
	GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep

	// GetAPMTarget returns the APM target value to use when packing dependencies with
	// microsoft/apm-action. Supported values are "copilot", "claude", and "all".
	// The default implementation returns "all" (packs all primitive types).
	GetAPMTarget() string

	// GetPreBundleSteps returns GitHub Actions steps that must run before the unified artifact
	// upload. These steps are injected prior to secret redaction so any engine-produced files
	// are moved into /tmp/gh-aw/ where they will be scanned and picked up by the artifact
	// upload under the correct common-ancestor path.
	// The default implementation returns an empty slice.
	GetPreBundleSteps(workflowData *WorkflowData) []GitHubActionStep
}

// MCPConfigProvider handles MCP (Model Context Protocol) configuration
// Engines that support MCP servers should implement this
type MCPConfigProvider interface {
	// RenderMCPConfig renders the MCP configuration for this engine to the given YAML builder
	RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error
}

// LogParser handles parsing and analyzing engine logs
// Engines can optionally implement this to provide detailed log parsing
type LogParser interface {
	// ParseLogMetrics extracts metrics from engine-specific log content
	ParseLogMetrics(logContent string, verbose bool) LogMetrics

	// GetLogParserScriptId returns the name of the JavaScript script to parse logs for this engine
	GetLogParserScriptId() string

	// GetLogFileForParsing returns the log file path to use for JavaScript parsing in the workflow
	// This may be different from the stdout/stderr log file if the engine produces separate detailed logs
	GetLogFileForParsing() string
}

// SecurityProvider handles security-related configuration
// Engines can optionally implement this to provide security features
type SecurityProvider interface {
	// GetDefaultDetectionModel returns the default model to use for threat detection
	// If empty, no default model is applied and the engine uses its standard default
	GetDefaultDetectionModel() string

	// GetRequiredSecretNames returns the list of secret names that this engine needs for execution
	// This includes engine-specific auth tokens and the MCP gateway API key when MCP servers are present
	// Returns: slice of secret names (e.g., ["COPILOT_GITHUB_TOKEN", "MCP_GATEWAY_API_KEY"])
	GetRequiredSecretNames(workflowData *WorkflowData) []string
}

// ModelEnvVarProvider is implemented by engines whose CLIs natively read a specific
// environment variable for model selection (e.g., COPILOT_MODEL, ANTHROPIC_MODEL).
// The default implementation in BaseEngine returns "" (no native env var).
type ModelEnvVarProvider interface {
	// GetModelEnvVarName returns the name of the native environment variable the CLI
	// uses for model selection (e.g., "COPILOT_MODEL", "ANTHROPIC_MODEL", "GEMINI_MODEL").
	// Returns an empty string if the engine does not support a native model env var.
	GetModelEnvVarName() string
}

// AgentFileProvider is an optional interface implemented by engines that have
// engine-specific instruction or configuration files that should be treated as
// security-sensitive manifests.  The compiler uses these lists to extend the
// global manifest-file protection so that engine-specific files (e.g. CLAUDE.md,
// .claude/) are automatically protected alongside dependency manifests.
type AgentFileProvider interface {
	// GetAgentManifestFiles returns the basenames of files that are specific to
	// this engine's instruction / configuration format (e.g. "CLAUDE.md").
	// Matching is by filename only, regardless of directory depth.
	GetAgentManifestFiles() []string

	// GetAgentManifestPathPrefixes returns path prefixes (relative to the repo
	// root) for directories that contain engine-specific configuration.
	// Any file whose diff path starts with one of these prefixes is treated as a
	// protected file (e.g. ".claude/").
	GetAgentManifestPathPrefixes() []string
}

// ConfigRenderer is an optional hook that runtimes may implement to emit generated
// config files or metadata before execution steps run.
type ConfigRenderer interface {
	// RenderConfig optionally generates runtime config files or metadata.
	// Returns a slice of GitHub Actions steps that write config to disk.
	// Implementations that don't need config files should return nil, nil.
	RenderConfig(target *ResolvedEngineTarget) ([]map[string]any, error)
}

// DriverProvider is an optional interface implemented by engines that provide a
// JavaScript driver script to wrap CLI execution with retry and recovery logic.
// The driver is placed in the setup actions directory and executed via Node.js
// as a transparent subprocess wrapper around the engine CLI.
type DriverProvider interface {
	// GetDriverScriptName returns the filename of the JavaScript driver script
	// (located in the setup actions directory) used to wrap CLI execution.
	// Returns an empty string if no driver is needed.
	GetDriverScriptName() string
}

// CodingAgentEngine is a composite interface that combines all focused interfaces
// This maintains backward compatibility with existing code while allowing more flexibility
// Implementations can choose to implement only the interfaces they need by embedding BaseEngine
type CodingAgentEngine interface {
	Engine
	CapabilityProvider
	WorkflowExecutor
	MCPConfigProvider
	LogParser
	SecurityProvider
	ModelEnvVarProvider
	ConfigRenderer
}

// BaseEngine provides common functionality for agentic engines
type BaseEngine struct {
	id                       string
	displayName              string
	description              string
	experimental             bool
	supportsToolsAllowlist   bool
	supportsMaxTurns         bool
	supportsMaxContinuations bool
	supportsWebSearch        bool
	supportsNativeAgentFile  bool
	supportsBareMode         bool
	llmGatewayPort           int
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

func (e *BaseEngine) SupportsMaxTurns() bool {
	return e.supportsMaxTurns
}

func (e *BaseEngine) SupportsWebSearch() bool {
	return e.supportsWebSearch
}

func (e *BaseEngine) SupportsMaxContinuations() bool {
	return e.supportsMaxContinuations
}

func (e *BaseEngine) SupportsNativeAgentFile() bool {
	return e.supportsNativeAgentFile
}

func (e *BaseEngine) SupportsBareMode() bool {
	return e.supportsBareMode
}

func (e *BaseEngine) getLLMGatewayPort() int {
	return e.llmGatewayPort
}

// GetDeclaredOutputFiles returns an empty list by default (engines can override)
func (e *BaseEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetDefaultDetectionModel returns empty string by default (no default model)
// Engines can override this to provide a cost-effective default for detection jobs
func (e *BaseEngine) GetDefaultDetectionModel() string {
	return ""
}

// GetModelEnvVarName returns empty string by default (no native model env var).
// Engines whose CLI natively supports a model env var (e.g. COPILOT_MODEL, ANTHROPIC_MODEL)
// should override this to return that variable name.
func (e *BaseEngine) GetModelEnvVarName() string {
	return ""
}

// GetLogFileForParsing returns the default log file path for parsing
// Engines can override this to use engine-specific log files
func (e *BaseEngine) GetLogFileForParsing() string {
	// Default to agent-stdio.log which contains stdout/stderr
	return "/tmp/gh-aw/agent-stdio.log"
}

// GetRequiredSecretNames returns an empty list by default
// Engines must override this to specify their required secrets
func (e *BaseEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	return []string{}
}

// GetSecretValidationStep returns an empty step by default.
// Engines that require secret validation must override this method.
func (e *BaseEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return GitHubActionStep{}
}

// GetFirewallLogsCollectionStep returns an empty slice by default.
// Engines that need to copy session or firewall state files before secret redaction should override this.
func (e *BaseEngine) GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{}
}

// GetAPMTarget returns "all" by default (packs all primitive types).
// CopilotEngine overrides this to return "copilot"; ClaudeEngine overrides to return "claude".
func (e *BaseEngine) GetAPMTarget() string {
	return "all"
}

// GetPreBundleSteps returns an empty slice by default.
// Engines that need to relocate output files into /tmp/gh-aw/ before the unified artifact
// upload should override this method.
func (e *BaseEngine) GetPreBundleSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{}
}

// ParseLogMetrics provides a default no-op implementation for log parsing
// Engines can override this to provide detailed log parsing and metrics extraction
func (e *BaseEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return LogMetrics{}
}

// GetLogParserScriptId returns empty string by default (no JavaScript parser)
// Engines can override this to provide a JavaScript parser for log analysis
func (e *BaseEngine) GetLogParserScriptId() string {
	return ""
}

// RenderMCPConfig provides a default no-op implementation for MCP configuration
// Engines can override this to provide custom MCP server configuration
func (e *BaseEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	// Default implementation does nothing - engines that support MCP should override this
	return nil
}

// GetAgentManifestFiles returns nil by default (no engine-specific manifest files).
// Engines with dedicated instruction files (e.g. CLAUDE.md, AGENTS.md) should override this.
func (e *BaseEngine) GetAgentManifestFiles() []string {
	return nil
}

// GetAgentManifestPathPrefixes returns nil by default (no engine-specific config directories).
// Engines with a dedicated config directory (e.g. .claude/) should override this.
func (e *BaseEngine) GetAgentManifestPathPrefixes() []string {
	return nil
}

// RenderConfig returns nil by default — engines that need to write config files before
// execution (e.g. provider/model config files) should override this method.
func (e *BaseEngine) RenderConfig(_ *ResolvedEngineTarget) ([]map[string]any, error) {
	return nil, nil
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
	agenticEngineLog.Print("Creating new engine registry")

	registry := &EngineRegistry{
		engines: make(map[string]CodingAgentEngine),
	}

	// Register built-in engines
	registry.Register(NewClaudeEngine())
	registry.Register(NewCodexEngine())
	registry.Register(NewCopilotEngine())
	registry.Register(NewGeminiEngine())
	registry.Register(NewOpenCodeEngine())
	registry.Register(NewCrushEngine())

	agenticEngineLog.Printf("Registered %d engines", len(registry.engines))
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
	type portProvider interface{ getLLMGatewayPort() int }
	if p, ok := engine.(portProvider); ok && p.getLLMGatewayPort() < 0 {
		panic(fmt.Sprintf("engine '%s': llmGatewayPort must be >= 0, got %d", engine.GetID(), p.getLLMGatewayPort()))
	}
	agenticEngineLog.Printf("Registering engine: id=%s, name=%s", engine.GetID(), engine.GetDisplayName())
	r.engines[engine.GetID()] = engine
}

// GetEngine retrieves an engine by ID
func (r *EngineRegistry) GetEngine(id string) (CodingAgentEngine, error) {
	agenticEngineLog.Printf("Looking up engine: id=%s", id)
	engine, exists := r.engines[id]
	if !exists {
		agenticEngineLog.Printf("Engine not found: id=%s", id)
		return nil, fmt.Errorf("unknown engine: %s", id)
	}
	agenticEngineLog.Printf("Found engine: id=%s, name=%s", id, engine.GetDisplayName())
	return engine, nil
}

// GetSupportedEngines returns a list of all supported engine IDs
func (r *EngineRegistry) GetSupportedEngines() []string {
	agenticEngineLog.Print("Getting list of supported engines")
	var engines []string
	for id := range r.engines {
		engines = append(engines, id)
	}
	sort.Strings(engines)
	return engines
}

// IsValidEngine checks if an engine ID is valid
func (r *EngineRegistry) IsValidEngine(id string) bool {
	_, exists := r.engines[id]
	return exists
}

// GetDefaultEngine returns the default engine configured by constants.DefaultEngine
func (r *EngineRegistry) GetDefaultEngine() CodingAgentEngine {
	return r.engines[string(constants.DefaultEngine)]
}

// GetAllAgentManifestFolders returns the union of all engines' GetAgentManifestPathPrefixes()
// with trailing slashes stripped, plus ".agents" as the gh-aw platform agent directory.
// The returned list is sorted and deduplicated, making the engine implementations the
// single source of truth for which directories the save/restore scripts protect.
func (r *EngineRegistry) GetAllAgentManifestFolders() []string {
	seen := map[string]bool{}
	var result []string
	for _, engine := range r.engines {
		provider, ok := engine.(AgentFileProvider)
		if !ok {
			continue
		}
		for _, prefix := range provider.GetAgentManifestPathPrefixes() {
			folder := strings.TrimSuffix(prefix, "/")
			if folder != "" && !seen[folder] {
				seen[folder] = true
				result = append(result, folder)
			}
		}
	}
	// Always include .agents — the gh-aw platform agent directory.
	// It is not owned by any specific engine but must always be snapshotted.
	if !seen[".agents"] {
		result = append(result, ".agents")
	}
	sort.Strings(result)
	return result
}

// GetAllAgentManifestFiles returns the union of all engines' GetAgentManifestFiles().
// The returned list is sorted and deduplicated, making the engine implementations the
// single source of truth for which root-level instruction files the save/restore scripts protect.
func (r *EngineRegistry) GetAllAgentManifestFiles() []string {
	seen := map[string]bool{}
	var result []string
	for _, engine := range r.engines {
		provider, ok := engine.(AgentFileProvider)
		if !ok {
			continue
		}
		for _, file := range provider.GetAgentManifestFiles() {
			if !seen[file] {
				seen[file] = true
				result = append(result, file)
			}
		}
	}
	sort.Strings(result)
	return result
}

// GetEngineByPrefix returns an engine that matches the given prefix
// This is useful for backward compatibility with strings like "codex-experimental"
func (r *EngineRegistry) GetEngineByPrefix(prefix string) (CodingAgentEngine, error) {
	agenticEngineLog.Printf("Looking up engine by prefix: %s", prefix)
	type engineCandidate struct {
		id     string
		engine CodingAgentEngine
	}
	var candidates []engineCandidate
	for id, engine := range r.engines {
		if strings.HasPrefix(prefix, id) {
			candidates = append(candidates, engineCandidate{id, engine})
		}
	}
	if len(candidates) == 0 {
		agenticEngineLog.Printf("No engine found matching prefix: %s", prefix)
		return nil, fmt.Errorf("no engine found matching prefix: %s", prefix)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].id < candidates[j].id })
	agenticEngineLog.Printf("Found %d engine candidate(s) for prefix %s, using: %s", len(candidates), prefix, candidates[0].id)
	return candidates[0].engine, nil
}
