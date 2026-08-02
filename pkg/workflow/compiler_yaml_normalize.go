package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerYAMLNormalizeLog = logger.New("workflow:compiler_yaml_normalize")

// maxConsecutiveBlankLines is the largest run of blank lines allowed in the
// normalized YAML. yamllint's empty-lines rule flags more than two consecutive
// blank lines (max: 2), so runs are capped here to keep generated lock files clean.
const maxConsecutiveBlankLines = 2

// normalizeBlankLines rewrites the assembled workflow YAML so that:
//   - whitespace-only lines are emitted as truly empty lines,
//   - trailing whitespace is trimmed from non-block-scalar content lines,
//   - no more than maxConsecutiveBlankLines blank lines appear in a row outside
//     block scalars, and
//   - the file ends with exactly one trailing newline (no trailing blank line).
//
// YAML literal/folded block scalars carry raw payload content, so their non-blank
// lines and blank-line runs must be preserved exactly. This pass therefore only
// trims/caps generator-owned structural YAML, while still clearing indentation-only
// blank lines everywhere to remove yamllint trailing-spaces and empty-lines noise.
//
// This implementation avoids strings.Split/strings.Join to reduce allocations: it scans
// the input byte-by-byte and builds the result with a single pre-allocated strings.Builder.
func normalizeBlankLines(yamlContent string) string {
	compilerYAMLNormalizeLog.Printf("Normalizing blank lines in %d bytes of YAML", len(yamlContent))
	state := newYAMLNormalizationState(len(yamlContent))
	pos := 0
	for pos < len(yamlContent) {
		end := strings.IndexByte(yamlContent[pos:], '\n')
		state.processLine(extractYAMLLine(yamlContent, pos, end))
		if end == -1 {
			break
		}
		pos += end + 1
	}

	// When lastNonBlankEnd is still 0 there were no non-blank lines at all
	// (empty input or all-whitespace). Return a single newline, which matches
	// the original strings.TrimRight(…, "\n") + "\n" behaviour for that case.
	// NOTE: b.String()[:0] must NOT be used here; the early return is intentional.
	if state.lastNonBlankEnd == 0 {
		compilerYAMLNormalizeLog.Print("Input contained no non-blank lines, returning single newline")
		return "\n"
	}
	compilerYAMLNormalizeLog.Printf("Normalized YAML to %d bytes", state.lastNonBlankEnd)
	return state.result()
}

type yamlNormalizationState struct {
	builder                 strings.Builder
	lastNonBlankEnd         int
	blankRun                int
	inBlockScalar           bool
	pendingBlockScalar      bool
	blockScalarHeaderIndent int
	blockScalarIndent       int
}

func newYAMLNormalizationState(capacity int) *yamlNormalizationState {
	var builder strings.Builder
	builder.Grow(capacity)
	return &yamlNormalizationState{builder: builder}
}

func extractYAMLLine(content string, pos, end int) string {
	if end == -1 {
		return content[pos:]
	}
	return content[pos : pos+end]
}

func (s *yamlNormalizationState) processLine(line string) {
	trimmed := strings.TrimRight(line, " \t")
	if s.processBlockScalarLine(line, trimmed) {
		return
	}
	s.processStructuralLine(trimmed)
}

func (s *yamlNormalizationState) processBlockScalarLine(line, trimmed string) bool {
	if !s.pendingBlockScalar && !s.inBlockScalar {
		return false
	}
	if trimmed == "" {
		s.builder.WriteByte('\n')
		return true
	}
	lineIndent := countLeadingSpaces(line)
	if s.pendingBlockScalar {
		s.pendingBlockScalar = false
		if lineIndent > s.blockScalarHeaderIndent {
			s.blockScalarIndent = lineIndent
			s.inBlockScalar = true
		}
	}
	if !s.inBlockScalar || lineIndent < s.blockScalarIndent {
		s.inBlockScalar = false
		return false
	}
	s.builder.WriteString(line)
	s.builder.WriteByte('\n')
	s.lastNonBlankEnd = s.builder.Len()
	return true
}

func (s *yamlNormalizationState) processStructuralLine(trimmed string) {
	if trimmed == "" {
		if s.blankRun < maxConsecutiveBlankLines {
			s.builder.WriteByte('\n')
			s.blankRun++
		}
		return
	}
	s.builder.WriteString(trimmed)
	s.builder.WriteByte('\n')
	s.lastNonBlankEnd = s.builder.Len()
	s.blankRun = 0
	if headerIndent, ok := blockScalarHeaderIndentForLine(trimmed); ok {
		s.pendingBlockScalar = true
		s.blockScalarHeaderIndent = headerIndent
	}
}

func (s *yamlNormalizationState) result() string {
	return s.builder.String()[:s.lastNonBlankEnd]
}

func countLeadingSpaces(line string) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}

func blockScalarHeaderIndentForLine(line string) (int, bool) {
	colon := strings.LastIndexByte(line, ':')
	if colon == -1 {
		return 0, false
	}

	rest := strings.TrimSpace(line[colon+1:])
	if rest == "" || (rest[0] != '|' && rest[0] != '>') {
		return 0, false
	}

	rest = rest[1:]
	for rest != "" {
		switch rest[0] {
		case '+', '-':
			rest = rest[1:]
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			rest = rest[1:]
		default:
			goto indicatorsDone
		}
	}

indicatorsDone:
	rest = strings.TrimSpace(rest)
	if rest != "" && rest[0] != '#' {
		return 0, false
	}

	return countLeadingSpaces(line), true
}
