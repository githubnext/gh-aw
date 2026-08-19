package parser

import "strings"

func splitPathAndSection(path string) (string, string) {
	if before, after, ok := strings.Cut(path, "#"); ok {
		return before, after
	}
	return path, ""
}
