package workflow

import "strings"

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
	var b strings.Builder
	b.Grow(len(yamlContent))

	// lastNonBlankEnd tracks the builder length immediately after writing the last
	// non-blank line (including its trailing newline). It starts at 0 and is only
	// advanced when a substantive line is written, so it stays 0 when all lines
	// are whitespace-only or the input is empty. Every line — blank or not — still
	// gets a '\n' written to b, so b.Len() and lastNonBlankEnd may diverge when
	// there are trailing blank lines.
	lastNonBlankEnd := 0
	// blankRun counts consecutive blank lines emitted since the last non-blank
	// structural line, so runs longer than maxConsecutiveBlankLines can be
	// collapsed outside block scalars.
	blankRun := 0
	inBlockScalar := false
	pendingBlockScalar := false
	blockScalarHeaderIndent := 0
	blockScalarIndent := 0
	pos := 0
	for pos < len(yamlContent) {
		// Find the end of the current line.
		end := strings.IndexByte(yamlContent[pos:], '\n')
		var line string
		if end == -1 {
			line = yamlContent[pos:]
		} else {
			line = yamlContent[pos : pos+end]
		}

		processStructuralLine := true
		trimmed := strings.TrimRight(line, " \t")
		if pendingBlockScalar || inBlockScalar {
			if trimmed == "" {
				// Whitespace-only lines inside block scalars are still semantically
				// blank, so emit them as empty lines but never cap the run.
				b.WriteByte('\n')
				processStructuralLine = false
			} else {
				lineIndent := countLeadingSpaces(line)
				if pendingBlockScalar {
					if lineIndent <= blockScalarHeaderIndent {
						pendingBlockScalar = false
					} else {
						blockScalarIndent = lineIndent
						inBlockScalar = true
						pendingBlockScalar = false
					}
				}
				if inBlockScalar {
					if lineIndent < blockScalarIndent {
						inBlockScalar = false
					} else {
						b.WriteString(line)
						b.WriteByte('\n')
						lastNonBlankEnd = b.Len()
						processStructuralLine = false
					}
				}
			}
		}

		if processStructuralLine {
			if trimmed == "" {
				// Blank structural line: emit at most maxConsecutiveBlankLines in a
				// row so yamllint's empty-lines rule is never exceeded. lastNonBlankEnd
				// is NOT updated here so that trailing blank lines (including a blank
				// final "line" produced by a file that ends with "\n\n" or by
				// whitespace-only text after the last real line) are excluded from the
				// returned slice.
				if blankRun < maxConsecutiveBlankLines {
					b.WriteByte('\n')
					blankRun++
				}
			} else {
				b.WriteString(trimmed)
				b.WriteByte('\n')
				lastNonBlankEnd = b.Len()
				blankRun = 0
				if headerIndent, ok := blockScalarHeaderIndentForLine(trimmed); ok {
					pendingBlockScalar = true
					blockScalarHeaderIndent = headerIndent
				}
			}
		}

		if end == -1 {
			break
		}
		pos += end + 1
	}

	// When lastNonBlankEnd is still 0 there were no non-blank lines at all
	// (empty input or all-whitespace). Return a single newline, which matches
	// the original strings.TrimRight(…, "\n") + "\n" behaviour for that case.
	// NOTE: b.String()[:0] must NOT be used here; the early return is intentional.
	if lastNonBlankEnd == 0 {
		return "\n"
	}
	// Slice the builder string to drop trailing blank lines. b.String() copies
	// the builder's internal buffer into a new string once; the slice avoids a
	// second copy that a separate strings.Builder trim would incur.
	return b.String()[:lastNonBlankEnd]
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
	for len(rest) > 0 {
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
