package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputMessagesLog = logger.New("workflow:safe_outputs_config_messages")

// ========================================
// Safe Output Messages Configuration
// ========================================

// parseMessagesConfig parses the messages configuration from safe-outputs frontmatter
func parseMessagesConfig(messagesMap map[string]any) *SafeOutputMessagesConfig {
	safeOutputMessagesLog.Printf("Parsing messages configuration with %d fields", len(messagesMap))
	config := &SafeOutputMessagesConfig{}

	if appendOnly, exists := messagesMap["append-only-comments"]; exists {
		if appendOnlyBool, ok := appendOnly.(bool); ok {
			config.AppendOnlyComments = appendOnlyBool
			safeOutputMessagesLog.Printf("Set append-only-comments: %t", appendOnlyBool)
		}
	}

	config.Footer = extractStringFromMap(messagesMap, "footer", nil)
	config.FooterInstall = extractStringFromMap(messagesMap, "footer-install", nil)
	config.FooterWorkflowRecompile = extractStringFromMap(messagesMap, "footer-workflow-recompile", nil)
	config.FooterWorkflowRecompileComment = extractStringFromMap(messagesMap, "footer-workflow-recompile-comment", nil)
	config.StagedTitle = extractStringFromMap(messagesMap, "staged-title", nil)
	config.StagedDescription = extractStringFromMap(messagesMap, "staged-description", nil)
	config.RunStarted = extractStringFromMap(messagesMap, "run-started", nil)
	config.RunSuccess = extractStringFromMap(messagesMap, "run-success", nil)
	config.RunFailure = extractStringFromMap(messagesMap, "run-failure", nil)
	config.DetectionFailure = extractStringFromMap(messagesMap, "detection-failure", nil)
	config.PullRequestCreated = extractStringFromMap(messagesMap, "pull-request-created", nil)
	config.IssueCreated = extractStringFromMap(messagesMap, "issue-created", nil)
	config.CommitPushed = extractStringFromMap(messagesMap, "commit-pushed", nil)
	config.AgentFailureIssue = extractStringFromMap(messagesMap, "agent-failure-issue", nil)
	config.AgentFailureComment = extractStringFromMap(messagesMap, "agent-failure-comment", nil)

	return config
}

// parseMentionsConfig parses the mentions configuration from safe-outputs frontmatter
// Mentions can be:
// - false: always escapes mentions
// - true: always allows mentions (error in strict mode)
// - object: detailed configuration with allow-team-members, allow-context, allowed, max
func parseMentionsConfig(mentions any) *MentionsConfig {
	safeOutputMessagesLog.Printf("Parsing mentions configuration: type=%T", mentions)
	config := &MentionsConfig{}

	// Handle boolean value
	if boolVal, ok := mentions.(bool); ok {
		config.Enabled = &boolVal
		safeOutputMessagesLog.Printf("Mentions configured as boolean: %t", boolVal)
		return config
	}

	// Handle object configuration
	if mentionsMap, ok := mentions.(map[string]any); ok {
		// Parse allow-team-members
		if allowTeamMembers, exists := mentionsMap["allow-team-members"]; exists {
			if val, ok := allowTeamMembers.(bool); ok {
				config.AllowTeamMembers = &val
			}
		}

		// Parse allow-context
		if allowContext, exists := mentionsMap["allow-context"]; exists {
			if val, ok := allowContext.(bool); ok {
				config.AllowContext = &val
			}
		}

		// Parse allowed list
		if allowed, exists := mentionsMap["allowed"]; exists {
			if allowedArray, ok := allowed.([]any); ok {
				var allowedStrings []string
				for _, item := range allowedArray {
					if str, ok := item.(string); ok {
						// Normalize username by removing '@' prefix if present
						normalized := str
						if len(str) > 0 && str[0] == '@' {
							normalized = str[1:]
							safeOutputMessagesLog.Printf("Normalized mention '%s' to '%s'", str, normalized)
						}
						allowedStrings = append(allowedStrings, normalized)
					}
				}
				config.Allowed = allowedStrings
			}
		}

		// Parse max
		if maxVal, exists := mentionsMap["max"]; exists {
			switch v := maxVal.(type) {
			case int:
				if v >= 1 {
					config.Max = &v
				}
			case int64:
				intVal := int(v)
				if intVal >= 1 {
					config.Max = &intVal
				}
			case uint64:
				intVal := int(v)
				if intVal >= 1 {
					config.Max = &intVal
				}
			case float64:
				intVal := int(v)
				// Warn if truncation occurs
				if v != float64(intVal) {
					safeOutputMessagesLog.Printf("mentions.max: float value %.2f truncated to integer %d", v, intVal)
				}
				if intVal >= 1 {
					config.Max = &intVal
				}
			}
		}
	}

	return config
}

// serializeMessagesConfig converts SafeOutputMessagesConfig to JSON for passing as environment variable
func serializeMessagesConfig(messages *SafeOutputMessagesConfig) (string, error) {
	if messages == nil {
		return "", nil
	}
	safeOutputMessagesLog.Print("Serializing messages configuration to JSON")
	jsonBytes, err := json.Marshal(messages)
	if err != nil {
		safeOutputMessagesLog.Printf("Failed to serialize messages config: %v", err)
		return "", fmt.Errorf("failed to serialize messages config: %w", err)
	}
	safeOutputMessagesLog.Printf("Serialized messages config: %d bytes", len(jsonBytes))
	return string(jsonBytes), nil
}
