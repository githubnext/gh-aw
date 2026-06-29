package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputDispatchRepositoryKeyCodemodLog = logger.New("cli:codemod_safe_output_dispatch_repository_key")

func getSafeOutputDispatchRepositoryKeyCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-dispatch-repository-key",
		Name:         "Rename safe-outputs.dispatch_repository to dispatch-repository",
		Description:  "Renames deprecated safe-outputs.dispatch_repository to safe-outputs.dispatch-repository.",
		IntroducedIn: "1.0.65",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if safeOutputDispatchRepositoryKeyHasBothKeys(frontmatter) {
				safeOutputDispatchRepositoryKeyCodemodLog.Print("WARN: safe-outputs has both dispatch_repository and dispatch-repository; manual review needed, skipping migration")
				return content, false, nil
			}
			if !safeOutputDispatchRepositoryKeyNeedsMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, renameSafeOutputDispatchRepositoryKey)
			if applied {
				safeOutputDispatchRepositoryKeyCodemodLog.Print("Renamed safe-outputs.dispatch_repository to safe-outputs.dispatch-repository")
			}
			return newContent, applied, err
		},
	}
}

func safeOutputDispatchRepositoryKeyHasBothKeys(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasOld := safeOutputsMap["dispatch_repository"]
	_, hasNew := safeOutputsMap["dispatch-repository"]
	return hasOld && hasNew
}

func safeOutputDispatchRepositoryKeyNeedsMigration(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasOld := safeOutputsMap["dispatch_repository"]
	_, hasNew := safeOutputsMap["dispatch-repository"]
	return hasOld && !hasNew
}

func renameSafeOutputDispatchRepositoryKey(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && !strings.HasPrefix(trimmed, "#") {
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent == safeOutputsChildIndent && strings.HasPrefix(trimmed, "dispatch_repository:") {
				newLine, replaced := findAndReplaceInLine(line, "dispatch_repository", "dispatch-repository")
				if replaced {
					result = append(result, newLine)
					modified = true
					safeOutputDispatchRepositoryKeyCodemodLog.Printf("Renamed dispatch_repository to dispatch-repository in safe-outputs on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
