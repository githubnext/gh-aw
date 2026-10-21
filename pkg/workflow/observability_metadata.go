package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

const otlpMetadataNamespace = "github.aw.metadata"

var otlpMetadataReferencePattern = regexp.MustCompile(`^\s*\$\{\{\s*github\.aw\.metadata\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}\s*$`)

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
		for name, value := range attributes {
			text, ok := value.(string)
			if !ok || !strings.Contains(text, otlpMetadataNamespace) {
				continue
			}
			scope := "observability.otlp." + field + "." + name
			if !otlpMetadataReferencePattern.MatchString(text) {
				return fmt.Errorf("%s must be a whole-value reference in the form ${{ github.aw.metadata.<key> }}", scope)
			}
			key := strings.TrimSpace(text)
			key = strings.TrimSpace(strings.TrimPrefix(key, "${{"))
			key = strings.TrimSpace(strings.TrimSuffix(key, "}}"))
			key = strings.TrimSpace(strings.TrimPrefix(key, otlpMetadataNamespace+"."))
			if metadata == nil {
				return fmt.Errorf("%s references metadata key %q, but the root workflow has no metadata map", scope, key)
			}
			metadataValue, exists := metadata[key]
			if !exists {
				return fmt.Errorf("%s references missing metadata key %q", scope, key)
			}
			resolved, ok := metadataValue.(string)
			if !ok {
				return fmt.Errorf("%s references metadata key %q, which must be a string", scope, key)
			}
			if strings.Contains(resolved, "${{") || strings.Contains(resolved, "}}") {
				return fmt.Errorf("%s cannot resolve metadata key %q because its value contains GitHub Actions expression delimiters", scope, key)
			}
			attributes[name] = resolved
		}
	}
	return nil
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
