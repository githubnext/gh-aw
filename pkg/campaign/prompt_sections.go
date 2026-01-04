package campaign

import (
	"fmt"
	"strings"
)

func appendPromptSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}

	// Titles should be single-line to keep markdown structure stable.
	title = strings.Join(strings.Fields(title), " ")
	fmt.Fprintf(b, "\n---\n# %s\n---\n%s", title, body)
}
