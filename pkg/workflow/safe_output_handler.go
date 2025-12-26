package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputHandlerLog = logger.New("workflow:safe_output_handler")

// SafeOutputHandler defines the interface for safe output message handlers.
// Each handler is responsible for building the steps needed to process
// a specific type of safe output message (e.g., create_issue, add_comment).
type SafeOutputHandler interface {
	// GetType returns the type identifier for this handler (e.g., "create_issue")
	GetType() string

	// IsEnabled checks if this handler should be processed based on the workflow configuration
	IsEnabled(data *WorkflowData) bool

	// BuildStepConfig builds the step configuration for this handler
	// Returns nil if the handler is not enabled or cannot be built
	BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig

	// GetOutputs returns the outputs that this handler produces
	// These are used to build the job outputs map
	GetOutputs() map[string]string

	// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
	RequiresTempIDMap() bool
}

// SafeOutputContext holds contextual information about previously processed handlers
// This allows handlers to reference outputs from earlier handlers in the sequence
type SafeOutputContext struct {
	// ThreatDetectionEnabled indicates if threat detection is enabled
	ThreatDetectionEnabled bool

	// MainJobName is the name of the main agent job
	MainJobName string

	// ProcessedHandlers tracks which handlers have been processed so far
	// Map key is the handler type (e.g., "create_issue")
	ProcessedHandlers map[string]bool

	// HandlerOutputs tracks the outputs from each processed handler
	// This allows later handlers to reference earlier outputs
	HandlerOutputs map[string]map[string]string

	// TempIDMapAvailable indicates if a temporary ID map is available from a previous handler
	TempIDMapAvailable bool

	// TempIDMapSource is the step ID that provides the temporary ID map
	TempIDMapSource string
}

// NewSafeOutputContext creates a new context for processing safe output handlers
func NewSafeOutputContext(mainJobName string, threatDetectionEnabled bool) *SafeOutputContext {
	return &SafeOutputContext{
		ThreatDetectionEnabled: threatDetectionEnabled,
		MainJobName:            mainJobName,
		ProcessedHandlers:      make(map[string]bool),
		HandlerOutputs:         make(map[string]map[string]string),
		TempIDMapAvailable:     false,
		TempIDMapSource:        "",
	}
}

// MarkProcessed marks a handler as processed and records its outputs
func (ctx *SafeOutputContext) MarkProcessed(handlerType string, outputs map[string]string) {
	ctx.ProcessedHandlers[handlerType] = true
	if len(outputs) > 0 {
		ctx.HandlerOutputs[handlerType] = outputs
	}
}

// IsProcessed checks if a handler has been processed
func (ctx *SafeOutputContext) IsProcessed(handlerType string) bool {
	return ctx.ProcessedHandlers[handlerType]
}

// GetHandlerOutput retrieves a specific output from a processed handler
func (ctx *SafeOutputContext) GetHandlerOutput(handlerType, outputKey string) string {
	if outputs, exists := ctx.HandlerOutputs[handlerType]; exists {
		return outputs[outputKey]
	}
	return ""
}

// SetTempIDMapSource marks that a temporary ID map is available from a specific step
func (ctx *SafeOutputContext) SetTempIDMapSource(stepID string) {
	ctx.TempIDMapAvailable = true
	ctx.TempIDMapSource = stepID
}

// SafeOutputHandlerRegistry manages the collection of safe output handlers
type SafeOutputHandlerRegistry struct {
	handlers []SafeOutputHandler
	logger   *logger.Logger
}

// NewSafeOutputHandlerRegistry creates a new handler registry
func NewSafeOutputHandlerRegistry() *SafeOutputHandlerRegistry {
	return &SafeOutputHandlerRegistry{
		handlers: make([]SafeOutputHandler, 0),
		logger:   safeOutputHandlerLog,
	}
}

// Register adds a handler to the registry
func (r *SafeOutputHandlerRegistry) Register(handler SafeOutputHandler) {
	r.handlers = append(r.handlers, handler)
	r.logger.Printf("Registered safe output handler: %s", handler.GetType())
}

// GetHandlers returns all registered handlers in order
func (r *SafeOutputHandlerRegistry) GetHandlers() []SafeOutputHandler {
	return r.handlers
}

// GetEnabledHandlers returns only the handlers that are enabled for the given workflow
func (r *SafeOutputHandlerRegistry) GetEnabledHandlers(data *WorkflowData) []SafeOutputHandler {
	enabled := make([]SafeOutputHandler, 0)
	for _, handler := range r.handlers {
		if handler.IsEnabled(data) {
			enabled = append(enabled, handler)
			r.logger.Printf("Handler enabled: %s", handler.GetType())
		}
	}
	return enabled
}
