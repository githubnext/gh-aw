package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

func migrateEngineFieldToTopLevel(
	content string,
	frontmatter map[string]any,
	engineField string,
	targetTopLevelField string,
	preserveTopLevelFields []string,
	log *logger.Logger,
	skipInlineMessage string,
	removedMessage string,
	migratedMessage string,
) (string, bool, error) {
	engineValue, hasEngine := frontmatter["engine"]
	if !hasEngine {
		return content, false, nil
	}
	engineMap, ok := engineValue.(map[string]any)
	if !ok {
		return content, false, nil
	}
	if _, hasEngineField := engineMap[engineField]; !hasEngineField {
		return content, false, nil
	}

	hasPreservedTopLevelField := false
	for _, field := range preserveTopLevelFields {
		if _, exists := frontmatter[field]; exists {
			hasPreservedTopLevelField = true
			break
		}
	}

	return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !isTopLevelKey(line) || !strings.HasPrefix(trimmed, "engine:") {
				continue
			}
			inlineValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "engine:"))
			if strings.HasPrefix(inlineValue, "{") && strings.Contains(inlineValue, engineField+":") {
				if log != nil {
					log.Print(skipInlineMessage)
				}
				return lines, false
			}
		}

		fieldSuffix := ""
		inEngineBlock := false
		engineIndent := ""
		engineFieldPrefix := engineField + ":"
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
				inEngineBlock = true
				engineIndent = getIndentation(line)
				continue
			}
			if inEngineBlock && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && len(getIndentation(line)) <= len(engineIndent) {
				inEngineBlock = false
			}
			if inEngineBlock && strings.HasPrefix(trimmed, engineFieldPrefix) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					fieldSuffix = parts[1]
				}
				break
			}
		}

		result, removed := removeFieldFromBlock(lines, engineField, "engine")
		if !removed {
			return lines, false
		}

		if hasPreservedTopLevelField {
			if log != nil {
				log.Print(removedMessage)
			}
			return result, true
		}

		insertAt := 0
		for i, line := range result {
			if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "engine:") {
				insertAt = i
				break
			}
		}

		topLevelLine := targetTopLevelField + ":" + fieldSuffix
		withTopLevel := make([]string, 0, len(result)+1)
		withTopLevel = append(withTopLevel, result[:insertAt]...)
		withTopLevel = append(withTopLevel, topLevelLine)
		withTopLevel = append(withTopLevel, result[insertAt:]...)

		if log != nil {
			log.Print(migratedMessage)
		}
		return withTopLevel, true
	})
}
