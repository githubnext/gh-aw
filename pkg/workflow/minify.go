// Package workflow provides JavaScript minification using terser.
//
// # Minification
//
// The minifier uses terser to compress and mangle JavaScript code for inline
// embedding in GitHub Actions YAML files. This reduces workflow file size
// and improves loading times.
//
// Minification is applied after bundling and before formatting for YAML.
package workflow

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var minifyLog = logger.New("workflow:minify")

// minificationEnabled controls whether JavaScript minification is enabled.
// It is enabled by default and can be disabled for debugging purposes.
var minificationEnabled = true

// minificationOnce ensures terser availability is checked only once
var minificationOnce sync.Once

// terserAvailable caches whether terser is available
var terserAvailable bool

// terserPath caches the path to the npx command
var terserPath string

// SetMinificationEnabled enables or disables JavaScript minification.
// This is primarily used for testing and debugging.
func SetMinificationEnabled(enabled bool) {
	minificationEnabled = enabled
}

// IsMinificationEnabled returns whether JavaScript minification is enabled.
func IsMinificationEnabled() bool {
	return minificationEnabled
}

// checkTerserAvailable checks if terser is available via npx
func checkTerserAvailable() {
	minificationOnce.Do(func() {
		path, err := exec.LookPath("npx")
		if err != nil {
			minifyLog.Printf("npx not found, minification disabled: %v", err)
			terserAvailable = false
			return
		}
		terserPath = path

		// Check if terser is available
		cmd := exec.Command(terserPath, "terser", "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			minifyLog.Printf("terser not available, minification disabled: %v", err)
			terserAvailable = false
			return
		}
		minifyLog.Printf("terser available: %s", strings.TrimSpace(string(output)))
		terserAvailable = true
	})
}

// MinifyJavaScript minifies JavaScript code using terser.
// If minification is disabled or terser is not available, returns the original code.
// The function uses terser with --compress and --mangle flags, then formats
// the output to break long lines for GitHub Actions YAML compatibility.
func MinifyJavaScript(code string) (string, error) {
	if !minificationEnabled {
		minifyLog.Print("Minification disabled, returning original code")
		return code, nil
	}

	checkTerserAvailable()
	if !terserAvailable {
		minifyLog.Print("Terser not available, returning original code")
		return code, nil
	}

	originalSize := len(code)
	minifyLog.Printf("Minifying JavaScript: %d bytes", originalSize)

	// Run terser with module, compress and mangle options
	// --module is required because GitHub Script code uses top-level await
	cmd := exec.Command(terserPath, "terser", "--module", "--compress", "--mangle")
	cmd.Stdin = strings.NewReader(code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Log the error but return original code - minification failures are not fatal
		minifyLog.Printf("Minification failed (returning original): %v, stderr: %s", err, stderr.String())
		return code, nil
	}

	minified := stdout.String()

	// Format the minified output to break long lines
	// GitHub Actions has a ~21KB limit for expression values, so we need to ensure
	// no single line exceeds a reasonable length
	formatted := formatMinifiedCode(minified)

	minifiedSize := len(formatted)

	// Calculate compression ratio
	ratio := float64(minifiedSize) / float64(originalSize) * 100
	savings := originalSize - minifiedSize

	minifyLog.Printf("Minification complete: %d -> %d bytes (%.1f%%, saved %d bytes)",
		originalSize, minifiedSize, ratio, savings)

	return formatted, nil
}

// MinifyJavaScriptOrFail minifies JavaScript code using terser.
// Unlike MinifyJavaScript, this function returns an error if minification fails.
// This is useful for testing or when minification is required.
func MinifyJavaScriptOrFail(code string) (string, error) {
	if !minificationEnabled {
		return "", fmt.Errorf("minification is disabled")
	}

	checkTerserAvailable()
	if !terserAvailable {
		return "", fmt.Errorf("terser is not available")
	}

	cmd := exec.Command(terserPath, "terser", "--module", "--compress", "--mangle")
	cmd.Stdin = strings.NewReader(code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("terser failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// maxLineLength is the maximum length of a line in minified output.
// GitHub Actions has a ~21KB limit for expression values, but we use a much
// smaller limit to ensure readability and avoid issues with YAML parsing.
// This is set to 500 characters to balance compression with readability.
const maxLineLength = 500

// formatMinifiedCode takes minified JavaScript and breaks long lines at safe points.
// This ensures the output stays within GitHub Actions expression size limits
// while maintaining valid JavaScript syntax.
func formatMinifiedCode(code string) string {
	// If the code is already short enough, return as-is
	if len(code) <= maxLineLength {
		return code
	}

	var result strings.Builder
	runes := []rune(code)
	lineStart := 0
	inString := false
	inTemplate := false
	stringChar := rune(0)
	braceDepth := 0 // Track brace depth for better break points

	for i := 0; i < len(runes); i++ {
		c := runes[i]

		// Track string literals to avoid breaking inside them
		if !inString && !inTemplate {
			if c == '"' || c == '\'' {
				inString = true
				stringChar = c
			} else if c == '`' {
				inTemplate = true
			} else if c == '{' {
				braceDepth++
			} else if c == '}' {
				braceDepth--
			}
		} else if inString {
			// Check for escaped character
			if c == '\\' && i+1 < len(runes) {
				i++ // Skip the next character
				continue
			}
			if c == stringChar {
				inString = false
			}
		} else if inTemplate {
			// Check for escaped character
			if c == '\\' && i+1 < len(runes) {
				i++ // Skip the next character
				continue
			}
			if c == '`' {
				inTemplate = false
			}
		}

		lineLen := i - lineStart

		// Check if we should insert a line break
		if lineLen >= maxLineLength && !inString && !inTemplate {
			// Find a safe break point - preferably after a semicolon, comma, or brace
			breakPoint := -1

			// Look back for a good break point
			for j := i; j > lineStart+maxLineLength/2; j-- {
				ch := runes[j]
				if ch == ';' || ch == ',' || ch == '{' || ch == '}' {
					breakPoint = j + 1 // Break after the character
					break
				}
			}

			// If no good break point found, break at current position
			// (only if not in a string or template)
			if breakPoint == -1 {
				breakPoint = i
			}

			// Write the content up to the break point
			result.WriteString(string(runes[lineStart:breakPoint]))
			result.WriteString("\n")
			lineStart = breakPoint
		}
	}

	// Write any remaining content
	if lineStart < len(runes) {
		result.WriteString(string(runes[lineStart:]))
	}

	return result.String()
}
