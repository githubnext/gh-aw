package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var otlpMetadataReferencePattern = regexp.MustCompile(`^\s*\$\{\{\s*github\.aw\.metadata\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}\s*$`)
var otlpMetadataAccessPattern = regexp.MustCompile(`^github\s*\.\s*aw\s*(?:\.\s*metadata\b|\[\s*'metadata'\s*\]|\[\s*"metadata"\s*\])`)

// resolveOTLPMetadataReferences replaces whole-value references in custom OTLP
// attribute maps with strings from the root workflow's metadata.
func resolveOTLPMetadataReferences(workflowData *WorkflowData) error {
	if workflowData == nil {
		return nil
	}
	obs := extractRawObservabilityMap(workflowData.RawFrontmatter)
	if obs == nil {
		return nil
	}
	otlp, ok := obs["otlp"].(map[string]any)
	if !ok {
		return nil
	}

	var metadata map[string]any
	if workflowData.RawFrontmatter != nil {
		if metadataValue, exists := workflowData.RawFrontmatter["metadata"]; exists {
			if metadataMap, isMap := metadataValue.(map[string]any); isMap {
				metadata = metadataMap
			}
		}
	}

	for _, field := range []string{"attributes", "resource-attributes"} {
		attributes := normalizeOTLPAttributeMap(otlp, field)
		if attributes == nil {
			continue
		}
		names := make([]string, 0, len(attributes))
		for name := range attributes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			value := attributes[name]
			text, ok := value.(string)
			if !ok || !containsOTLPMetadataAccess(text) {
				continue
			}
			scope := "observability.otlp." + field + "." + name
			resolved, err := resolveOTLPMetadataValue(text, metadata, scope)
			if err != nil {
				return err
			}
			attributes[name] = resolved
		}
	}
	return nil
}

func resolveOTLPMetadataValue(text string, metadata map[string]any, scope string) (string, error) {
	match := otlpMetadataReferencePattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", fmt.Errorf("%s must be a whole-value reference in the form ${{ github.aw.metadata.<key> }}", scope)
	}
	key := ""
	for index, capture := range match {
		if index == 1 {
			key = capture
			break
		}
	}
	if metadata == nil {
		return "", fmt.Errorf("%s references metadata key %q, but the root workflow has no metadata map", scope, key)
	}
	metadataValue, exists := metadata[key]
	if !exists {
		return "", fmt.Errorf("%s references missing metadata key %q", scope, key)
	}
	resolved, ok := metadataValue.(string)
	if !ok {
		return "", fmt.Errorf("%s references metadata key %q, which must be a string", scope, key)
	}
	if strings.Contains(resolved, "${{") || strings.Contains(resolved, "}}") {
		return "", fmt.Errorf("%s cannot resolve metadata key %q because its value contains GitHub Actions expression delimiters", scope, key)
	}
	return resolved, nil
}

func containsOTLPMetadataAccess(value string) bool {
	searchFrom := 0
	for {
		open := strings.Index(value[searchFrom:], "${{")
		if open == -1 {
			return false
		}
		expressionStart := searchFrom + open + len("${{")
		closed := false
		for index := expressionStart; index < len(value); {
			current, ok := otlpStringByteAt(value, index)
			if !ok {
				break
			}
			switch current {
			case '\'', '"':
				index = skipOTLPExpressionString(value, index)
				continue
			case '}':
				next, ok := otlpStringByteAt(value, index+1)
				if ok && next == '}' {
					searchFrom = index + 2
					closed = true
				}
			}
			if closed {
				break
			}
			previous, hasPrevious := otlpStringByteAt(value, index-1)
			if current == 'g' && (!hasPrevious || index == expressionStart || !isOTLPIdentifierCharacter(previous)) && otlpMetadataAccessPattern.MatchString(value[index:]) {
				return true
			}
			index++
		}
		if !closed {
			return false
		}
	}
}

func skipOTLPExpressionString(value string, start int) int {
	quote, ok := otlpStringByteAt(value, start)
	if !ok {
		return len(value)
	}
	for index := start + 1; index < len(value); index++ {
		current, ok := otlpStringByteAt(value, index)
		if !ok {
			return len(value)
		}
		if current == '\\' {
			index++
			continue
		}
		if current == quote {
			next, ok := otlpStringByteAt(value, index+1)
			if ok && next == quote {
				index++
				continue
			}
			return index + 1
		}
	}
	return len(value)
}

func otlpStringByteAt(value string, index int) (byte, bool) {
	if index < 0 || index >= len(value) {
		return 0, false
	}
	return value[index], true
}

func isOTLPIdentifierCharacter(char byte) bool {
	return char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}

func normalizeOTLPAttributeMap(otlp map[string]any, field string) map[string]any {
	switch attributes := otlp[field].(type) {
	case map[string]any:
		return attributes
	case map[string]string:
		normalized := make(map[string]any, len(attributes))
		for name, value := range attributes {
			normalized[name] = value
		}
		otlp[field] = normalized
		return normalized
	default:
		return nil
	}
}
