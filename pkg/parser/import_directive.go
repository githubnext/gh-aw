package parser

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var importDirectiveLog = logger.New("parser:import_directive")

// IncludeDirectivePattern matches @include, @import (deprecated), or {{#import (new) directives
// The colon after #import is optional and ignored if present
var IncludeDirectivePattern = regexp.MustCompile(`^(?:@(?:include|import)(\?)?\s+(.+)|{{#import(\?)?\s*:?\s*(.+?)\s*}})$`)

// LegacyIncludeDirectivePattern matches only the deprecated @include and @import directives
var LegacyIncludeDirectivePattern = regexp.MustCompile(`^@(?:include|import)(\?)?\s+(.+)$`)

// ImportDirectiveMatch holds the parsed components of an import directive
type ImportDirectiveMatch struct {
	IsOptional bool
	Path       string
	IsLegacy   bool
	Original   string
}

// ParseImportDirective parses an import directive and returns its components
func ParseImportDirective(line string) *ImportDirectiveMatch {
	trimmedLine := strings.TrimSpace(line)

	// Fast-path: import directives must start with '@' or '{'; skip the regex for all other lines.
	if len(trimmedLine) == 0 || (trimmedLine[0] != '@' && trimmedLine[0] != '{') {
		return nil
	}

	// Check if it matches the import pattern at all
	matches := IncludeDirectivePattern.FindStringSubmatch(trimmedLine)
	if matches == nil {
		return nil
	}

	// Determine legacy vs new syntax from the captured groups of the first match.
	// Group 2 (path for @include/@import) is non-empty iff the legacy alternative matched.
	isLegacy := matches[2] != ""
	importDirectiveLog.Printf("Parsing import directive: legacy=%t, line=%s", isLegacy, trimmedLine)

	var isOptional bool
	var path string

	if isLegacy {
		// Legacy syntax: @include? path or @import? path
		// Group 1: optional marker, Group 2: path
		isOptional = matches[1] == "?"
		path = strings.TrimSpace(matches[2])
	} else {
		// New syntax: {{#import?: path}} or {{#import: path}} (colon is optional)
		// Group 3: optional marker, Group 4: path
		isOptional = matches[3] == "?"
		path = strings.TrimSpace(matches[4])
	}

	match := &ImportDirectiveMatch{
		IsOptional: isOptional,
		Path:       path,
		IsLegacy:   isLegacy,
		Original:   trimmedLine,
	}
	importDirectiveLog.Printf("Parsed import directive: path=%s, optional=%t, legacy=%t", path, isOptional, isLegacy)
	return match
}
