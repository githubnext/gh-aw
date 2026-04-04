package agentdrain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var maskLog = logger.New("agentdrain:mask")

// Masker applies a sequence of regex substitution rules to normalize log lines.
type Masker struct {
	rules []compiledRule
}

type compiledRule struct {
	name        string
	re          *regexp.Regexp
	replacement string
}

// NewMasker compiles the given MaskRules into a Masker ready for use.
// Returns an error if any pattern fails to compile.
func NewMasker(rules []MaskRule) (*Masker, error) {
	maskLog.Printf("Compiling %d mask rules", len(rules))
	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			maskLog.Printf("Failed to compile mask rule %q: %v", r.Name, err)
			return nil, fmt.Errorf("agentdrain: mask rule %q: %w", r.Name, err)
		}
		compiled = append(compiled, compiledRule{
			name:        r.Name,
			re:          re,
			replacement: r.Replacement,
		})
	}
	maskLog.Printf("Masker ready with %d compiled rules", len(compiled))
	return &Masker{rules: compiled}, nil
}

// Mask applies all mask rules in order and returns the transformed line.
func (m *Masker) Mask(line string) string {
	for _, r := range m.rules {
		line = r.re.ReplaceAllString(line, r.replacement)
	}
	return line
}

// FlattenEvent converts an AgentEvent into a deterministic string suitable for
// template mining. Field keys are sorted alphabetically; fields listed in
// excludeFields are omitted. The result looks like:
//
//	stage=tool_call key1=val1 key2=val2
func FlattenEvent(evt AgentEvent, excludeFields []string) string {
	maskLog.Printf("Flattening event: stage=%s, fields=%d, exclude=%d", evt.Stage, len(evt.Fields), len(excludeFields))
	excluded := make(map[string]bool, len(excludeFields))
	for _, f := range excludeFields {
		excluded[f] = true
	}

	keys := make([]string, 0, len(evt.Fields))
	for k := range evt.Fields {
		if !excluded[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)+1)
	if evt.Stage != "" {
		parts = append(parts, "stage="+evt.Stage)
	}
	for _, k := range keys {
		parts = append(parts, k+"="+evt.Fields[k])
	}
	return strings.Join(parts, " ")
}

// Tokenize splits a log line on whitespace and returns the individual tokens.
func Tokenize(line string) []string {
	return strings.Fields(line)
}
