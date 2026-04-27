package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

// memorySanitizerLog is the logger for the memory sanitizer module.
var memorySanitizerLog = logger.New("workflow:memory_sanitizer")

// sanitizeMemoryScriptName is the filename of the runtime memory sanitization script.
// The script scans memory directories for prompt injection patterns per ASI-06.
const sanitizeMemoryScriptName = "sanitize_memory.sh"

// generateRepoMemorySanitizationStep emits a workflow step that scans the
// repo-memory directory for prompt injection content after the clone step.
// This addresses OWASP Agentic Top 10 ASI-06 (Memory & Context Poisoning).
func generateRepoMemorySanitizationStep(builder *strings.Builder, memory RepoMemoryEntry, memoryDir string) {
	memorySanitizerLog.Printf("Generating repo-memory content scan step for memory id=%s dir=%s", memory.ID, memoryDir)

	if memory.Wiki {
		fmt.Fprintf(builder, "      - name: Scan wiki-memory for prompt injection (%s)\n", memory.ID)
	} else {
		fmt.Fprintf(builder, "      - name: Scan repo-memory for prompt injection (%s)\n", memory.ID)
	}
	builder.WriteString("        env:\n")
	fmt.Fprintf(builder, "          GH_AW_SCAN_DIR: %s\n", memoryDir)
	fmt.Fprintf(builder, "        run: bash \"${RUNNER_TEMP}/gh-aw/actions/%s\"\n", sanitizeMemoryScriptName)
}
