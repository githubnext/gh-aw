package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// JSONPathLocation represents a location in YAML source corresponding to a JSON path
type JSONPathLocation struct {
	Line   int
	Column int
	Found  bool
}

// ExtractJSONPathFromValidationError extracts JSON path information from jsonschema validation errors
func ExtractJSONPathFromValidationError(err error) []JSONPathInfo {
	var paths []JSONPathInfo

	if validationError, ok := err.(*jsonschema.ValidationError); ok {
		// Process each cause (individual validation error)
		for _, cause := range validationError.Causes {
			path := JSONPathInfo{
				Path:     convertInstanceLocationToJSONPath(cause.InstanceLocation),
				Message:  cause.Error(),
				Location: cause.InstanceLocation,
			}
			paths = append(paths, path)
		}
	}

	return paths
}

// JSONPathInfo holds information about a validation error and its path
type JSONPathInfo struct {
	Path     string   // JSON path like "/tools/1" or "/age"
	Message  string   // Error message
	Location []string // Instance location from jsonschema (e.g., ["tools", "1"])
}

// convertInstanceLocationToJSONPath converts jsonschema InstanceLocation to JSON path string
func convertInstanceLocationToJSONPath(location []string) string {
	if len(location) == 0 {
		return ""
	}

	var parts []string
	for _, part := range location {
		parts = append(parts, "/"+part)
	}
	return strings.Join(parts, "")
}

// LocateJSONPathInYAML finds the line/column position of a JSON path in YAML source
func LocateJSONPathInYAML(yamlContent string, jsonPath string) JSONPathLocation {
	if jsonPath == "" {
		// Root level error - return start of content
		return JSONPathLocation{Line: 1, Column: 1, Found: true}
	}

	// Parse the path segments
	pathSegments := parseJSONPath(jsonPath)
	if len(pathSegments) == 0 {
		return JSONPathLocation{Line: 1, Column: 1, Found: true}
	}

	// For now, use a simple line-by-line approach to find the path
	// This is less precise than using the YAML parser's position info,
	// but will work as a starting point
	location := findPathInYAMLLines(yamlContent, pathSegments)
	return location
}

// findPathInYAMLLines finds a JSON path in YAML content using line-by-line analysis
func findPathInYAMLLines(yamlContent string, pathSegments []PathSegment) JSONPathLocation {
	lines := strings.Split(yamlContent, "\n")

	// Start from the beginning
	currentLevel := 0
	arrayContexts := make(map[int]int) // level -> current array index

	for lineNum, line := range lines {
		lineNumber := lineNum + 1 // 1-based line numbers
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Calculate indentation level
		lineLevel := (len(line) - len(strings.TrimLeft(line, " \t"))) / 2

		// Check if this line matches our path
		matches, column := matchesPathAtLevel(line, pathSegments, lineLevel, arrayContexts)
		if matches {
			return JSONPathLocation{Line: lineNumber, Column: column, Found: true}
		}

		// Update array contexts for list items
		if strings.HasPrefix(trimmedLine, "-") {
			arrayContexts[lineLevel]++
		} else if lineLevel <= currentLevel {
			// Reset array contexts for deeper levels when we move to a shallower level
			for level := lineLevel + 1; level <= currentLevel; level++ {
				delete(arrayContexts, level)
			}
		}

		currentLevel = lineLevel
	}

	return JSONPathLocation{Line: 1, Column: 1, Found: false}
}

// matchesPathAtLevel checks if a line matches the target path at the current level
func matchesPathAtLevel(line string, pathSegments []PathSegment, level int, arrayContexts map[int]int) (bool, int) {
	if len(pathSegments) == 0 {
		return false, 0
	}

	trimmedLine := strings.TrimSpace(line)

	// For now, implement a simple key matching approach
	// This is a simplified version - in a full implementation we'd need to track
	// the complete path context as we traverse the YAML

	if level < len(pathSegments) {
		segment := pathSegments[level]

		if segment.Type == "key" {
			// Look for "key:" pattern
			keyPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(segment.Value) + `\s*:`)
			if keyPattern.MatchString(trimmedLine) {
				// Found the key - return position after the colon
				colonIndex := strings.Index(line, ":")
				if colonIndex != -1 {
					return level == len(pathSegments)-1, colonIndex + 2
				}
			}
		} else if segment.Type == "index" {
			// For array elements, check if this is a list item at the right index
			if strings.HasPrefix(trimmedLine, "-") {
				currentIndex := arrayContexts[level]
				if currentIndex == segment.Index {
					return level == len(pathSegments)-1, strings.Index(line, "-") + 2
				}
			}
		}
	}

	return false, 0
}

// parseJSONPath parses a JSON path string into segments
func parseJSONPath(path string) []PathSegment {
	if path == "" || path == "/" {
		return []PathSegment{}
	}

	// Remove leading slash and split by slash
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	var segments []PathSegment
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check if this is an array index
		if index, err := strconv.Atoi(part); err == nil {
			segments = append(segments, PathSegment{Type: "index", Value: part, Index: index})
		} else {
			segments = append(segments, PathSegment{Type: "key", Value: part})
		}
	}

	return segments
}

// PathSegment represents a segment in a JSON path
type PathSegment struct {
	Type  string // "key" or "index"
	Value string // The raw value
	Index int    // Parsed index for array elements
}
