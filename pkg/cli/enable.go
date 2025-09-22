package cli

// EnableWorkflows enables workflows matching a pattern
func EnableWorkflows(pattern string) error {
	return toggleWorkflows(pattern, true)
}
