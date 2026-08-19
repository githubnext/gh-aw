package parser

import "strings"

func splitPathAndSection(path string) (string, string) {
	if strings.Contains(path, "#") {
		parts := strings.SplitN(path, "#", 2)
		return parts[0], parts[1]
	}
	return path, ""
}
