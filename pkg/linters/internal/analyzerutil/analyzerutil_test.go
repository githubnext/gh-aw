//go:build !integration

package analyzerutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

func TestNew(t *testing.T) {
	run := func(*analysis.Pass) (any, error) { return nil, nil }

	analyzer := New("example", "example documentation", run)

	if analyzer.Name != "example" || analyzer.Doc != "example documentation" {
		t.Errorf("New() metadata = (%q, %q), want (%q, %q)", analyzer.Name, analyzer.Doc, "example", "example documentation")
	}
	if analyzer.URL != repositoryURL+"example" {
		t.Errorf("New() URL = %q, want %q", analyzer.URL, repositoryURL+"example")
	}
	if len(analyzer.Requires) != 3 || analyzer.Requires[0] != inspect.Analyzer || analyzer.Requires[1] != nolint.Analyzer || analyzer.Requires[2] != filecheck.Analyzer {
		t.Errorf("New() Requires = %v, want standard dependencies", analyzer.Requires)
	}
}

func TestNewAtPath(t *testing.T) {
	analyzer := NewAtPath("example", "example documentation", "example-path", nil)

	if analyzer.URL != repositoryURL+"example-path" {
		t.Errorf("NewAtPath() URL = %q, want %q", analyzer.URL, repositoryURL+"example-path")
	}
}

func TestPreorder(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "example.go", `package example
func example() {
	first()
	second()
}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer: inspector.New([]*ast.File{file}),
		},
	}

	var names []string
	_, err = Preorder(pass, []ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		call := node.(*ast.CallExpr)
		names = append(names, call.Fun.(*ast.Ident).Name)
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"first", "second"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Preorder() visited %v, want %v", names, want)
	}
}

func TestIndexes(t *testing.T) {
	noLintIndex := nolint.DirectiveIndex{"example.go": {1: {"example": {}}}}
	generatedIndex := filecheck.GeneratedIndex{"generated.go": {}}
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]any{
			nolint.Analyzer:    noLintIndex,
			filecheck.Analyzer: generatedIndex,
		},
	}

	gotGenerated, gotNoLint, err := Indexes(pass)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotNoLint, noLintIndex) {
		t.Errorf("Indexes() nolint index = %v, want %v", gotNoLint, noLintIndex)
	}
	if !reflect.DeepEqual(gotGenerated, generatedIndex) {
		t.Errorf("Indexes() generated index = %v, want %v", gotGenerated, generatedIndex)
	}
}

func TestIndexesError(t *testing.T) {
	tests := map[string]map[*analysis.Analyzer]any{
		"missing nolint result":    {filecheck.Analyzer: filecheck.GeneratedIndex{}},
		"missing filecheck result": {nolint.Analyzer: nolint.DirectiveIndex{}},
	}

	for name, resultOf := range tests {
		t.Run(name, func(t *testing.T) {
			generatedFiles, noLintIndex, err := Indexes(&analysis.Pass{ResultOf: resultOf})
			if err == nil {
				t.Fatal("Indexes() error = nil, want error")
			}
			if generatedFiles != nil || noLintIndex != nil {
				t.Errorf("Indexes() = (%v, %v), want (nil, nil) on error", generatedFiles, noLintIndex)
			}
		})
	}
}

func TestPreorderIndexed(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "example.go", `package example
func example() {
	first()
	second()
}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	noLintIndex := nolint.DirectiveIndex{"example.go": {1: {"example": {}}}}
	generatedIndex := filecheck.GeneratedIndex{"generated.go": {}}
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer:   inspector.New([]*ast.File{file}),
			nolint.Analyzer:    noLintIndex,
			filecheck.Analyzer: generatedIndex,
		},
	}

	var names []string
	_, err = PreorderIndexed(pass, []ast.Node{(*ast.CallExpr)(nil)}, func(gotPass *analysis.Pass, node ast.Node, gotGenerated filecheck.GeneratedIndex, gotNoLint nolint.DirectiveIndex) {
		if gotPass != pass {
			t.Errorf("PreorderIndexed() passed %v, want %v", gotPass, pass)
		}
		if !reflect.DeepEqual(gotNoLint, noLintIndex) || !reflect.DeepEqual(gotGenerated, generatedIndex) {
			t.Errorf("PreorderIndexed() indexes = (%v, %v), want (%v, %v)", gotGenerated, gotNoLint, generatedIndex, noLintIndex)
		}
		names = append(names, node.(*ast.CallExpr).Fun.(*ast.Ident).Name)
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"first", "second"}; !reflect.DeepEqual(names, want) {
		t.Errorf("PreorderIndexed() visited %v, want %v", names, want)
	}
}

func TestPreorderIndexedError(t *testing.T) {
	visited := false
	_, err := PreorderIndexed(&analysis.Pass{ResultOf: map[*analysis.Analyzer]any{}}, []ast.Node{(*ast.CallExpr)(nil)},
		func(*analysis.Pass, ast.Node, filecheck.GeneratedIndex, nolint.DirectiveIndex) { visited = true })
	if err == nil {
		t.Fatal("PreorderIndexed() error = nil, want error")
	}
	if visited {
		t.Error("PreorderIndexed() visited nodes despite index error")
	}
}
