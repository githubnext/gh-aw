package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// mergeToolsFromJSON merges multiple JSON tool objects from content
func mergeToolsFromJSON(content string) (string, error) {
	log.Printf("Merging tools from JSON: content_size=%d bytes", len(content))
	// Clean up the content first
	content = strings.TrimSpace(content)

	// Try to parse as a single JSON object first
	var singleObj map[string]any
	if err := json.Unmarshal([]byte(content), &singleObj); err == nil {
		if len(singleObj) > 0 {
			result, err := json.Marshal(singleObj)
			if err != nil {
				return "{}", err
			}
			return string(result), nil
		}
	}

	// Find all JSON objects in the content (line by line)
	var jsonObjects []map[string]any

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "{}" {
			continue
		}

		var toolsObj map[string]any
		if err := json.Unmarshal([]byte(line), &toolsObj); err == nil {
			if len(toolsObj) > 0 { // Only add non-empty objects
				jsonObjects = append(jsonObjects, toolsObj)
			}
		}
	}

	// If no valid objects found, return empty
	if len(jsonObjects) == 0 {
		log.Print("No valid JSON objects found in content, returning empty object")
		return "{}", nil
	}

	log.Printf("Found %d JSON objects to merge", len(jsonObjects))
	// Merge all objects
	merged := make(map[string]any)
	for _, obj := range jsonObjects {
		var err error
		merged, err = MergeTools(merged, obj)
		if err != nil {
			return "{}", err
		}
	}

	// Convert back to JSON
	result, err := json.Marshal(merged)
	if err != nil {
		return "{}", err
	}

	return string(result), nil
}

// MergeTools merges two neutral tool configurations.
// Only supports merging arrays and maps for neutral tools (bash, web-fetch, web-search, edit, mcp-*).
// Removes all legacy Claude tool merging logic.
func MergeTools(base, additional map[string]any) (map[string]any, error) {
	log.Printf("Merging tools: base_keys=%d, additional_keys=%d", len(base), len(additional))
	result := make(map[string]any)

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Merge additional
	for key, newValue := range additional {
		if existingValue, exists := result[key]; exists {
			// Both have the same key, merge them

			// If both are arrays, merge and deduplicate
			_, existingIsArray := existingValue.([]any)
			_, newIsArray := newValue.([]any)
			if existingIsArray && newIsArray {
				merged := mergeAllowedArrays(existingValue, newValue)
				result[key] = merged
				continue
			}

			// If both are maps, check for special merging cases
			existingMap, existingIsMap := existingValue.(map[string]any)
			newMap, newIsMap := newValue.(map[string]any)
			if existingIsMap && newIsMap {
				// Check if this is an MCP tool (has MCP-compatible type)
				var existingType, newType string
				if existingMcp, hasMcp := existingMap["mcp"]; hasMcp {
					if mcpMap, ok := existingMcp.(map[string]any); ok {
						existingType, _ = mcpMap["type"].(string)
					}
				}
				if newMcp, hasMcp := newMap["mcp"]; hasMcp {
					if mcpMap, ok := newMcp.(map[string]any); ok {
						newType, _ = mcpMap["type"].(string)
					}
				}

				if isExistingMCP := IsMCPType(existingType); isExistingMCP {
					if isNewMCP := IsMCPType(newType); isNewMCP {
						// Both are MCP tools, check for conflicts
						mergedMap, err := mergeMCPTools(existingMap, newMap)
						if err != nil {
							return nil, fmt.Errorf("MCP tool conflict for '%s': %v", key, err)
						}
						result[key] = mergedMap
						continue
					}
				}

				// Both are maps, check for 'allowed' arrays to merge
				if existingAllowed, hasExistingAllowed := existingMap["allowed"]; hasExistingAllowed {
					if newAllowed, hasNewAllowed := newMap["allowed"]; hasNewAllowed {
						// Merge allowed arrays
						merged := mergeAllowedArrays(existingAllowed, newAllowed)
						mergedMap := make(map[string]any)
						for k, v := range existingMap {
							mergedMap[k] = v
						}
						for k, v := range newMap {
							mergedMap[k] = v
						}
						mergedMap["allowed"] = merged
						result[key] = mergedMap
						continue
					}
				}

				// No 'allowed' arrays to merge, recursively merge the maps
				recursiveMerged, err := MergeTools(existingMap, newMap)
				if err != nil {
					return nil, err
				}
				result[key] = recursiveMerged
			} else {
				// Not both same type, overwrite with new value
				result[key] = newValue
			}
		} else {
			// New key, just add it
			result[key] = newValue
		}
	}

	return result, nil
}

// mergeAllowedArrays merges two allowed arrays and removes duplicates
func mergeAllowedArrays(existing, new any) []any {
	var result []any
	seen := make(map[string]bool)

	// Add existing items
	if existingSlice, ok := existing.([]any); ok {
		for _, item := range existingSlice {
			if str, ok := item.(string); ok {
				if !seen[str] {
					result = append(result, str)
					seen[str] = true
				}
			}
		}
	}

	// Add new items
	if newSlice, ok := new.([]any); ok {
		for _, item := range newSlice {
			if str, ok := item.(string); ok {
				if !seen[str] {
					result = append(result, str)
					seen[str] = true
				}
			}
		}
	}

	return result
}

// mergeMCPTools merges two MCP tool configurations, detecting conflicts except for 'allowed' arrays
func mergeMCPTools(existing, new map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Copy existing properties
	for k, v := range existing {
		result[k] = v
	}

	// Merge new properties, checking for conflicts
	for key, newValue := range new {
		if existingValue, exists := result[key]; exists {
			if key == "allowed" {
				// Special handling for allowed arrays - merge them
				if existingArray, ok := existingValue.([]any); ok {
					if newArray, ok := newValue.([]any); ok {
						result[key] = mergeAllowedArrays(existingArray, newArray)
						continue
					}
				}
				// If not arrays, fall through to conflict check
			} else if key == "mcp" {
				// Special handling for mcp sub-objects - merge them recursively
				if existingMcp, ok := existingValue.(map[string]any); ok {
					if newMcp, ok := newValue.(map[string]any); ok {
						mergedMcp, err := mergeMCPTools(existingMcp, newMcp)
						if err != nil {
							return nil, fmt.Errorf("MCP config conflict: %v", err)
						}
						result[key] = mergedMcp
						continue
					}
				}
				// If not both maps, fall through to conflict check
			}

			// Check for conflicts (values must be equal)
			if !areEqual(existingValue, newValue) {
				return nil, fmt.Errorf("conflicting values for '%s': existing=%v, new=%v", key, existingValue, newValue)
			}
			// Values are equal, keep existing
		} else {
			// New property, add it
			result[key] = newValue
		}
	}

	return result, nil
}

// areEqual compares two values for equality, handling different types appropriately
func areEqual(a, b any) bool {
	// Convert to JSON for comparison to handle different types consistently
	aJSON, aErr := json.Marshal(a)
	bJSON, bErr := json.Marshal(b)

	if aErr != nil || bErr != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
