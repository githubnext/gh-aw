package workflow

import "fmt"

// validReactions defines the set of valid reaction values
var validReactions = map[string]bool{
	"+1":       true,
	"-1":       true,
	"laugh":    true,
	"confused": true,
	"heart":    true,
	"hooray":   true,
	"rocket":   true,
	"eyes":     true,
	"none":     true,
}

// isValidReaction checks if a reaction value is valid according to the schema
func isValidReaction(reaction string) bool {
	return validReactions[reaction]
}

// getValidReactions returns the list of valid reaction entries
func getValidReactions() []string {
	reactions := make([]string, 0, len(validReactions))
	for reaction := range validReactions {
		reactions = append(reactions, reaction)
	}
	return reactions
}

// parseReactionValue converts a reaction value from YAML to a string.
// YAML parsers may return +1 and -1 as integers, so this function handles
// both string and numeric types.
func parseReactionValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return intToReactionString(int64(v))
	case int64:
		return intToReactionString(v)
	case uint64:
		if v == 1 {
			return "+1", nil
		}
		return "", fmt.Errorf("invalid reaction value '%d': must be one of %v", v, getValidReactions())
	default:
		return "", fmt.Errorf("invalid reaction type: expected string, got %T", value)
	}
}

// intToReactionString converts an integer to a reaction string.
// Only 1 (+1) and -1 are valid integer values for reactions.
func intToReactionString(v int64) (string, error) {
	switch v {
	case 1:
		return "+1", nil
	case -1:
		return "-1", nil
	default:
		return "", fmt.Errorf("invalid reaction value '%d': must be one of %v", v, getValidReactions())
	}
}
