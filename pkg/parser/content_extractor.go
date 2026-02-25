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

// extractStepsFromContent extracts steps section from frontmatter as YAML string
func extractStepsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract steps section
	steps, exists := result.Frontmatter["steps"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how CustomSteps are handled in compiler)
	stepsYAML, err := yaml.Marshal(steps)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(stepsYAML)), nil
}

// extractServicesFromContent extracts services section from frontmatter as YAML string
func extractServicesFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract services section
	services, exists := result.Frontmatter["services"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how steps are handled)
	servicesYAML, err := yaml.Marshal(services)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(servicesYAML)), nil
}

// ExtractPermissionsFromContent extracts permissions section from frontmatter as JSON string
func ExtractPermissionsFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "permissions", "{}")
}

// extractPostStepsFromContent extracts post-steps section from frontmatter as YAML string
func extractPostStepsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract post-steps section
	postSteps, exists := result.Frontmatter["post-steps"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how steps are handled)
	postStepsYAML, err := yaml.Marshal(postSteps)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(postStepsYAML)), nil
}

// extractFrontmatterField extracts a specific field from frontmatter as JSON string
func extractFrontmatterField(content, fieldName, emptyValue string) (string, error) {
	contentExtractorLog.Printf("Extracting field: %s", fieldName)
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		contentExtractorLog.Printf("Failed to extract frontmatter for field %s: %v", fieldName, err)
		return emptyValue, nil // Return empty value on error
	}

	// Extract the requested field
	fieldValue, exists := result.Frontmatter[fieldName]
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

// extractOnSectionField extracts a specific field from the on: section in frontmatter as JSON string
func extractOnSectionField(content, fieldName string) (string, error) {
	contentExtractorLog.Printf("Extracting on: section field: %s", fieldName)
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		contentExtractorLog.Printf("Failed to extract frontmatter for field %s: %v", fieldName, err)
		return "[]", nil // Return empty array on error
	}

	// Extract the "on" section
	onValue, exists := result.Frontmatter["on"]
	if !exists {
		contentExtractorLog.Printf("Field 'on' not found in frontmatter")
		return "[]", nil
	}

	// The on: section should be a map
	onMap, ok := onValue.(map[string]any)
	if !ok {
		contentExtractorLog.Printf("Field 'on' is not a map: %T", onValue)
		return "[]", nil
	}

	// Extract the requested field from the on: section
	fieldValue, exists := onMap[fieldName]
	if !exists {
		contentExtractorLog.Printf("Field %s not found in 'on' section", fieldName)
		return "[]", nil
	}

	// Normalize field value to an array
	var normalizedValue []any
	switch v := fieldValue.(type) {
	case string:
		// Single string value
		if v != "" {
			normalizedValue = []any{v}
		}
	case []any:
		// Already an array
		normalizedValue = v
	case []string:
		// String array - convert to []any
		for _, s := range v {
			normalizedValue = append(normalizedValue, s)
		}
	default:
		contentExtractorLog.Printf("Unexpected type for field %s: %T", fieldName, fieldValue)
		return "[]", nil
	}

	// Return JSON string
	jsonData, err := json.Marshal(normalizedValue)
	if err != nil {
		contentExtractorLog.Printf("Failed to marshal field %s to JSON: %v", fieldName, err)
		return "[]", nil
	}

	contentExtractorLog.Printf("Successfully extracted field %s from on: section: %d bytes", fieldName, len(jsonData))
	return string(jsonData), nil
}
