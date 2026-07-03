package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputDeduplicateByTitleIntegerCodemodLog = logger.New("cli:codemod_safe_output_deduplicate_by_title_integer")

var deduplicateByTitleIntegerValuePattern = regexp.MustCompile(`^(\s*)(\d+)(\s*)(#.*)?$`)

func getSafeOutputDeduplicateByTitleIntegerCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-deduplicate-by-title-integer-to-boolean",
		Name:         "Convert legacy deduplicate-by-title integers to booleans",
		Description:  "Converts legacy integer safe-outputs.create-issue deduplicate-by-title values to boolean true.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !safeOutputsCreateIssueNeedsDeduplicateByTitleIntegerMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, convertCreateIssueDeduplicateByTitleIntegersToBoolean)
			if applied {
				safeOutputDeduplicateByTitleIntegerCodemodLog.Print("Converted legacy create-issue deduplicate-by-title integers to boolean true")
			}
			return newContent, applied, err
		},
	}
}

func safeOutputsCreateIssueNeedsDeduplicateByTitleIntegerMigration(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	createIssueAny, ok := safeOutputsMap["create-issue"]
	if !ok {
		return false
	}
	createIssueMap, ok := createIssueAny.(map[string]any)
	if !ok {
		return false
	}
	value, ok := createIssueMap["deduplicate-by-title"]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case int, int64, uint64:
		return true
	case float64:
		return v >= 0 && float64(int(v)) == v
	default:
		return false
	}
}

func convertCreateIssueDeduplicateByTitleIntegersToBoolean(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	inCreateIssue := false
	createIssueIndent := ""
	createIssueChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
				inCreateIssue = false
				createIssueIndent = ""
				createIssueChildIndent = ""
			}
			if inCreateIssue && hasExitedBlock(line, createIssueIndent) {
				inCreateIssue = false
				createIssueIndent = ""
				createIssueChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			inCreateIssue = false
			createIssueIndent = ""
			createIssueChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent == safeOutputsChildIndent && strings.TrimSuffix(trimmed, ":") == "create-issue" {
				inCreateIssue = true
				createIssueIndent = indent
				createIssueChildIndent = ""
			} else if indent == safeOutputsChildIndent {
				inCreateIssue = false
				createIssueIndent = ""
				createIssueChildIndent = ""
			}
			result = append(result, line)
			continue
		}

		if inCreateIssue && createIssueChildIndent == "" && isDescendant(indent, createIssueIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "- ") {
			createIssueChildIndent = indent
		}

		if inCreateIssue && indent == createIssueChildIndent && strings.HasPrefix(trimmed, "deduplicate-by-title:") {
			newLine, converted := convertDeduplicateByTitleIntegerLineToBoolean(line)
			if converted {
				result = append(result, newLine)
				modified = true
				safeOutputDeduplicateByTitleIntegerCodemodLog.Printf("Converted deduplicate-by-title integer to boolean on line %d", i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}

func convertDeduplicateByTitleIntegerLineToBoolean(line string) (string, bool) {
	indent := getIndentation(line)
	trimmedLine := strings.TrimSpace(line)
	valuePart := strings.TrimPrefix(trimmedLine, "deduplicate-by-title:")

	matches := deduplicateByTitleIntegerValuePattern.FindStringSubmatch(valuePart)
	if matches == nil {
		return line, false
	}

	trailingWS := matches[3]
	comment := matches[4]
	if comment != "" {
		return fmt.Sprintf("%sdeduplicate-by-title: true%s%s", indent, trailingWS, comment), true
	}
	return indent + "deduplicate-by-title: true", true
}
