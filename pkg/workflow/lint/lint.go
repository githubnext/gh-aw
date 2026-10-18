// Package lint provides an extensible, in-process linter for Actions/AW ASTs.
// Rules inspect read-only ASTs and report structured, value-free diagnostics.
package lint

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
)

type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Input struct {
	Workflow *actions.Workflow
	// Document optionally supplies authored AW fields, including the prompt.
	// Workflow must be the compiled form, not a projection of raw frontmatter.
	Document *aw.Document
}

type Diagnostic struct {
	RuleID   string            `json:"rule_id"`
	Severity Severity          `json:"severity"`
	Path     string            `json:"path"`
	Line     int               `json:"script_line,omitempty"`
	Message  string            `json:"message"`
	Related  []RelatedLocation `json:"related,omitempty"`
}

type RelatedLocation struct {
	Path   string `json:"path"`
	Symbol string `json:"symbol"`
}

// Rule is independent of the taint engine: future structural and semantic
// checkers can share the registry, selection, diagnostics, and CLI integration.
// Check must not mutate Input or include sensitive values in diagnostics.
type Rule struct {
	ID          string
	Description string
	Severity    Severity
	Check       func(context.Context, Input) ([]Diagnostic, error)
}

type Runner struct {
	rules []Rule
}

var ruleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

func New(rules ...Rule) (*Runner, error) {
	seen := make(map[string]struct{})
	for _, rule := range rules {
		if !ruleIDPattern.MatchString(rule.ID) || rule.Description == "" || rule.Check == nil {
			return nil, errors.New("lint rule requires a kebab-case id, description, and check")
		}
		if rule.Severity != Error && rule.Severity != Warning {
			return nil, fmt.Errorf("lint rule %q requires error or warning severity", rule.ID)
		}
		if _, exists := seen[rule.ID]; exists {
			return nil, fmt.Errorf("duplicate lint rule %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
	}
	return &Runner{rules: slices.Clone(rules)}, nil
}

func (r *Runner) Rules() []Rule {
	return slices.Clone(r.rules)
}

// Select rejects unknown IDs rather than silently skipping a requested check.
// An empty selection runs every registered rule.
func (r *Runner) Select(ids ...string) (*Runner, error) {
	if len(ids) == 0 {
		return New(r.rules...)
	}
	var selected []Rule
	seen := make(map[string]struct{})
	for _, id := range ids {
		found := false
		for _, rule := range r.rules {
			if rule.ID != id {
				continue
			}
			found = true
			if _, exists := seen[id]; !exists {
				selected = append(selected, rule)
				seen[id] = struct{}{}
			}
			break
		}
		if !found {
			return nil, fmt.Errorf("unknown workflow lint rule %q", id)
		}
	}
	return New(selected...)
}

// Run returns diagnostics even when another rule fails. Tooling failures are
// errors, not successful empty reports. Rule IDs/severities are registry-owned.
func (r *Runner) Run(ctx context.Context, input Input) ([]Diagnostic, error) {
	if input.Workflow == nil {
		return nil, errors.New("workflow lint requires an Actions AST")
	}
	var diagnostics []Diagnostic
	var failures []error
	for _, rule := range r.rules {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		findings, err := rule.Check(ctx, input)
		if err != nil {
			failures = append(failures, fmt.Errorf("lint rule %s: %w", rule.ID, err))
		}
		for _, diagnostic := range findings {
			diagnostic.RuleID = rule.ID
			if diagnostic.Severity == "" {
				diagnostic.Severity = rule.Severity
			}
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	sortDiagnostics(diagnostics)
	diagnostics = slices.CompactFunc(diagnostics, func(a, b Diagnostic) bool {
		return a.RuleID == b.RuleID && a.Severity == b.Severity && a.Path == b.Path &&
			a.Line == b.Line && a.Message == b.Message && slices.Equal(a.Related, b.Related)
	})
	return diagnostics, errors.Join(failures...)
}

func sortDiagnostics(diagnostics []Diagnostic) {
	slices.SortFunc(diagnostics, func(a, b Diagnostic) int {
		for _, pair := range [][2]string{{a.Path, b.Path}, {a.RuleID, b.RuleID}} {
			if order := strings.Compare(pair[0], pair[1]); order != 0 {
				return order
			}
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		if order := strings.Compare(string(a.Severity), string(b.Severity)); order != 0 {
			return order
		}
		return strings.Compare(a.Message, b.Message)
	})
}

func Default() *Runner {
	return &Runner{rules: []Rule{
		SecretLeakRule(),
		ActionPinRule(),
		JobTimeoutRule(),
		TemplateInjectionRule(),
		WorkflowRunScopeRule(),
		PermissionScopeRule(),
		VersionCheckRule(),
		ConditionalImportRule(),
		PrivateToPublicRule(),
	}}
}
