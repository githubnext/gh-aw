package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var runsOnSnippetLog = logger.New("workflow:runs_on_snippet")

func runsOnMarshalOptions() []yaml.EncodeOption {
	opts := append([]yaml.EncodeOption{}, DefaultMarshalOptions...)
	return append(opts, yaml.IndentSequence(true))
}

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
		yamlBytes, err = yaml.MarshalWithOptions(yaml.MapSlice{{Key: "runs-on", Value: orderedValue}}, runsOnMarshalOptions()...)
	} else {
		yamlBytes, err = yaml.MarshalWithOptions(map[string]any{"runs-on": value}, runsOnMarshalOptions()...)
	}
	if err != nil {
		runsOnSnippetLog.Printf("Failed to marshal runs-on snippet: %v", err)
		return ""
	}

	return strings.TrimSuffix(string(yamlBytes), "\n")
}

func normalizeRunsOnSnippet(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	snippet := value
	if !strings.HasPrefix(snippet, "runs-on:") {
		snippet = "runs-on: " + snippet
	}

	var snippetMap map[string]any
	if err := yaml.Unmarshal([]byte(snippet), &snippetMap); err == nil {
		if runsOnValue, ok := snippetMap["runs-on"]; ok {
			if rendered := renderRunsOnSnippet(runsOnValue); rendered != "" {
				return rendered
			}
		}
	} else {
		runsOnSnippetLog.Printf("Could not parse runs-on snippet as YAML map, using raw form: %v", err)
	}
	return ensureRunsOnContinuationIndent(snippet)
}

func ensureRunsOnContinuationIndent(snippet string) string {
	lines := strings.Split(snippet, "\n")
	if len(lines) <= 1 {
		return snippet
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" || strings.HasPrefix(lines[i], " ") {
			continue
		}
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}
