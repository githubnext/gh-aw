package workflow

import "strings"

// removeXMLComments removes XML comments (<!-- -->) from markdown content
// while preserving comments that appear within code blocks
func removeXMLComments(content string) string {
	// Track if we're inside a code block to avoid removing comments in code
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	var openMarker string
	inXMLComment := false

	for _, line := range lines {
		// Check for code block markers (3 or more ` or ~)
		trimmedLine := strings.TrimSpace(line)

		if !inCodeBlock && isValidCodeBlockMarker(trimmedLine) {
			// Opening a code block
			openMarker, _ = extractCodeBlockMarker(trimmedLine)
			inCodeBlock = true
			result = append(result, line)
			continue
		} else if inCodeBlock && isMatchingCodeBlockMarker(trimmedLine, openMarker) {
			// Closing the code block with matching marker
			inCodeBlock = false
			openMarker = ""
			result = append(result, line)
			continue
		}

		// If we're in a code block, preserve the line as-is
		if inCodeBlock {
			result = append(result, line)
			continue
		}

		// Process the line for XML comments
		processedLine, wasInComment, isInComment := removeXMLCommentsFromLine(line, inXMLComment)
		inXMLComment = isInComment

		if !wasInComment && !isInComment {
			// Line had no comment involvement, keep as-is
			result = append(result, processedLine)
		} else if !wasInComment && isInComment {
			// Line started a multiline comment, keep the processed part and add empty line
			if strings.TrimSpace(processedLine) != "" {
				result = append(result, processedLine)
			}
			result = append(result, "")
		} else if wasInComment && !isInComment {
			// Line ended a multiline comment, keep the processed part
			if strings.TrimSpace(processedLine) != "" {
				result = append(result, processedLine)
			}
		}
		// If wasInComment && isInComment, we're in the middle of a comment, skip the line
	}

	return strings.Join(result, "\n")
}

// removeXMLCommentsFromLine removes XML comments from a single line
// Returns: processed line, was initially in comment, is now in comment
func removeXMLCommentsFromLine(line string, inXMLComment bool) (string, bool, bool) {
	result := line
	wasInComment := inXMLComment

	for {
		if inXMLComment {
			// We're in a multiline comment, look for closing tag
			if closeIndex := strings.Index(result, "-->"); closeIndex != -1 {
				// Found closing tag, remove everything up to and including it
				result = result[closeIndex+3:]
				inXMLComment = false
				// Continue processing in case there are more comments on this line
			} else {
				// No closing tag found, entire line is part of the comment
				return "", wasInComment, inXMLComment
			}
		} else {
			// Not in a comment, look for opening tag
			if openIndex := strings.Index(result, "<!--"); openIndex != -1 {
				// Found opening tag
				if closeIndex := strings.Index(result[openIndex:], "-->"); closeIndex != -1 {
					// Complete comment on same line
					actualCloseIndex := openIndex + closeIndex + 3
					result = result[:openIndex] + result[actualCloseIndex:]
					// Continue processing in case there are more comments on this line
				} else {
					// Start of multiline comment
					result = result[:openIndex]
					inXMLComment = true
					break
				}
			} else {
				// No opening tag found, done processing this line
				break
			}
		}
	}

	return result, wasInComment, inXMLComment
}

// extractCodeBlockMarker extracts the marker string and language from a code block line
// Returns marker string (e.g., "```", "~~~~") and language specifier
func extractCodeBlockMarker(trimmedLine string) (string, string) {
	if len(trimmedLine) < 3 {
		return "", ""
	}

	var count int

	// Check for backticks
	if strings.HasPrefix(trimmedLine, "```") {
		for i, r := range trimmedLine {
			if r == '`' {
				count++
			} else {
				// Found language specifier or other content
				return strings.Repeat("`", count), strings.TrimSpace(trimmedLine[i:])
			}
		}
		// All characters are backticks
		return strings.Repeat("`", count), ""
	}

	// Check for tildes
	if strings.HasPrefix(trimmedLine, "~~~") {
		for i, r := range trimmedLine {
			if r == '~' {
				count++
			} else {
				// Found language specifier or other content
				return strings.Repeat("~", count), strings.TrimSpace(trimmedLine[i:])
			}
		}
		// All characters are tildes
		return strings.Repeat("~", count), ""
	}

	return "", ""
}

// isValidCodeBlockMarker checks if a trimmed line is a valid code block marker (3 or more ` or ~)
func isValidCodeBlockMarker(trimmedLine string) bool {
	marker, _ := extractCodeBlockMarker(trimmedLine)
	return len(marker) >= 3
}

// isMatchingCodeBlockMarker checks if the trimmed line matches the opening marker
func isMatchingCodeBlockMarker(trimmedLine string, openMarker string) bool {
	marker, _ := extractCodeBlockMarker(trimmedLine)
	if len(marker) == 0 || len(openMarker) == 0 {
		return false
	}

	// Markers must be the same type (both backticks or both tildes)
	if marker[0] != openMarker[0] {
		return false
	}

	// Closing marker must have at least as many characters as opening marker
	return len(marker) >= len(openMarker)
}
