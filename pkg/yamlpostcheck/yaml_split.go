package yamlpostcheck

import "strings"

// SplitYAMLHeader separates the leading comment block (the gh-aw-metadata /
// gh-aw-manifest / logo / description header that the compiler prepends to
// every lock file) from the YAML body that follows it.
//
// The header consists of all consecutive lines at the beginning of content
// that start with "#" or are blank.  The body is everything that remains.
//
// If content contains no non-comment lines, header is the full content and
// body is empty.
func SplitYAMLHeader(content string) (header, body string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// A non-empty, non-comment line marks the start of the YAML body.
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			header = strings.Join(lines[:i], "\n")
			body = strings.Join(lines[i:], "\n")
			return header, body
		}
	}
	// All lines are comments / blank — the whole thing is a header.
	return content, ""
}

// JoinYAMLHeaderBody joins a previously split header and body back into a
// single YAML document.  A newline is inserted between them only when both
// parts are non-empty and the header does not already end with a newline.
func JoinYAMLHeaderBody(header, body string) string {
	if header == "" {
		return body
	}
	if body == "" {
		return header
	}
	// Ensure exactly one newline between header and body.
	if strings.HasSuffix(header, "\n") {
		return header + body
	}
	return header + "\n" + body
}
