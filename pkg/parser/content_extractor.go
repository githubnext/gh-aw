package parser

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var contentExtractorLog = logger.New("parser:content_extractor")

// extractToolsFromContent extracts tools and mcp-servers sections from frontmatter as JSON string
func extractToolsFromContent(content string) (string, error) {
	log.Printf("Extracting tools from content: size=%d bytes", len(content))
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		log.Printf("Failed to extract frontmatter: %v", err)
		return "{}", nil // Return empty object on error to match bash behavior
	}

	// Create a map to hold the merged result
	extracted := make(map[string]any)

	// Helper function to merge a field into extracted map
	mergeField := func(fieldName string) {
		if fieldValue, exists := result.Frontmatter[fieldName]; exists {
			if fieldMap, ok := fieldValue.(map[string]any); ok {
				maps.Copy(extracted, fieldMap)
			}
		}
	}

	// Extract and merge tools section (tools are stored as tool_name: tool_config)
	mergeField("tools")

	// Extract and merge mcp-servers section (mcp-servers are stored as server_name: server_config)
	mergeField("mcp-servers")

	// If nothing was extracted, return empty object
	if len(extracted) == 0 {
		log.Print("No tools or mcp-servers found in content")
		return "{}", nil
	}

	log.Printf("Extracted %d tool/server configurations", len(extracted))
	// Convert to JSON string
	extractedJSON, err := json.Marshal(extracted)
	if err != nil {
		return "{}", nil
	}

	return strings.TrimSpace(string(extractedJSON)), nil
}

// extractToolsFromFrontmatter extracts tools and mcp-servers sections from an already-parsed
// frontmatter map as a JSON string. This avoids re-parsing YAML when the frontmatter has
// already been parsed, which is the common case for imported files (both builtin and local).
func extractToolsFromFrontmatter(frontmatter map[string]any) (string, error) {
	// Create a map to hold the merged result
	extracted := make(map[string]any)

	// Helper function to merge a field into extracted map.
	// Non-map values (e.g. arrays in custom-agent format) are skipped here;
	// schema validation elsewhere will report them as invalid if appropriate.
	mergeField := func(fieldName string) {
		if fieldValue, exists := frontmatter[fieldName]; exists {
			if fieldMap, ok := fieldValue.(map[string]any); ok {
				maps.Copy(extracted, fieldMap)
			}
		}
	}

	// Extract and merge tools section (tools are stored as tool_name: tool_config)
	mergeField("tools")

	// Extract and merge mcp-servers section (mcp-servers are stored as server_name: server_config)
	mergeField("mcp-servers")

	// If nothing was extracted, return empty object
	if len(extracted) == 0 {
		return "{}", nil
	}

	// Convert to JSON string. On marshal failure, return an empty object to
	// match the behaviour of extractToolsFromContent (defensive, not a hard error).
	extractedJSON, err := json.Marshal(extracted)
	if err != nil {
		return "{}", nil
	}

	return strings.TrimSpace(string(extractedJSON)), nil
}

// extractFrontmatterField extracts a specific field from frontmatter as JSON string
func extractFrontmatterField(content, fieldName, emptyValue string) (string, error) {
	contentExtractorLog.Printf("Extracting field: %s", fieldName)
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		contentExtractorLog.Printf("Failed to extract frontmatter for field %s: %v", fieldName, err)
		return emptyValue, nil // Return empty value on error
	}

	return extractFieldJSONFromMap(result.Frontmatter, fieldName, emptyValue)
}

// extractFieldJSONFromMap extracts a specific field from an already-parsed frontmatter map as a JSON string.
// This avoids re-parsing YAML when the frontmatter has already been parsed.
func extractFieldJSONFromMap(frontmatter map[string]any, fieldName, emptyValue string) (string, error) {
	contentExtractorLog.Printf("Extracting field from map: %s", fieldName)

	// Extract the requested field
	fieldValue, exists := frontmatter[fieldName]
	if !exists {
		contentExtractorLog.Printf("Field %s not found in frontmatter", fieldName)
		return emptyValue, nil
	}

	// Convert to JSON string
	fieldJSON, err := json.Marshal(fieldValue)
	if err != nil {
		contentExtractorLog.Printf("Failed to marshal field %s to JSON: %v", fieldName, err)
		return emptyValue, nil
	}

	contentExtractorLog.Printf("Successfully extracted field %s: size=%d bytes", fieldName, len(fieldJSON))
	return strings.TrimSpace(string(fieldJSON)), nil
}

// extractYAMLFieldFromMap extracts a specific field from an already-parsed frontmatter map as a YAML string.
// This avoids re-parsing YAML when the frontmatter has already been parsed.
func extractYAMLFieldFromMap(frontmatter map[string]any, fieldName string) (string, error) {
	contentExtractorLog.Printf("Extracting YAML field from map: %s", fieldName)

	fieldValue, exists := frontmatter[fieldName]
	if !exists {
		return "", nil
	}

	fieldYAML, err := yaml.Marshal(fieldValue)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(fieldYAML)), nil
}

// extractOnSectionFieldFromMap extracts a specific field from the on: section in an already-parsed
// frontmatter map as a JSON string. This avoids re-parsing YAML when the frontmatter has already been parsed.
func extractOnSectionFieldFromMap(frontmatter map[string]any, fieldName string) (string, error) {
	contentExtractorLog.Printf("Extracting on: section field from map: %s", fieldName)

	onValue, exists := frontmatter["on"]
	if !exists {
		return "[]", nil
	}

	onMap, ok := onValue.(map[string]any)
	if !ok {
		return "[]", nil
	}

	fieldValue, exists := onMap[fieldName]
	if !exists {
		return "[]", nil
	}

	var normalizedValue []any
	switch v := fieldValue.(type) {
	case string:
		if v != "" {
			normalizedValue = []any{v}
		}
	case []any:
		normalizedValue = v
	case []string:
		for _, s := range v {
			normalizedValue = append(normalizedValue, s)
		}
	default:
		return "[]", nil
	}

	jsonData, err := json.Marshal(normalizedValue)
	if err != nil {
		return "[]", nil
	}

	return string(jsonData), nil
}

// extractOnSectionAnyFieldFromMap extracts a specific field from the on: section in an already-parsed
// frontmatter map as a JSON string, handling any value type.
// This avoids re-parsing YAML when the frontmatter has already been parsed.
func extractOnSectionAnyFieldFromMap(frontmatter map[string]any, fieldName string) (string, error) {
	contentExtractorLog.Printf("Extracting on: section field (any) from map: %s", fieldName)

	onValue, exists := frontmatter["on"]
	if !exists {
		return "", nil
	}

	onMap, ok := onValue.(map[string]any)
	if !ok {
		return "", nil
	}

	fieldValue, exists := onMap[fieldName]
	if !exists {
		return "", nil
	}

	jsonData, err := json.Marshal(fieldValue)
	if err != nil {
		return "", nil
	}

	return string(jsonData), nil
}
