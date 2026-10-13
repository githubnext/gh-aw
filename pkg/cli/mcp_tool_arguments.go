package cli

// Keep all CLI options before this boundary, including server-owned options.
func appendMCPPositionals(args []string, values ...string) []string {
	return append(append(args, "--"), values...)
}
