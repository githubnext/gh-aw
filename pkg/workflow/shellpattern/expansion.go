// Package shellpattern defines lexical shell patterns blocked by the
// safe-output harness. These are policy checks, not shell execution analysis.
package shellpattern

import (
	"regexp"
	"strings"
)

type Expansion struct {
	Start       int
	End         int
	Description string
}

var expansionPattern = regexp.MustCompile(
	`(?P<paramTransform>\$\{[A-Za-z_][A-Za-z0-9_]*@[^}]*\})` +
		`|(?P<indirectExpand>\$\{![A-Za-z_])` +
		`|(?P<commandSubst>\$\()` +
		"|(?P<backtick>`[^`\n]+`)",
)

var expansionDescriptions = map[string]string{
	"paramTransform": "parameter transformation (e.g. ${var@P})",
	"indirectExpand": "indirect expansion (e.g. ${!var})",
	"commandSubst":   "command substitution (e.g. $(command))",
	"backtick":       "backtick command substitution (e.g. `command`)",
}

// FirstUnsafeExpansion preserves the compiler's lexical harness policy,
// including matches in quoted text and comments. Arithmetic expansion is
// allowed; nested command substitutions inside it are still checked.
func FirstUnsafeExpansion(script string) (Expansion, bool) {
	if !strings.ContainsAny(script, "$`") {
		return Expansion{}, false
	}
	for _, match := range expansionPattern.FindAllStringSubmatchIndex(script, -1) {
		for group, name := range expansionPattern.SubexpNames() {
			if name == "" || match[group*2] < 0 { //nolint:uncheckedsliceindex // Regexp results contain two indices for every subexpression of this same pattern.
				continue
			}
			start, end := match[group*2], match[group*2+1] //nolint:uncheckedsliceindex // SubexpNames and match capture pairs come from the same compiled regexp.
			if name == "commandSubst" && strings.HasPrefix(script[start:], "$((") {
				continue
			}
			return Expansion{Start: start, End: end, Description: expansionDescriptions[name]}, true
		}
	}
	return Expansion{}, false
}
