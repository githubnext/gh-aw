package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// writeScript writes a GitHub Actions script step with proper YAML indentation.
// It generates indented YAML, removes JavaScript comments to save size, and
// renders environment variables from the provided env map.
//
// Parameters:
//   - yaml: strings.Builder to write the YAML content to
//   - title: the step name/title
//   - jssource: the JavaScript source code
//   - env: map of environment variables to include in the env section
//   - githubToken: optional GitHub token for the with section (empty string to omit)
func writeScript(yaml *strings.Builder, title, jssource string, env map[string]string, githubToken string) {
	// Write the step name
	yaml.WriteString(fmt.Sprintf("      - name: %s\n", title))
	yaml.WriteString("        uses: actions/github-script@v8\n")

	// Write environment variables if any are provided
	if len(env) > 0 {
		yaml.WriteString("        env:\n")
		// Sort keys for consistent output
		var keys []string
		for key := range env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			yaml.WriteString(fmt.Sprintf("          %s: %s\n", key, env[key]))
		}
	}

	// Write the with section and script
	yaml.WriteString("        with:\n")
	
	// Add github-token if provided
	if githubToken != "" {
		yaml.WriteString(fmt.Sprintf("          github-token: %s\n", githubToken))
	}
	
	yaml.WriteString("          script: |\n")

	// Remove JavaScript comments and write with proper indentation
	cleanedScript := removeJavaScriptComments(jssource)
	WriteJavaScriptToYAML(yaml, cleanedScript)
}

// removeJavaScriptComments removes single-line (//) and multi-line (/* */) comments
// from JavaScript source code to reduce size while preserving functionality.
func removeJavaScriptComments(js string) string {
	lines := strings.Split(js, "\n")
	var result []string

	inMultiLineComment := false

	for _, line := range lines {
		processedLine := line

		// Handle continuation of multi-line comment from previous line
		if inMultiLineComment {
			// Look for end of multi-line comment
			if endIndex := strings.Index(processedLine, "*/"); endIndex != -1 {
				// Found end of multi-line comment, keep everything after it
				processedLine = processedLine[endIndex+2:]
				inMultiLineComment = false
			} else {
				// Still in multi-line comment, skip entire line
				continue
			}
		}

		// Process the line for comments (may have partial line if we just exited a multi-line comment)
		for {
			// Look for single-line comment
			singleLineIndex := strings.Index(processedLine, "//")
			// Look for start of multi-line comment
			multiLineIndex := strings.Index(processedLine, "/*")

			if singleLineIndex == -1 && multiLineIndex == -1 {
				// No comments found, keep the remaining line
				break
			}

			if singleLineIndex != -1 && (multiLineIndex == -1 || singleLineIndex < multiLineIndex) {
				// Single-line comment comes first, remove everything from // onwards
				processedLine = processedLine[:singleLineIndex]
				break
			}

			if multiLineIndex != -1 {
				// Multi-line comment starts
				endIndex := strings.Index(processedLine[multiLineIndex+2:], "*/")
				if endIndex != -1 {
					// Multi-line comment ends on same line
					before := processedLine[:multiLineIndex]
					after := processedLine[multiLineIndex+2+endIndex+2:]
					processedLine = before + after
					// Continue processing the rest of the line
				} else {
					// Multi-line comment continues to next line
					processedLine = processedLine[:multiLineIndex]
					inMultiLineComment = true
					break
				}
			}
		}

		// Only add non-empty lines to avoid extra whitespace
		// Also trim trailing whitespace from the line
		trimmedLine := strings.TrimSpace(processedLine)
		if trimmedLine != "" {
			result = append(result, trimmedLine)
		}
	}

	return strings.Join(result, "\n")
}

// removeJavaScriptCommentsRegex provides an alternative implementation using regex
// This is kept as a reference but the line-by-line approach above is preferred
// for better handling of edge cases and string literals containing comment-like patterns.
func removeJavaScriptCommentsRegex(js string) string {
	// Remove single-line comments (// to end of line)
	singleLineRegex := regexp.MustCompile(`//.*`)
	js = singleLineRegex.ReplaceAllString(js, "")

	// Remove multi-line comments (/* ... */)
	multiLineRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	js = multiLineRegex.ReplaceAllString(js, "")

	// Remove empty lines that may have been created and trim whitespace
	lines := strings.Split(js, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return strings.Join(result, "\n")
}