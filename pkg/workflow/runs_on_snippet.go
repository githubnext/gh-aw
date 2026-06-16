package workflow

import (
	"strings"

	"github.com/goccy/go-yaml"
)

// renderRunsOnSnippet serializes a runs-on value into a "runs-on: ..." YAML snippet.
// Returns empty string for empty/unset values.
func renderRunsOnSnippet(value any) string {
	if isEmptyRunsOnValue(value) {
		return ""
	}

	var yamlBytes []byte
	var err error
	if valueMap, ok := value.(map[string]any); ok {
		orderedValue := OrderMapFields(valueMap, []string{})
		yamlBytes, err = yaml.MarshalWithOptions(yaml.MapSlice{{Key: "runs-on", Value: orderedValue}}, DefaultMarshalOptions...)
	} else {
		yamlBytes, err = yaml.Marshal(map[string]any{"runs-on": value})
	}
	if err != nil {
		return ""
	}

	return strings.TrimSuffix(string(yamlBytes), "\n")
}
