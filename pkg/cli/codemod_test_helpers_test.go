//go:build !integration

package cli

// splitLines splits a string into lines without using strings.Split,
// preserving the behavior expected by codemod tests.
func splitLines(content string) []string {
	lines := []string{}
	current := ""
	for _, char := range content {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
