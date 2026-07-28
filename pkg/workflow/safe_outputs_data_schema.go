package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

var supportedDataSchemaTypes = map[string]struct{}{
	"object":  {},
	"array":   {},
	"string":  {},
	"number":  {},
	"integer": {},
	"boolean": {},
}

var dataSchemaAllowedKeys = map[string]struct{}{
	"type":                 {},
	"description":          {},
	"properties":           {},
	"required":             {},
	"items":                {},
	"enum":                 {},
	"additionalProperties": {},
	"minLength":            {},
	"maxLength":            {},
	"minimum":              {},
	"maximum":              {},
	"pattern":              {},
}

var dataSchemaBodyTypes = []string{
	"create_issue",
	"add_comment",
	"create_pull_request",
	"create_pull_request_review_comment",
	"submit_pull_request_review",
	"reply_to_pull_request_review_comment",
}

func isDataSchemaEnabledType(typeName string) bool {
	return slices.Contains(dataSchemaBodyTypes, typeName)
}

func validateSafeOutputsDataSchema(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}
	enabled, schema, schemaExpression, err := resolveSafeOutputsDataSchema(config)
	if err != nil {
		return err
	}
	config.DataEnabled = enabled
	config.NormalizedDataSchema = schema
	config.DataSchemaExpression = schemaExpression
	return nil
}

func resolveSafeOutputsDataSchema(config *SafeOutputsConfig) (bool, map[string]any, string, error) {
	if config == nil {
		return false, nil, "", nil
	}
	if config.Data == nil {
		return false, nil, "", nil
	}

	switch v := config.Data.(type) {
	case bool:
		if !v {
			return false, nil, "", nil
		}
		return true, nil, "", nil
	case map[string]any:
		normalized, err := simplifyDataSchemaNode(v, "safe-outputs.data", true)
		if err != nil {
			return false, nil, "", err
		}
		if normalizedType, _ := normalized["type"].(string); normalizedType != "object" {
			return false, nil, "", fmt.Errorf("safe-outputs.data must resolve to an object schema, got %q", normalizedType)
		}
		return true, normalized, "", nil
	case string:
		trimmed := strings.TrimSpace(v)
		if containsExpression(trimmed) {
			return true, nil, trimmed, nil
		}
		var parsed any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			schemaMap, ok := parsed.(map[string]any)
			if !ok {
				return false, nil, "", errors.New("safe-outputs.data string JSON must decode to an object schema")
			}
			normalized, normalizeErr := simplifyDataSchemaNode(schemaMap, "safe-outputs.data", true)
			if normalizeErr != nil {
				return false, nil, "", normalizeErr
			}
			if normalizedType, _ := normalized["type"].(string); normalizedType != "object" {
				return false, nil, "", fmt.Errorf("safe-outputs.data must resolve to an object schema, got %q", normalizedType)
			}
			return true, normalized, "", nil
		}
		return false, nil, "", errors.New("safe-outputs.data string values must be a GitHub Actions expression or JSON object schema")
	default:
		return false, nil, "", errors.New("safe-outputs.data must be false, true, an inline schema object, or a GitHub Actions expression")
	}
}

func simplifyDataSchemaNode(raw any, path string, allowShorthand bool) (map[string]any, error) {
	if typeName, ok := raw.(string); ok {
		if !allowShorthand {
			return nil, fmt.Errorf("%s: string shorthand is not allowed here", path)
		}
		if _, exists := supportedDataSchemaTypes[typeName]; !exists {
			return nil, fmt.Errorf("%s: unsupported type %q", path, typeName)
		}
		return map[string]any{"type": typeName}, nil
	}

	node, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected an object schema", path)
	}

	explicit := hasDataSchemaKeywords(node)
	if !explicit && allowShorthand {
		// Shorthand object syntax:
		// data:
		//   verdict: string
		//   score: number
		explicit = true
		node = map[string]any{
			"type":       "object",
			"properties": node,
		}
	}

	if !explicit {
		return nil, fmt.Errorf("%s: expected JSON schema keywords or shorthand properties", path)
	}

	for key := range node {
		if _, exists := dataSchemaAllowedKeys[key]; exists {
			continue
		}
		return nil, fmt.Errorf("%s: unsupported keyword %q", path, key)
	}

	result := make(map[string]any)
	typeName, _ := node["type"].(string)
	if typeName == "" {
		switch {
		case node["properties"] != nil || node["required"] != nil || node["additionalProperties"] != nil:
			typeName = "object"
		case node["items"] != nil:
			typeName = "array"
		}
	}
	if typeName != "" {
		if _, exists := supportedDataSchemaTypes[typeName]; !exists {
			return nil, fmt.Errorf("%s.type: unsupported type %q", path, typeName)
		}
		result["type"] = typeName
	}

	if desc, ok := node["description"]; ok {
		descStr, ok := desc.(string)
		if !ok {
			return nil, fmt.Errorf("%s.description: must be a string", path)
		}
		result["description"] = descStr
	}

	if enumVal, exists := node["enum"]; exists {
		enumList, ok := enumVal.([]any)
		if !ok || len(enumList) == 0 {
			return nil, fmt.Errorf("%s.enum: must be a non-empty array", path)
		}
		for i, enumItem := range enumList {
			switch enumItem.(type) {
			case string, float64, bool, int, int64:
			default:
				return nil, fmt.Errorf("%s.enum[%d]: must be a scalar value", path, i)
			}
		}
		result["enum"] = enumList
	}

	switch typeName {
	case "object":
		propertiesVal, hasProperties := node["properties"]
		if !hasProperties {
			return nil, fmt.Errorf("%s.properties: is required for object schemas", path)
		}
		propertiesMap, ok := propertiesVal.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s.properties: must be an object", path)
		}
		normalizedProperties := make(map[string]any, len(propertiesMap))
		for key, propertySchema := range propertiesMap {
			normalizedProperty, err := simplifyDataSchemaNode(propertySchema, fmt.Sprintf("%s.properties.%s", path, key), true)
			if err != nil {
				return nil, err
			}
			normalizedProperties[key] = normalizedProperty
		}
		result["properties"] = normalizedProperties
		requiredSet := make(map[string]struct{}, len(normalizedProperties))
		if requiredVal, exists := node["required"]; exists {
			requiredItems, ok := requiredVal.([]any)
			if !ok {
				return nil, fmt.Errorf("%s.required: must be an array of strings", path)
			}
			for i, requiredItem := range requiredItems {
				requiredName, ok := requiredItem.(string)
				if !ok || strings.TrimSpace(requiredName) == "" {
					return nil, fmt.Errorf("%s.required[%d]: must be a non-empty string", path, i)
				}
				if _, exists := normalizedProperties[requiredName]; !exists {
					return nil, fmt.Errorf("%s.required[%d]: unknown property %q", path, i, requiredName)
				}
				requiredSet[requiredName] = struct{}{}
			}
		}
		// OpenAI Codex structured outputs compatibility:
		// - require all object properties at every object level
		// - represent optionality explicitly in schema types instead of omitting from required
		// To keep output deterministic, store names in lexical order.
		requiredNames := make([]string, 0, len(normalizedProperties))
		for propertyName := range normalizedProperties {
			requiredSet[propertyName] = struct{}{}
		}
		for requiredName := range requiredSet {
			requiredNames = append(requiredNames, requiredName)
		}
		sort.Strings(requiredNames)
		result["required"] = requiredNames
		if additionalProps, exists := node["additionalProperties"]; exists {
			additionalPropsBool, ok := additionalProps.(bool)
			if !ok {
				return nil, fmt.Errorf("%s.additionalProperties: must be boolean", path)
			}
			if additionalPropsBool {
				return nil, fmt.Errorf("%s.additionalProperties: must be false for OpenAI Codex structured outputs compatibility", path)
			}
			result["additionalProperties"] = false
		} else {
			result["additionalProperties"] = false
		}
	case "array":
		items, exists := node["items"]
		if !exists {
			return nil, fmt.Errorf("%s.items: is required for array schemas", path)
		}
		normalizedItems, err := simplifyDataSchemaNode(items, path+".items", true)
		if err != nil {
			return nil, err
		}
		result["items"] = normalizedItems
	case "string":
		if minLen, exists := node["minLength"]; exists {
			result["minLength"] = minLen
		}
		if maxLen, exists := node["maxLength"]; exists {
			result["maxLength"] = maxLen
		}
		if pattern, exists := node["pattern"]; exists {
			result["pattern"] = pattern
		}
	case "number", "integer":
		if min, exists := node["minimum"]; exists {
			result["minimum"] = min
		}
		if max, exists := node["maximum"]; exists {
			result["maximum"] = max
		}
	}

	return result, nil
}

func hasDataSchemaKeywords(node map[string]any) bool {
	for key := range node {
		if _, exists := dataSchemaAllowedKeys[key]; exists {
			return true
		}
	}
	return false
}
