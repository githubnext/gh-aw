package workflow

// isValidReaction checks if a reaction value is valid according to the schema
func isValidReaction(reaction string) bool {
	validReactions := map[string]bool{
		"+1":       true,
		"-1":       true,
		"laugh":    true,
		"confused": true,
		"heart":    true,
		"hooray":   true,
		"rocket":   true,
		"eyes":     true,
	}
	return validReactions[reaction]
}
