package campaign

import (
	"fmt"
	"strings"
)

// AppendPromptSection appends a section with a header and body to the builder.
// This is used to structure campaign orchestrator prompts with clear section boundaries.
func AppendPromptSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}

	// Titles should be single-line to keep markdown structure stable.
	title = strings.Join(strings.Fields(title), " ")
	fmt.Fprintf(b, "\n---\n# %s\n---\n%s", title, body)
}
