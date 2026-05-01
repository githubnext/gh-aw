package workflow

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
)

// validateStructuredOutput checks that structured-output is only used with the codex engine
// and that the configuration is complete (either schema or schema-file is specified).
func validateStructuredOutput(workflowData *WorkflowData) error {
	if workflowData.StructuredOutput == nil {
		return nil
	}

	// structured-output is only supported with the codex engine
	if workflowData.AI != string(constants.CodexEngine) {
		return fmt.Errorf("structured-output is only supported with the codex engine (current engine: %q). "+
			"Other engines do not support native structured output schema enforcement", workflowData.AI)
	}

	config := workflowData.StructuredOutput

	// Require exactly one of schema or schema-file
	hasSchema := len(config.Schema) > 0
	hasSchemaFile := config.SchemaFile != ""

	if !hasSchema && !hasSchemaFile {
		return errors.New("structured-output requires either 'schema' (inline JSON Schema) or 'schema-file' (path to a JSON Schema file)")
	}

	if hasSchema && hasSchemaFile {
		return errors.New("structured-output cannot specify both 'schema' and 'schema-file'; use one or the other")
	}

	return nil
}
