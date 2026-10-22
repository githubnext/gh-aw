package lint

import (
	"context"
	"runtime/trace"
	"slices"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var performanceLog = logger.New("workflow:lint:perf")

func checkRulePerformance(ctx context.Context, rule Rule, input Input) ([]Diagnostic, error) {
	if !performanceLog.Enabled() && !trace.IsEnabled() {
		return rule.Check(ctx, input)
	}
	region := trace.StartRegion(ctx, "workflow:lint:rule:"+rule.ID)
	defer region.End()
	performanceLog.Printf("rule_start rule_id=%s actions=%t aw=%t", rule.ID, input.Workflow != nil, input.Document != nil)
	started := time.Now()
	findings, err := rule.Check(ctx, input)
	performanceLog.Printf("rule_complete rule_id=%s duration_ns=%d diagnostics=%d failed=%t",
		rule.ID, time.Since(started).Nanoseconds(), len(findings), err != nil)
	return findings, err
}

func normalizeDiagnosticsPerformance(ctx context.Context, diagnostics []Diagnostic) []Diagnostic {
	region := trace.StartRegion(ctx, "workflow:lint:diagnostics")
	defer region.End()
	var started time.Time
	if performanceLog.Enabled() {
		started = time.Now()
	}
	before := len(diagnostics)
	sortDiagnostics(diagnostics)
	diagnostics = slices.CompactFunc(diagnostics, func(a, b Diagnostic) bool {
		return a.RuleID == b.RuleID && a.Severity == b.Severity && a.Path == b.Path &&
			a.Line == b.Line && a.Message == b.Message && slices.Equal(a.Related, b.Related)
	})
	if performanceLog.Enabled() {
		performanceLog.Printf("diagnostics_complete duration_ns=%d before=%d after=%d",
			time.Since(started).Nanoseconds(), before, len(diagnostics))
	}
	return diagnostics
}
