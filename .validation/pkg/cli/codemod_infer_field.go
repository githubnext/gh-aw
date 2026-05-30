package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var inferFieldCodemodLog = logger.New("cli:codemod_infer_field")

// getInferToDisableModelInvocationCodemod creates a codemod that migrates the
// deprecated top-level 'infer' field to 'disable-model-invocation', inverting
// the boolean value (infer: false → disable-model-invocation: true).
func getInferToDisableModelInvocationCodemod() Codemod {
	return Codemod{
		ID:           "infer-to-disable-model-invocation",
		Name:         "Migrate 'infer' to 'disable-model-invocation'",
		Description:  "Migrates the deprecated top-level 'infer' field to 'disable-model-invocation', inverting the boolean value (e.g. infer: false → disable-model-invocation: true).",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			inferValue, hasInfer := frontmatter["infer"]
			if !hasInfer {
				return content, false, nil
			}

			// If disable-model-invocation is already present, treat it as authoritative
			// and just remove the now-invalid 'infer' line.
			if _, hasDisable := frontmatter["disable-model-invocation"]; hasDisable {
				inferFieldCodemodLog.Print("Both 'infer' and 'disable-model-invocation' present - removing 'infer'")
				newContent, applied, err := applyFrontmatterLineTransform(content, removeInferLine)
				if applied {
					inferFieldCodemodLog.Print("Removed 'infer' field (kept existing 'disable-model-invocation')")
				}
				return newContent, applied, err
			}

			inferBool, ok := inferValue.(bool)
			if !ok {
				inferFieldCodemodLog.Print("'infer' value is not a boolean - skipping migration")
				return content, false, nil
			}

			// 'infer' and 'disable-model-invocation' are semantic inverses:
			//   infer: false  →  disable-model-invocation: true
			//   infer: true   →  disable-model-invocation: false
			disableValue := !inferBool

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return migrateInferToDisableModelInvocation(lines, disableValue)
			})
			if applied {
				inferFieldCodemodLog.Printf("Migrated 'infer: %v' to 'disable-model-invocation: %v'", inferBool, disableValue)
			}
			return newContent, applied, err
		},
	}
}

func removeInferLine(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "infer:") {
			modified = true
			inferFieldCodemodLog.Printf("Removed 'infer' field on line %d", i+1)
			continue
		}
		result = append(result, line)
	}

	return result, modified
}

func migrateInferToDisableModelInvocation(lines []string, disableValue bool) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "infer:") {
			indent := getIndentation(line)
			newLine := fmt.Sprintf("%sdisable-model-invocation: %v", indent, disableValue)
			result = append(result, newLine)
			modified = true
			inferFieldCodemodLog.Printf("Replaced 'infer' with 'disable-model-invocation: %v' on line %d", disableValue, i+1)
			continue
		}

		result = append(result, line)
	}

	return result, modified
}
