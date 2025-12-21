package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var labelTriggerParserLog = logger.New("workflow:label_trigger_parser")

// parseLabelTriggerShorthand parses a string in the format
// "issue labeled label1 label2 ..." or "pull_request labeled label1 label2 ..."
// and returns the entity type and label names.
// Returns an empty string for entityType if not a valid label trigger shorthand.
// Returns an error if the format is invalid.
// Note: Discussion labeled triggers are not supported as GitHub Actions does not support label filtering for discussions.
func parseLabelTriggerShorthand(input string) (entityType string, labelNames []string, isLabelTrigger bool, err error) {
	input = strings.TrimSpace(input)

	// Split into tokens
	tokens := strings.Fields(input)
	if len(tokens) < 3 {
		// Need at least: "issue/pull_request labeled label1"
		return "", nil, false, nil
	}

	// Check for different patterns:
	// 1. "issue labeled label1 label2 ..."
	// 2. "pull_request labeled label1 label2 ..."

	var startIdx int

	if tokens[0] == "issue" && tokens[1] == "labeled" {
		// Pattern 1: "issue labeled label1 label2 ..."
		entityType = "issues"
		startIdx = 2
	} else if tokens[0] == "pull_request" && tokens[1] == "labeled" {
		// Pattern 2: "pull_request labeled label1 label2 ..."
		entityType = "pull_request"
		startIdx = 2
	} else {
		// Not a label trigger shorthand
		return "", nil, false, nil
	}

	// Extract label names
	if len(tokens) <= startIdx {
		return "", nil, true, fmt.Errorf("label trigger shorthand requires at least one label name")
	}

	labelNames = tokens[startIdx:]

	// Validate label names are not empty
	for _, label := range labelNames {
		if strings.TrimSpace(label) == "" {
			return "", nil, true, fmt.Errorf("label names cannot be empty in label trigger shorthand")
		}
	}

	labelTriggerParserLog.Printf("Parsed label trigger shorthand: %s -> entity: %s, labels: %v", input, entityType, labelNames)

	return entityType, labelNames, true, nil
}

// expandLabelTriggerShorthand takes an entity type and label names and returns a map that represents
// the expanded label trigger + workflow_dispatch configuration with item_number input.
// Note: Only "issues" and "pull_request" entity types are supported, as GitHub Actions
// does not support label filtering for discussions.
func expandLabelTriggerShorthand(entityType string, labelNames []string) map[string]any {
	// Create the trigger configuration based on entity type
	var triggerKey string
	switch entityType {
	case "issues":
		triggerKey = "issues"
	case "pull_request":
		triggerKey = "pull_request"
	default:
		triggerKey = "issues" // Default to issues (though this shouldn't happen with our parser)
	}

	// Build the trigger configuration
	// Add a marker to indicate this uses native GitHub Actions label filtering
	// (not job condition filtering), so names should not be commented out
	triggerConfig := map[string]any{
		"types":                         []any{"labeled"},
		"names":                         labelNames,
		"__gh_aw_native_label_filter__": true, // Marker to prevent commenting out names
	}

	// Create workflow_dispatch with item_number input
	workflowDispatchConfig := map[string]any{
		"inputs": map[string]any{
			"item_number": map[string]any{
				"description": "The number of the " + getItemTypeName(entityType),
				"required":    true,
				"type":        "string",
			},
		},
	}

	return map[string]any{
		triggerKey:          triggerConfig,
		"workflow_dispatch": workflowDispatchConfig,
	}
}

// getItemTypeName returns the human-readable item type name for the entity type
func getItemTypeName(entityType string) string {
	switch entityType {
	case "issues":
		return "issue"
	case "pull_request":
		return "pull request"
	default:
		return "item" // Fallback (though this shouldn't happen with our parser)
	}
}
