package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var handlerProcessorLog = logger.New("workflow:safe_output_handler_processor")

// processHandlersSequentially processes safe output handlers in sequential order,
// maintaining a temporary ID map that is passed between handlers.
// Returns the generated steps, outputs, permissions, and step names.
func (c *Compiler) processHandlersSequentially(
	data *WorkflowData,
	mainJobName string,
	threatDetectionEnabled bool,
	registry *SafeOutputHandlerRegistry,
) ([]string, map[string]string, *Permissions, []string) {
	var steps []string
	outputs := make(map[string]string)
	permissions := NewPermissions()
	var stepNames []string

	// Create context for handler processing
	ctx := NewSafeOutputContext(mainJobName, threatDetectionEnabled)

	// Process each handler in order
	for _, handler := range registry.GetEnabledHandlers(data) {
		handlerType := handler.GetType()
		handlerProcessorLog.Printf("Processing handler: %s", handlerType)

		// Build step configuration for this handler
		stepConfig := handler.BuildStepConfig(c, data, ctx)
		if stepConfig == nil {
			handlerProcessorLog.Printf("Handler %s returned nil config, skipping", handlerType)
			continue
		}

		// Build the YAML steps for this handler
		stepYAML := c.buildConsolidatedSafeOutputStep(data, *stepConfig)
		steps = append(steps, stepYAML...)
		stepNames = append(stepNames, stepConfig.StepID)

		// Add handler outputs to job outputs
		handlerOutputs := handler.GetOutputs()
		for key, value := range handlerOutputs {
			outputs[key] = value
		}

		// Add handler-specific permissions
		switch handlerType {
		case "create_issue":
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
			// Mark that temporary ID map is now available
			ctx.SetTempIDMapSource("create_issue")
		case "create_discussion":
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		case "add_comment":
			permissions.Merge(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite())
		case "update_issue":
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		case "update_discussion":
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		case "close_issue":
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		case "close_discussion":
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		}

		// Mark handler as processed
		ctx.MarkProcessed(handlerType, stepConfig.Outputs)

		handlerProcessorLog.Printf("Handler %s processed successfully", handlerType)
	}

	handlerProcessorLog.Printf("Processed %d handlers, generated %d steps", len(stepNames), len(stepNames))

	return steps, outputs, permissions, stepNames
}

// getHandlerRegistry creates and initializes the safe output handler registry
// with all available handlers in the correct processing order.
func getHandlerRegistry() *SafeOutputHandlerRegistry {
	registry := NewSafeOutputHandlerRegistry()

	// Register handlers in processing order:
	// 1. create_issue - generates temporary ID map
	// 2. create_discussion - consumes temporary ID map
	// 3. update_issue - consumes temporary ID map
	// 4. update_discussion - consumes temporary ID map
	// 5. close_issue - consumes temporary ID map
	// 6. close_discussion - consumes temporary ID map
	// 7. add_comment - consumes temporary ID map and references other handler outputs

	registry.Register(NewCreateIssueHandler())
	registry.Register(NewCreateDiscussionHandler())
	registry.Register(NewUpdateIssueHandler())
	registry.Register(NewUpdateDiscussionHandler())
	registry.Register(NewCloseIssueHandler())
	registry.Register(NewCloseDiscussionHandler())
	registry.Register(NewAddCommentHandler())

	// Note: Other handlers (create_pull_request, etc.) will be migrated later
	// and added to the registry at that time

	return registry
}
