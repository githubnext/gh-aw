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

	n := newBlankLineNormalizer(len(yamlContent))
	pos := 0
	for pos < len(yamlContent) {
		end, line := nextYAMLLine(yamlContent, pos)
		n.writeLine(line)
		if end == -1 {
			break
		}
		pos += end + 1
	}

	return n.result()
}

type blankLineNormalizer struct {
	b                       strings.Builder
	lastNonBlankEnd         int
	blankRun                int
	inBlockScalar           bool
	pendingBlockScalar      bool
	blockScalarHeaderIndent int
	blockScalarIndent       int
}

func newBlankLineNormalizer(size int) *blankLineNormalizer {
	n := &blankLineNormalizer{}
	n.b.Grow(size)
	return n
}

func nextYAMLLine(content string, pos int) (int, string) {
	end := strings.IndexByte(content[pos:], '\n')
	if end == -1 {
		return end, content[pos:]
	}
	return end, content[pos : pos+end]
}

func (n *blankLineNormalizer) writeLine(line string) {
	processStructuralLine, trimmed := n.writeBlockScalarLine(line)
	if !processStructuralLine {
		return
	}
	n.writeStructuralLine(trimmed)
}

func (n *blankLineNormalizer) writeBlockScalarLine(line string) (bool, string) {
	trimmed := strings.TrimRight(line, " \t")
	if !n.pendingBlockScalar && !n.inBlockScalar {
		return true, trimmed
	}
	if trimmed == "" {
		n.b.WriteByte('\n')
		return false, trimmed
	}
	lineIndent := countLeadingSpaces(line)
	if n.pendingBlockScalar {
		if lineIndent <= n.blockScalarHeaderIndent {
			n.pendingBlockScalar = false
		} else {
			n.blockScalarIndent = lineIndent
			n.inBlockScalar = true
			n.pendingBlockScalar = false
		}
	}
	if n.inBlockScalar && lineIndent >= n.blockScalarIndent {
		n.b.WriteString(line)
		n.b.WriteByte('\n')
		n.lastNonBlankEnd = n.b.Len()
		return false, trimmed
	}
	n.inBlockScalar = false
	return true, trimmed
}

func (n *blankLineNormalizer) writeStructuralLine(trimmed string) {
	if trimmed == "" {
		if n.blankRun < maxConsecutiveBlankLines {
			n.b.WriteByte('\n')
			n.blankRun++
		}
		return
	}
	n.b.WriteString(trimmed)
	n.b.WriteByte('\n')
	n.lastNonBlankEnd = n.b.Len()
	n.blankRun = 0
	if headerIndent, ok := blockScalarHeaderIndentForLine(trimmed); ok {
		n.pendingBlockScalar = true
		n.blockScalarHeaderIndent = headerIndent
	}
}

func (n *blankLineNormalizer) result() string {
	// When lastNonBlankEnd is still 0 there were no non-blank lines at all
	// (empty input or all-whitespace). Return a single newline, which matches
	// the original strings.TrimRight(…, "\n") + "\n" behaviour for that case.
	// NOTE: b.String()[:0] must NOT be used here; the early return is intentional.
	if n.lastNonBlankEnd == 0 {
		compilerYAMLNormalizeLog.Print("Input contained no non-blank lines, returning single newline")
		return "\n"
	}
	// Slice the builder string to drop trailing blank lines. b.String() copies
	// the builder's internal buffer into a new string once; the slice avoids a
	// second copy that a separate strings.Builder trim would incur.
	compilerYAMLNormalizeLog.Printf("Normalized YAML to %d bytes", n.lastNonBlankEnd)
	return n.b.String()[:n.lastNonBlankEnd]
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
