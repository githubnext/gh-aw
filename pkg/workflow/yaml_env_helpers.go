package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var yamlEnvHelpersLog = logger.New("workflow:yaml_env_helpers")

// writeYAMLEnv emits a single YAML env variable with proper escaping.
// Uses %q to produce a valid YAML double-quoted scalar that escapes ", \, newlines, and control characters,
// preventing YAML structure injection from frontmatter-derived values.
func writeYAMLEnv(b *strings.Builder, indent, key, value string) {
	fmt.Fprintf(b, "%s%s: %q\n", indent, key, value)
}

// formatYAMLEnv returns a properly escaped YAML env variable string.
// Uses %q to produce a valid YAML double-quoted scalar — safe for use anywhere a string is needed.
func formatYAMLEnv(indent, key, value string) string {
	return fmt.Sprintf("%s%s: %q\n", indent, key, value)
}

func quoteYAMLValueContainingColonSpace(value string) string {
	if value == "" ||
		strings.HasPrefix(value, "\"") ||
		strings.HasPrefix(value, "'") ||
		strings.HasPrefix(value, "|") ||
		strings.HasPrefix(value, ">") ||
		strings.HasPrefix(value, "{") ||
		strings.HasPrefix(value, "[") {
		return value
	}
	if strings.Contains(value, ": ") {
		return strconv.Quote(value)
	}
	return value
}

// quoteEnvValuesContainingColonSpace patches YAML text in-place for env blocks,
// quoting direct env values that contain ": " so they remain valid scalars.
//
// Assumptions:
//   - Input YAML is compiler-generated and consistently indented.
//   - We only rewrite direct children of env: maps (not nested mappings).
//   - Both "env:" and "- env:" are handled because env can appear either as a
//     regular mapping key or inline on a list item in YAML syntax.
func quoteEnvValuesContainingColonSpace(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	yamlEnvHelpersLog.Printf("Scanning %d YAML lines for env values needing quoting", len(lines))
	inEnv := false
	envIndent := 0
	envChildIndent := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if inEnv && indent <= envIndent {
			inEnv = false
			envChildIndent = -1
		}

		if trimmed == "env:" || trimmed == "- env:" {
			inEnv = true
			envIndent = indent
			envChildIndent = -1
			continue
		}
		if !inEnv || indent <= envIndent {
			continue
		}
		if envChildIndent == -1 {
			envChildIndent = indent
		}
		if indent != envChildIndent {
			continue
		}

		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		quotedValue := quoteYAMLValueContainingColonSpace(line[idx+2:])
		if quotedValue != line[idx+2:] {
			lines[i] = line[:idx+2] + quotedValue
			yamlEnvHelpersLog.Printf("Quoted env value on line %d containing ': '", i+1)
		}
	}

	return strings.Join(lines, "\n")
}
