// Package analyzertest provides regression-test helpers for custom linters.
package analyzertest

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// RunTogether checks independent diagnostic-only analyzers against one fixture.
// Diagnostic messages are prefixed with the originating analyzer's name so want
// comments can distinguish overlapping reports. The analysis test driver runs
// their dependencies; each analyzer receives only its own dependency results.
// Analyzers that produce results or facts, or depend on another selected
// analyzer, are unsupported.
func RunTogether(t *testing.T, testdata string, analyzers []*analysis.Analyzer, patterns ...string) []*analysistest.Result {
	t.Helper()
	combined, err := together(analyzers)
	if err != nil {
		t.Fatal(err)
	}
	return analysistest.Run(t, testdata, combined, patterns...)
}

func together(analyzers []*analysis.Analyzer) (*analysis.Analyzer, error) {
	if len(analyzers) == 0 {
		return nil, errors.New("RunTogether requires at least one analyzer")
	}
	if err := analysis.Validate(analyzers); err != nil {
		return nil, fmt.Errorf("RunTogether: %w", err)
	}
	selected := make(map[*analysis.Analyzer]bool, len(analyzers))
	for _, analyzer := range analyzers {
		if selected[analyzer] {
			return nil, fmt.Errorf("RunTogether: duplicate analyzer %q", analyzer.Name)
		}
		if analyzer.ResultType != nil || len(analyzer.FactTypes) != 0 {
			return nil, fmt.Errorf("RunTogether: analyzer %q must not produce results or facts", analyzer.Name)
		}
		selected[analyzer] = true
	}
	visited := make(map[*analysis.Analyzer]bool)
	var checkDependency func(*analysis.Analyzer) error
	checkDependency = func(dependency *analysis.Analyzer) error {
		if selected[dependency] {
			return fmt.Errorf("RunTogether: selected analyzer %q is also a dependency", dependency.Name)
		}
		if visited[dependency] {
			return nil
		}
		visited[dependency] = true
		for _, required := range dependency.Requires {
			if err := checkDependency(required); err != nil {
				return err
			}
		}
		return nil
	}
	combined := &analysis.Analyzer{
		Name: "together",
		Doc:  "checks complementary diagnostic analyzers in regression fixtures",
		Run: func(pass *analysis.Pass) (any, error) {
			return runTogether(pass, analyzers)
		},
	}
	direct := make(map[*analysis.Analyzer]bool)
	for _, analyzer := range analyzers {
		for _, required := range analyzer.Requires {
			if err := checkDependency(required); err != nil {
				return nil, err
			}
			if !direct[required] {
				direct[required] = true
				combined.Requires = append(combined.Requires, required)
			}
		}
	}
	return combined, nil
}

func runTogether(pass *analysis.Pass, analyzers []*analysis.Analyzer) (any, error) {
	for _, analyzer := range analyzers {
		analyzerPass := *pass
		analyzerPass.Analyzer = analyzer
		analyzerPass.ResultOf = make(map[*analysis.Analyzer]any, len(analyzer.Requires))
		for _, required := range analyzer.Requires {
			analyzerPass.ResultOf[required] = pass.ResultOf[required]
		}
		analyzerPass.Report = func(diagnostic analysis.Diagnostic) {
			diagnostic.Message = analyzer.Name + ": " + diagnostic.Message
			pass.Report(diagnostic)
		}
		result, err := analyzer.Run(&analyzerPass)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", analyzer.Name, err)
		}
		if result != nil {
			return nil, fmt.Errorf("RunTogether: analyzer %q returned an unexpected result", analyzer.Name)
		}
	}
	return nil, nil
}
