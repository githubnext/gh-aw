package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var inlineSectionLog = logger.New("parser:inline_section_helpers")

var h2HeadingRegex = regexp.MustCompile(`(?m)^##[ \t]`)

// blankLineRunRegex matches three or more consecutive newlines, which can
// appear at the seam where an explicitly-closed inline section is removed
// from between two chunks of surrounding main markdown.
var blankLineRunRegex = regexp.MustCompile(`\n{3,}`)

// collapseBlankLines collapses runs of 3+ consecutive newlines down to a
// single blank line (two newlines), keeping reassembled main markdown tidy.
func collapseBlankLines(s string) string {
	return blankLineRunRegex.ReplaceAllString(s, "\n\n")
}

func collectH2Positions(markdown string) []int {
	var h2Positions []int
	for _, m := range h2HeadingRegex.FindAllStringIndex(markdown, -1) {
		h2Positions = append(h2Positions, m[0])
	}
	return h2Positions
}

func nextH2After(offset int, h2Positions []int, markdownLength int) int {
	for _, pos := range h2Positions {
		if pos >= offset {
			return pos
		}
	}
	return markdownLength
}

// inlineSectionEndMarker describes a single explicit end-marker match found in
// the markdown (e.g. "## end skill: `name`" or "## end agent: `name`").
type inlineSectionEndMarker struct {
	name  string // the identifier captured from the end marker
	start int    // byte offset where the end marker's heading line begins
	end   int    // byte offset right after the end marker's line (incl. newline)
}

// collectInlineSectionEndMarkers scans markdown for all matches of endRegex,
// which must have a single capture group for the section name.
func collectInlineSectionEndMarkers(markdown string, endRegex *regexp.Regexp) []inlineSectionEndMarker {
	var markers []inlineSectionEndMarker
	for _, m := range endRegex.FindAllStringSubmatchIndex(markdown, -1) {
		lineEnd := m[1]
		if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
			lineEnd++
		}
		markers = append(markers, inlineSectionEndMarker{
			name:  markdown[m[2]:m[3]],
			start: m[0],
			end:   lineEnd,
		})
	}
	return markers
}

// extractInlineSections collects all sections delimited by marker positions in
// markdown. The caller provides allStarts (already validated non-empty), the
// endRegex used to recognize explicit end markers for this section type (e.g.
// "## end skill: `name`"), and a makeItem factory that converts a
// (name, content) pair into the desired result type T.
//
// A section normally ends at the next level-2 Markdown heading (##) or EOF —
// whichever comes first (legacy, implicit closing). If an explicit end marker
// matching the section's name is found before the next start marker (or EOF),
// the section's content is bounded by that end marker instead, and any text
// following the end marker (up to the next start marker) is preserved as main
// markdown rather than discarded. This lets an inline section be embedded in
// the middle of a document — for example via an import — without swallowing
// unrelated content that follows it.
//
// It returns the reassembled main markdown and the collected items. An error
// is returned if an end marker references a name that does not correspond to
// any start marker of the same type (an "orphan" end marker), which is almost
// always an authoring mistake (e.g. a typo in the name).
func extractInlineSections[T any](markdown string, allStarts [][]int, endRegex *regexp.Regexp, makeItem func(name, content string) T) (mainMarkdown string, items []T, err error) {
	h2Positions := collectH2Positions(markdown)
	endMarkers := collectInlineSectionEndMarkers(markdown, endRegex)
	usedEnd := make([]bool, len(endMarkers))

	var mainParts []string
	cursor := 0
	prevExplicit := true // text before the first marker is always kept
	for i, m := range allStarts {
		if prevExplicit {
			mainParts = append(mainParts, markdown[cursor:m[0]])
		}

		name := markdown[m[2]:m[3]]
		lineEnd := m[1]
		if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
			lineEnd++
		}

		windowEnd := len(markdown)
		if i+1 < len(allStarts) {
			windowEnd = allStarts[i+1][0]
		}

		var matchedEnd *inlineSectionEndMarker
		for ei := range endMarkers {
			e := &endMarkers[ei]
			if usedEnd[ei] || e.name != name || e.start < lineEnd || e.start >= windowEnd {
				continue
			}
			matchedEnd = e
			usedEnd[ei] = true
			break
		}

		var content string
		var newCursor int
		explicit := matchedEnd != nil
		if explicit {
			content = strings.TrimSpace(markdown[lineEnd:matchedEnd.start])
			newCursor = matchedEnd.end
			inlineSectionLog.Printf("Extracted inline section %q (%d bytes of content, explicit end marker)", name, len(content))
		} else {
			contentEnd := nextH2After(lineEnd, h2Positions, len(markdown))
			content = strings.TrimSpace(markdown[lineEnd:contentEnd])
			newCursor = contentEnd
			inlineSectionLog.Printf("Extracted inline section %q (%d bytes of content, implicit end)", name, len(content))
		}

		items = append(items, makeItem(name, content))
		cursor = newCursor
		prevExplicit = explicit
	}
	if prevExplicit {
		mainParts = append(mainParts, markdown[cursor:])
	}
	mainMarkdown = strings.TrimRight(collapseBlankLines(strings.Join(mainParts, "")), "\n")

	for ei, used := range usedEnd {
		if !used {
			return "", nil, fmt.Errorf("end marker for unknown section %q (no matching start marker with that name)", endMarkers[ei].name)
		}
	}

	return mainMarkdown, items, nil
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
