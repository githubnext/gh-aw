package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var detectionLog = logger.New("workflow:detection")

// detectTextOutputUsage checks if the markdown content uses ${{ needs.activation.outputs.text }}
func (c *Compiler) detectTextOutputUsage(markdownContent string) bool {
	// Check for the specific GitHub Actions expression
	hasUsage := strings.Contains(markdownContent, "${{ needs.activation.outputs.text }}")
	detectionLog.Printf("Detected usage of activation.outputs.text: %v", hasUsage)
	return hasUsage
}
