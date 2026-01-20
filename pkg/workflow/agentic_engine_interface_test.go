package workflow

// Compile-time assertions to verify that all engine implementations satisfy the focused interfaces
// These assertions ensure that refactoring the interface doesn't break existing implementations

// Verify ClaudeEngine implements all interfaces
var _ EngineMetadata = (*ClaudeEngine)(nil)
var _ EngineCapabilities = (*ClaudeEngine)(nil)
var _ EngineInstaller = (*ClaudeEngine)(nil)
var _ EngineExecutor = (*ClaudeEngine)(nil)
var _ EngineLogParser = (*ClaudeEngine)(nil)
var _ CodingAgentEngine = (*ClaudeEngine)(nil)

// Verify CopilotEngine implements all interfaces
var _ EngineMetadata = (*CopilotEngine)(nil)
var _ EngineCapabilities = (*CopilotEngine)(nil)
var _ EngineInstaller = (*CopilotEngine)(nil)
var _ EngineExecutor = (*CopilotEngine)(nil)
var _ EngineLogParser = (*CopilotEngine)(nil)
var _ CodingAgentEngine = (*CopilotEngine)(nil)

// Verify CodexEngine implements all interfaces
var _ EngineMetadata = (*CodexEngine)(nil)
var _ EngineCapabilities = (*CodexEngine)(nil)
var _ EngineInstaller = (*CodexEngine)(nil)
var _ EngineExecutor = (*CodexEngine)(nil)
var _ EngineLogParser = (*CodexEngine)(nil)
var _ CodingAgentEngine = (*CodexEngine)(nil)

// Verify CustomEngine implements all interfaces
var _ EngineMetadata = (*CustomEngine)(nil)
var _ EngineCapabilities = (*CustomEngine)(nil)
var _ EngineInstaller = (*CustomEngine)(nil)
var _ EngineExecutor = (*CustomEngine)(nil)
var _ EngineLogParser = (*CustomEngine)(nil)
var _ CodingAgentEngine = (*CustomEngine)(nil)
