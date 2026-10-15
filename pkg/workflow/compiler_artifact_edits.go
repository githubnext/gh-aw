package workflow

import (
	stdcmp "cmp"
	"errors"
	"slices"
	"strings"
)

type artifactStepEdit struct {
	start int
	end   int
	value string
	steps []string
}

func artifactDownloadEdits(step artifactProtocolStep, sources map[string]string, job string) []artifactStepEdit {
	var edits []artifactStepEdit
	for _, key := range []string{"name", "pattern"} {
		value, ok := step.With[key].(string)
		if !ok {
			continue
		}
		fieldStart := strings.Index(step.raw, "          "+key+": ")
		if fieldStart < 0 || !strings.Contains(value, localArtifactPrefix) {
			continue
		}
		line, _, _ := strings.Cut(step.raw[fieldStart:], "\n")
		valueOffset := fieldStart + strings.Index(line, value)
		for offset := 0; offset < len(value); {
			index := strings.Index(value[offset:], localArtifactPrefix)
			if index < 0 {
				break
			}
			start := offset + index
			end := len(value)
			if strings.HasPrefix(value, "{") {
				// Skip the braces inside the prefix expression.
				end = start + len(localArtifactPrefix)
				if next := strings.IndexAny(value[end:], ",}"); next >= 0 {
					end += next
				} else {
					end = len(value)
				}
			}
			producer := sources[value[start:end]]
			if producer != "" && producer != job {
				edits = append(edits, artifactStepEdit{
					start: step.start + valueOffset + start, end: step.start + valueOffset + start + len(localArtifactPrefix),
					value: "${{ steps.artifact-source-" + producer + ".outputs.prefix }}",
				})
			}
			offset = start + len(localArtifactPrefix)
		}
	}
	return edits
}

func artifactDownloadGuardEdit(step artifactProtocolStep, guard string) artifactStepEdit {
	if step.If == "" {
		offset := step.start + strings.Index(step.raw, "\n") + 1
		if guard == "false" {
			guard = "${{ false }}"
		}
		return artifactStepEdit{start: offset, end: offset, value: "        if: " + guard + "\n"}
	}
	start := strings.Index(step.raw, "        if: ")
	line, _, _ := strings.Cut(step.raw[start:], "\n")
	original := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(step.If, "${{"), "}}"))
	return artifactStepEdit{start: step.start + start, end: step.start + start + len(line),
		value: "        if: ${{ (" + original + ") && " + guard + " }}"}
}

// Preserve existing compiler step chunks while changing only bound artifact inputs.
func applyArtifactStepEdits(original []string, edits []artifactStepEdit) ([]string, error) {
	slices.SortStableFunc(edits, func(a, b artifactStepEdit) int { return stdcmp.Compare(a.start, b.start) })
	var result []string
	base := 0
	remaining := edits
	for chunkIndex, chunk := range original {
		var rewritten strings.Builder
		cursor := 0
		consumed := 0
		for _, edit := range remaining {
			if edit.start >= base+len(chunk) && chunkIndex != len(original)-1 {
				break
			}
			start, end := edit.start-base, edit.end-base
			if start < cursor || end > len(chunk) {
				return nil, errors.New("artifact input edit crosses a compiler step chunk")
			}
			rewritten.WriteString(chunk[cursor:start])
			if len(edit.steps) > 0 {
				result = append(result, rewritten.String())
				rewritten.Reset()
				result = append(result, edit.steps...)
			} else {
				rewritten.WriteString(edit.value)
			}
			cursor = end
			consumed++
		}
		remaining = remaining[consumed:]
		rewritten.WriteString(chunk[cursor:])
		result = append(result, rewritten.String())
		base += len(chunk)
	}
	if len(remaining) != 0 {
		return nil, errors.New("artifact input edit exceeds the job's steps")
	}
	return result, nil
}
