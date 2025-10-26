package workflow

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
