package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

var missingReportCodemodLog = logger.New("cli:codemod_safe_output_missing_reports")

func getSafeOutputMissingToolCodemod() Codemod {
	return newMissingReportCodemod("tool", []string{"missing-tool", "missing-tools"})
}

func getSafeOutputMissingDataCodemod() Codemod {
	return newMissingReportCodemod("data", []string{"missing-data"})
}

func newMissingReportCodemod(kind string, aliases []string) Codemod {
	target := "report-missing-" + kind
	return Codemod{
		ID:           "safe-output-report-missing-" + kind,
		Name:         "Rename missing " + kind + " reporting configuration",
		Description:  "Renames safe-outputs." + strings.Join(aliases, " and safe-outputs.") + " to safe-outputs." + target + ".",
		IntroducedIn: "1.5.0",
		Guided:       true,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			outputs, ok := frontmatter["safe-outputs"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			source := ""
			for _, alias := range aliases {
				if _, exists := outputs[alias]; exists {
					if _, conflict := outputs[target]; conflict || source != "" {
						return content, false, fmt.Errorf("safe-outputs has conflicting missing %s reporting keys; keep a single %s entry before running gh aw fix", kind, target)
					}
					source = alias
				}
			}
			if source == "" {
				return content, false, nil
			}
			return renameMissingReportKey(content, source, target)
		},
	}
}

func renameMissingReportKey(content, source, target string) (string, bool, error) {
	var transformErr error
	result, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		path, pathErr := yaml.PathString("$.safe-outputs")
		if pathErr != nil {
			transformErr = pathErr
			return lines, false
		}
		node, readErr := path.ReadNode(strings.NewReader(strings.Join(lines, "\n")))
		if readErr != nil {
			transformErr = readErr
			return lines, false
		}
		mapping, ok := node.(ast.MapNode)
		if !ok {
			transformErr = fmt.Errorf("safe-outputs.%s must be renamed to %s manually; the mapping uses an unsupported YAML form", source, target)
			return lines, false
		}
		return replaceMissingReportKeyToken(lines, mapping, source, target, &transformErr)
	})
	if transformErr != nil {
		return content, false, transformErr
	}
	if applied {
		missingReportCodemodLog.Printf("Renamed safe-outputs.%s to %s", source, target)
	}
	return result, applied, err
}

func replaceMissingReportKeyToken(lines []string, mapping ast.MapNode, source, target string, transformErr *error) ([]string, bool) {
	for it := mapping.MapRange(); it.Next(); {
		key := it.Key().GetToken()
		if key.Value != source {
			continue
		}
		lineIndex, column := key.Position.Line-1, key.Position.Column-1
		if lineIndex < 0 || lineIndex >= len(lines) {
			break
		}
		line := []rune(lines[lineIndex])
		if column < 0 || column >= len(line) {
			break
		}
		keyText := string(line[column:])
		if strings.HasPrefix(keyText, "'") || strings.HasPrefix(keyText, "\"") {
			column++
		}
		end := column + len([]rune(source))
		if column < 0 || end > len(line) || string(line[column:end]) != source {
			break
		}
		result := slices.Clone(lines)
		if lineIndex >= 0 && lineIndex < len(result) {
			result[lineIndex] = string(line[:column]) + target + string(line[end:])
			return result, true
		}
	}
	*transformErr = fmt.Errorf("cannot safely locate safe-outputs.%s; rename it to %s manually", source, target)
	return lines, false
}
