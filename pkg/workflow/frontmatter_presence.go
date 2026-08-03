package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/parser"
)

// hasMeaningfulFrontmatter reports whether a parsed frontmatter block should be
// treated as present. Parsed fields count as present. For empty parsed maps,
// comment-only frontmatter is still present while whitespace-only blocks are not.
func hasMeaningfulFrontmatter(result *parser.FrontmatterResult) bool {
	if result == nil {
		return false
	}
	if len(result.Frontmatter) > 0 {
		return true
	}
	for _, line := range result.FrontmatterLines {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}
