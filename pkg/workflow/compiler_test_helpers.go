package workflow

import "strings"

func containsInNonCommentLines(content, search string) bool {
	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(line, search) {
			return true
		}
	}
	return false
}

// indexInNonCommentLines returns the index (relative to the original content) of the first
// occurrence of search that appears in a non-comment line. This is used for order comparisons
// where we need to verify step ordering while ignoring matches in comment lines (such as
// frontmatter embedded as comments). Returns -1 if not found.
func indexInNonCommentLines(content, search string) int {
	lines := strings.Split(content, "\n")
	offset := 0
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			offset += len(line) + 1 // +1 for newline
			continue
		}
		if idx := strings.Index(line, search); idx != -1 {
			return offset + idx
		}
		offset += len(line) + 1 // +1 for newline
	}
	return -1
}

func extractJobSection(yamlContent, jobName string) string {
	lines := strings.Split(yamlContent, "\n")
	var jobLines []string
	inJob := false
	jobPrefix := "  " + jobName + ":"

	for i, line := range lines {
		if strings.HasPrefix(line, jobPrefix) {
			inJob = true
			jobLines = append(jobLines, line)
			continue
		}

		if inJob {
			// If we hit another job at the same level (starts with "  " and ends with ":" or ": #"),
			// stop. Job lines now may have inline comments (e.g. "  job: # annotation").
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
				// Extract the part before any comment to check for job name
				lineWithoutComment, _, _ := strings.Cut(line, " #")
				if strings.HasSuffix(strings.TrimSpace(lineWithoutComment), ":") {
					break
				}
			}
			// If we hit the end of jobs section, stop
			if strings.HasPrefix(line, "jobs:") && i > 0 {
				break
			}
			jobLines = append(jobLines, line)
		}
	}

	return strings.Join(jobLines, "\n")
}
