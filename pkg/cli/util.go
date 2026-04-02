package cli

// boolPtr returns a pointer to the given bool value, used for optional *bool fields.
func boolPtr(b bool) *bool { return &b }
