//go:build !integration

package analyzertest

import (
	"errors"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func diagnosticAnalyzer(name string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: name,
		Doc:  "test analyzer",
		Run:  func(*analysis.Pass) (any, error) { return nil, nil },
	}
}

func TestRunTogether(t *testing.T) {
	t.Parallel()
	dependencyRuns := 0
	dependency := diagnosticAnalyzer("dependency")
	dependency.ResultType = reflect.TypeFor[string]()
	dependency.Run = func(*analysis.Pass) (any, error) {
		dependencyRuns++
		return "dependency result", nil
	}
	first := diagnosticAnalyzer("first")
	first.Requires = []*analysis.Analyzer{dependency}
	second := diagnosticAnalyzer("second")
	for _, analyzer := range []*analysis.Analyzer{first, second} {
		analyzer.Run = func(pass *analysis.Pass) (any, error) {
			if pass.Analyzer != analyzer {
				t.Errorf("pass.Analyzer = %v, want %v", pass.Analyzer, analyzer)
			}
			if len(pass.ResultOf) != len(analyzer.Requires) {
				t.Errorf("%s received another analyzer's dependency results", analyzer.Name)
			}
			for _, required := range analyzer.Requires {
				if pass.ResultOf[required] != "dependency result" {
					t.Errorf("dependency result = %v, want dependency result", pass.ResultOf[required])
				}
			}
			pass.Reportf(pass.Files[0].Name.Pos(), "report")
			return nil, nil
		}
	}
	results := RunTogether(t, analysistest.TestData(), []*analysis.Analyzer{first, second}, "together")
	if dependencyRuns != 1 {
		t.Errorf("dependency ran %d times, want 1", dependencyRuns)
	}
	if len(results) != 1 || len(results[0].Diagnostics) != 2 {
		t.Errorf("results = %v, want one package with two diagnostics", results)
	}
}

func TestTogetherPreservesDiagnostics(t *testing.T) {
	t.Parallel()
	analyzer := diagnosticAnalyzer("first")
	want := analysis.Diagnostic{
		Pos: token.Pos(1), End: token.Pos(2), Category: "category", Message: "first: report",
		SuggestedFixes: []analysis.SuggestedFix{{Message: "fix"}},
		Related:        []analysis.RelatedInformation{{Pos: token.Pos(3), Message: "related"}},
	}
	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		diagnostic := want
		diagnostic.Message = "report"
		pass.Report(diagnostic)
		return nil, nil
	}
	combined, err := together([]*analysis.Analyzer{analyzer})
	if err != nil {
		t.Fatal(err)
	}
	var got []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer: combined,
		Report:   func(diagnostic analysis.Diagnostic) { got = append(got, diagnostic) },
	}
	if _, err := combined.Run(pass); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []analysis.Diagnostic{want}) {
		t.Errorf("diagnostics = %#v, want %#v", got, []analysis.Diagnostic{want})
	}
	if pass.Analyzer != combined || pass.ResultOf != nil {
		t.Error("combined pass was mutated")
	}
}

type testFact struct{}

func (*testFact) AFact() {}

func TestTogetherRejectsUnsupportedAnalyzers(t *testing.T) {
	t.Parallel()
	result := diagnosticAnalyzer("result")
	result.ResultType = reflect.TypeFor[string]()
	facts := diagnosticAnalyzer("facts")
	facts.FactTypes = []analysis.Fact{&testFact{}}
	leaf := diagnosticAnalyzer("leaf")
	dependent := diagnosticAnalyzer("dependent")
	dependent.Requires = []*analysis.Analyzer{leaf}
	transitive := diagnosticAnalyzer("transitive")
	transitive.Requires = []*analysis.Analyzer{dependent}
	tests := []struct {
		name      string
		analyzers []*analysis.Analyzer
		want      string
	}{
		{"empty", nil, "at least one"},
		{"nil", []*analysis.Analyzer{nil}, "nil"},
		{"duplicate", []*analysis.Analyzer{leaf, leaf}, "duplicate"},
		{"results", []*analysis.Analyzer{result}, "results or facts"},
		{"facts", []*analysis.Analyzer{facts}, "results or facts"},
		{"selected dependency", []*analysis.Analyzer{dependent, leaf}, "also a dependency"},
		{"transitive selected dependency", []*analysis.Analyzer{transitive, leaf}, "also a dependency"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := together(tt.analyzers); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("together() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestTogetherRunErrors(t *testing.T) {
	t.Parallel()
	want := errors.New("analyzer failure")
	for _, unexpectedResult := range []bool{false, true} {
		analyzer := diagnosticAnalyzer("first")
		analyzer.Run = func(*analysis.Pass) (any, error) {
			if unexpectedResult {
				return "undeclared result", nil
			}
			return nil, want
		}
		combined, err := together([]*analysis.Analyzer{analyzer})
		if err != nil {
			t.Fatal(err)
		}
		_, err = combined.Run(&analysis.Pass{})
		if unexpectedResult {
			if err == nil || !strings.Contains(err.Error(), "unexpected result") {
				t.Errorf("Run() error = %v, want unexpected result", err)
			}
		} else if !errors.Is(err, want) {
			t.Errorf("Run() error = %v, want %v", err, want)
		}
	}
}
