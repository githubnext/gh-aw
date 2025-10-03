package workflow

// CommentEventMapping defines the mapping between event identifiers and their GitHub Actions event configurations
type CommentEventMapping struct {
	EventName string   // GitHub Actions event name (e.g., "issues", "issue_comment")
	Types     []string // Event types (e.g., ["opened", "edited", "reopened"])
}

// GetAllCommentEvents returns all possible comment-related events for command triggers
func GetAllCommentEvents() []CommentEventMapping {
	return []CommentEventMapping{
		{
			EventName: "issues",
			Types:     []string{"opened", "edited", "reopened"},
		},
		{
			EventName: "issue_comment",
			Types:     []string{"created", "edited"},
		},
		{
			EventName: "pull_request",
			Types:     []string{"opened", "edited", "reopened"},
		},
		{
			EventName: "pull_request_review_comment",
			Types:     []string{"created", "edited"},
		},
	}
}

// GetCommentEventByIdentifier returns the event mapping for a given identifier
// Supports both short identifiers (e.g., "issue") and full event names (e.g., "issues")
func GetCommentEventByIdentifier(identifier string) *CommentEventMapping {
	// Map short identifiers to full event names
	identifierMap := map[string]string{
		"issue":                       "issues",
		"issues":                      "issues",
		"comment":                     "issue_comment",
		"issue_comment":               "issue_comment",
		"pr":                          "pull_request",
		"pull_request":                "pull_request",
		"pr_review":                   "pull_request_review_comment",
		"pr_review_comment":           "pull_request_review_comment",
		"pull_request_review_comment": "pull_request_review_comment",
	}

	eventName, ok := identifierMap[identifier]
	if !ok {
		return nil
	}

	// Find and return the matching event mapping
	allEvents := GetAllCommentEvents()
	for i := range allEvents {
		if allEvents[i].EventName == eventName {
			return &allEvents[i]
		}
	}

	return nil
}

// ParseCommandEvents parses the events field from command configuration
// Returns a list of event identifiers to enable, or nil for default (all events)
func ParseCommandEvents(eventsValue any) []string {
	if eventsValue == nil {
		return nil // Default: all events
	}

	// Handle string value (e.g., "*" or single event)
	if str, ok := eventsValue.(string); ok {
		if str == "*" {
			return nil // Explicit all events
		}
		return []string{str}
	}

	// Handle array of strings
	if arr, ok := eventsValue.([]any); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return nil // Default if parsing fails
}

// FilterCommentEvents returns only the comment events specified by the identifiers
// If identifiers is nil or empty, returns all comment events
func FilterCommentEvents(identifiers []string) []CommentEventMapping {
	if len(identifiers) == 0 {
		return GetAllCommentEvents()
	}

	var result []CommentEventMapping
	for _, identifier := range identifiers {
		if mapping := GetCommentEventByIdentifier(identifier); mapping != nil {
			result = append(result, *mapping)
		}
	}

	return result
}

// GetCommentEventNames returns just the event names from a list of mappings
func GetCommentEventNames(mappings []CommentEventMapping) []string {
	names := make([]string, len(mappings))
	for i, mapping := range mappings {
		names[i] = mapping.EventName
	}
	return names
}
