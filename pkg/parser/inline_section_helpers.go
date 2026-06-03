package parser

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var inlineSectionLog = logger.New("parser:inline_section_helpers")

var h2HeadingRegex = regexp.MustCompile(`(?m)^##[ \t]`)

func collectH2Positions(markdown string) []int {
	var h2Positions []int
	for _, m := range h2HeadingRegex.FindAllStringIndex(markdown, -1) {
		h2Positions = append(h2Positions, m[0])
	}
	return h2Positions
}

func extractInlineSection(markdown string, marker []int, h2Positions []int) (string, string) {
	name := markdown[marker[2]:marker[3]]
	lineEnd := marker[1]
	if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
		lineEnd++
	}
	contentEnd := nextH2After(lineEnd, h2Positions, len(markdown))
	content := strings.TrimSpace(markdown[lineEnd:contentEnd])
	inlineSectionLog.Printf("Extracted inline section %q (%d bytes of content)", name, len(content))
	return name, content
}

func nextH2After(offset int, h2Positions []int, markdownLength int) int {
	for _, pos := range h2Positions {
		if pos >= offset {
			return pos
		}
	}
	return markdownLength
}

func validateUniqueInlineSectionNames(markdown string, allStarts [][]int, createDuplicateError func(name string) error) error {
	seen := make(map[string]struct{})
	for _, m := range allStarts {
		name := markdown[m[2]:m[3]]
		if _, exists := seen[name]; exists {
			inlineSectionLog.Printf("Duplicate inline section name detected: %q", name)
			return createDuplicateError(name)
		}
		seen[name] = struct{}{}
	}
	inlineSectionLog.Printf("Validated %d unique inline section name(s)", len(seen))
	return nil
}
