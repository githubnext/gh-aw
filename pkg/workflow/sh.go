package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed sh/pr_context_prompt.md
var prContextPromptText string

//go:embed sh/print_prompt_summary.sh
var printPromptSummaryScript string

//go:embed sh/create_prompt_first.sh
var createPromptFirstScript string

//go:embed sh/generate_git_patch.sh
var generateGitPatchScript string

//go:embed sh/create_cache_memory_dir.sh
var createCacheMemoryDirScript string

//go:embed sh/create_gh_aw_tmp_dir.sh
var createGhAwTmpDirScript string

//go:embed sh/xpia_prompt.md
var xpiaPromptText string

//go:embed sh/temp_folder_prompt.md
var tempFolderPromptText string

//go:embed sh/github_context_prompt.md
var githubContextPromptText string

//go:embed sh/playwright_prompt.md
var playwrightPromptText string

//go:embed sh/edit_tool_prompt.md
var editToolPromptText string

// WriteShellScriptToYAML writes a shell script with proper indentation to a strings.Builder
func WriteShellScriptToYAML(yaml *strings.Builder, script string, indent string) {
	scriptLines := strings.Split(script, "\n")
	for _, line := range scriptLines {
		// Skip empty lines at the beginning or end
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(yaml, "%s%s\n", indent, line)
		}
	}
}

// WritePromptTextToYAML writes prompt text to a YAML heredoc with proper indentation.
// It chunks the text into groups of lines of less than MaxPromptChunkSize characters, with a maximum of MaxPromptChunks chunks.
// Each chunk is written as a separate heredoc to avoid GitHub Actions step size limits (21KB).
func WritePromptTextToYAML(yaml *strings.Builder, text string, indent string) {
	textLines := strings.Split(text, "\n")
	chunks := chunkLines(textLines, indent, MaxPromptChunkSize, MaxPromptChunks)

	// Write each chunk as a separate heredoc
	for _, chunk := range chunks {
		yaml.WriteString(indent + "cat >> $GH_AW_PROMPT << 'PROMPT_EOF'\n")
		for _, line := range chunk {
			fmt.Fprintf(yaml, "%s%s\n", indent, line)
		}
		yaml.WriteString(indent + "PROMPT_EOF\n")
	}
}

// chunkLines splits lines into chunks where each chunk's total size (including indent) is less than maxSize.
// Returns at most maxChunks chunks. If content exceeds the limit, it truncates at the last chunk.
func chunkLines(lines []string, indent string, maxSize int, maxChunks int) [][]string {
	if len(lines) == 0 {
		return [][]string{{}}
	}

	var chunks [][]string
	var currentChunk []string
	currentSize := 0

	for _, line := range lines {
		// Calculate size including indent and newline
		lineSize := len(indent) + len(line) + 1

		// If adding this line would exceed the limit, start a new chunk
		if currentSize+lineSize > maxSize && len(currentChunk) > 0 {
			// Check if we've reached the maximum number of chunks
			if len(chunks) >= maxChunks-1 {
				// We're at the last allowed chunk, so add remaining lines to current chunk
				currentChunk = append(currentChunk, line)
				currentSize += lineSize
				continue
			}

			// Start a new chunk
			chunks = append(chunks, currentChunk)
			currentChunk = []string{line}
			currentSize = lineSize
		} else {
			currentChunk = append(currentChunk, line)
			currentSize += lineSize
		}
	}

	// Add the last chunk if there's content
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	// If we still have no chunks, return an empty chunk
	if len(chunks) == 0 {
		return [][]string{{}}
	}

	return chunks
}
