package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputMessagesLog = logger.New("workflow:safe_outputs_config_messages")

const disclosureHeaderDefaultSentinel = "true"

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
	config.BodyHeader = extractStringFromMap(messagesMap, "body-header", nil)

	// Handle disclosure-header: can be bool (true for default built-in text) or custom string
	if dh, exists := messagesMap["disclosure-header"]; exists {
		switch v := dh.(type) {
		case bool:
			if v {
				config.DisclosureHeader = disclosureHeaderDefaultSentinel
			}
		case string:
			config.DisclosureHeader = v
		}
	}

	return config
}

// parseMentionsConfig parses the mentions configuration from safe-outputs frontmatter
// Mentions can be:
// - false: always escapes mentions
// - true: always allows mentions (error in strict mode)
// - object: detailed configuration with allowed-collaborators, allow-context, allowed, max
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
		// Parse allowed-collaborators (preferred) with fallback to deprecated allow-team-members
		config.AllowedCollaborators = parseMentionBoolPtr(mentionsMap, "allowed-collaborators", "allow-team-members")

		// Parse allow-context
		config.AllowContext = parseMentionBoolPtr(mentionsMap, "allow-context")

		// Parse allowed list
		config.Allowed = parseMentionStringList(mentionsMap, "allowed", "mention")

		// Parse allowed-teams list
		config.AllowedTeams = parseMentionStringList(mentionsMap, "allowed-teams", "team mention")

		// Parse max
		config.Max = parseMentionsMax(mentionsMap)
	}

	return config
}

func parseMentionBoolPtr(values map[string]any, keys ...string) *bool {
	for _, key := range keys {
		raw, exists := values[key]
		if !exists {
			continue
		}
		if val, ok := raw.(bool); ok {
			return &val
		}
		return nil
	}
	return nil
}

func parseMentionStringList(values map[string]any, key, logKind string) []string {
	raw, exists := values[key]
	if !exists {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	var stringsOut []string
	for _, item := range items {
		if str, ok := item.(string); ok {
			stringsOut = append(stringsOut, normalizeMentionString(str, logKind))
		}
	}
	return stringsOut
}

func normalizeMentionString(value, logKind string) string {
	normalized := value
	if value != "" && value[0] == '@' {
		normalized = value[1:]
		safeOutputMessagesLog.Printf("Normalized %s '%s' to '%s'", logKind, value, normalized)
	}
	return normalized
}

func parseMentionsMax(values map[string]any) *int {
	maxVal, exists := values["max"]
	if !exists {
		return nil
	}
	switch v := maxVal.(type) {
	case int:
		return validMentionsMax(v)
	case int64:
		return validMentionsMax(int(v))
	case uint64:
		return validMentionsMax(int(v))
	case float64:
		intVal := int(v)
		if v != float64(intVal) {
			safeOutputMessagesLog.Printf("mentions.max: float value %.2f truncated to integer %d", v, intVal)
		}
		return validMentionsMax(intVal)
	default:
		return nil
	}
}

func validMentionsMax(value int) *int {
	if value < 1 {
		return nil
	}
	return &value
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
