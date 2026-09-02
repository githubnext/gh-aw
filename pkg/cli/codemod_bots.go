package cli

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var botsCodemodLog = logger.New("cli:codemod_bots")

// getBotsToOnBotsCodemod creates a codemod for moving top-level 'bots' to 'on.bots'
func getBotsToOnBotsCodemod() Codemod {
	codemod := newMoveTopLevelKeyToOnBlockCodemod(moveToOnBlockConfig{
		ID:           "bots-to-on-bots",
		Name:         "Move bots to on.bots",
		Description:  "Moves the top-level 'bots' field to 'on.bots' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		FieldKey:     "bots",
		IsInlineSingle: func(v string) bool {
			return strings.HasPrefix(v, "[")
		},
		Log: botsCodemodLog,
	})
	baseApply := codemod.Apply
	codemod.Apply = func(content string, frontmatter map[string]any) (string, bool, error) {
		topBots, hasTopBots := frontmatter["bots"]
		onMap, hasOnMap := frontmatter["on"].(map[string]any)
		onBots, hasOnBots := onMap["bots"]
		if !hasTopBots || !hasOnMap || !hasOnBots {
			return baseApply(content, frontmatter)
		}

		mergedBots, ok := mergeLegacyBots(onBots, topBots)
		if !ok {
			return content, false, nil
		}
		return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
			return mergeLegacyBotsLines(lines, mergedBots)
		})
	}
	return codemod
}

func mergeLegacyBots(onBots, topBots any) ([]string, bool) {
	merged := make([]string, 0)
	seen := make(map[string]struct{})
	for _, value := range []any{onBots, topBots} {
		bots, ok := value.([]any)
		if !ok {
			return nil, false
		}
		for _, botValue := range bots {
			bot, ok := botValue.(string)
			if !ok {
				return nil, false
			}
			if _, exists := seen[bot]; !exists {
				seen[bot] = struct{}{}
				merged = append(merged, bot)
			}
		}
	}
	return merged, true
}

func mergeLegacyBotsLines(lines []string, bots []string) ([]string, bool) {
	topBotsStart, topBotsEnd := findBotsBlock(lines, 0, len(lines), 0, false)
	onStart := -1
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "on:") {
			onStart = i
			break
		}
	}
	if topBotsStart == -1 || onStart == -1 {
		return lines, false
	}

	onEnd := len(lines)
	for i := onStart + 1; i < len(lines); i++ {
		if isTopLevelKey(lines[i]) {
			onEnd = i
			break
		}
	}
	onBotsStart, onBotsEnd := findBotsBlock(lines, onStart+1, onEnd, len(getIndentation(lines[onStart])), true)
	if onBotsStart == -1 {
		return lines, false
	}
	onBotsIndent := getIndentation(lines[onBotsStart])

	encodedBots, err := json.Marshal(bots)
	if err != nil {
		return lines, false
	}
	result := make([]string, 0, len(lines))
	for i, line := range lines {
		if (i >= topBotsStart && i <= topBotsEnd) || (i >= onBotsStart && i <= onBotsEnd) {
			continue
		}
		result = append(result, line)
		if i == onStart {
			result = append(result, onBotsIndent+"bots: "+string(encodedBots))
		}
	}
	return result, true
}

func findBotsBlock(lines []string, start, end, indent int, nested bool) (int, int) {
	for i := start; i < end; i++ {
		line := lines[i]
		lineIndent := len(getIndentation(line))
		if (!nested && lineIndent != indent) || (nested && lineIndent <= indent) || !strings.HasPrefix(strings.TrimSpace(line), "bots:") {
			continue
		}
		blockEnd := i
		for j := i + 1; j < end && isNestedUnder(lines[j], getIndentation(line)); j++ {
			blockEnd = j
		}
		return i, blockEnd
	}
	return -1, -1
}
