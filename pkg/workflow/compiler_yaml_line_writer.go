package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var yamlLineWriterLog = logger.New("workflow:compiler_yaml_line_writer")

// yamlBlockScalarState tracks whether the scanner is currently inside a YAML
// literal (|) or folded (>) block scalar. It is used by appendYAMLLine to
// decide whether a line's trailing whitespace may be safely trimmed.
//
// The zero value is ready to use. Create a new instance per YAML stream and
// pass it to every appendYAMLLine call in that stream.
type yamlBlockScalarState struct {
	pending      bool // saw a block-scalar header; waiting for the first content line
	active       bool // inside a block scalar's payload
	headerIndent int  // leading-space count of the block-scalar header line
	bodyIndent   int  // leading-space count of the first payload line
}

// update advances the block-scalar state machine for sourceLine (the raw line
// as it appears in the source YAML, before any prefix or indentation adjustment
// is applied). It returns true when sourceLine is part of a block scalar's
// payload and its content must NOT be right-trimmed.
func (s *yamlBlockScalarState) update(sourceLine string) bool {
	trimmed := strings.TrimRight(sourceLine, " \t")

	if s.pending || s.active {
		if trimmed == "" {
			// Blank lines keep the state alive without advancing it:
			// inside an active scalar they are payload; inside a pending
			// scalar they simply delay the first content line.
			return s.active
		}
		lineIndent := countLeadingSpaces(sourceLine)
		if s.pending {
			if lineIndent <= s.headerIndent {
				// No indented content followed the header; leave the scalar.
				s.pending = false
				// Fall through to structural handling below.
			} else {
				s.bodyIndent = lineIndent
				s.active = true
				s.pending = false
				yamlLineWriterLog.Printf("Entering block scalar payload: headerIndent=%d, bodyIndent=%d", s.headerIndent, s.bodyIndent)
				return true // first payload line
			}
		}
		if s.active {
			if lineIndent < s.bodyIndent {
				// Outdented non-blank line: we have left the block scalar.
				s.active = false
				yamlLineWriterLog.Printf("Leaving block scalar payload: lineIndent=%d < bodyIndent=%d", lineIndent, s.bodyIndent)
				// Fall through to structural handling below.
			} else {
				return true // still inside the block scalar
			}
		}
	}

	// Structural line: detect whether it introduces a new block scalar.
	if trimmed != "" {
		if headerIndent, ok := blockScalarHeaderIndentForLine(trimmed); ok {
			s.pending = true
			s.headerIndent = headerIndent
			yamlLineWriterLog.Printf("Detected block scalar header: headerIndent=%d", headerIndent)
		}
	}
	return false
}

// appendYAMLLine writes content to b with the given prefix.
//
//   - Blank content (empty or whitespace-only) is always written as a bare
//     newline with no prefix, regardless of isBlockScalarContent, to avoid
//     emitting lines that contain only spaces (yamllint trailing-spaces).
//   - When isBlockScalarContent is true, non-blank content is written verbatim
//     so that trailing whitespace that is semantically significant inside a
//     shell script or template literal is preserved.
//   - When isBlockScalarContent is false, trailing whitespace is trimmed before
//     writing so that yamllint trailing-spaces warnings are suppressed.
//
// Callers obtain isBlockScalarContent by calling yamlBlockScalarState.update
// with the original source line. When the source indentation is adjusted before
// writing (e.g. a 2-space prefix is stripped in writeStepsSection), callers
// must still pass the original source line to update so that indent-based
// entry/exit detection remains accurate, and then pass the adjusted content
// here.
func appendYAMLLine(b *strings.Builder, prefix, content string, isBlockScalarContent bool) {
	// Always emit blank lines as bare newlines to avoid trailing-space violations.
	if strings.TrimRight(content, " \t") == "" {
		b.WriteByte('\n')
		return
	}
	if isBlockScalarContent {
		b.WriteString(prefix)
		b.WriteString(content)
		b.WriteByte('\n')
		return
	}
	b.WriteString(prefix)
	b.WriteString(strings.TrimRight(content, " \t"))
	b.WriteByte('\n')
}
