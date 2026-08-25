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
	t.Parallel()
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
	t.Parallel()
	analyzer := NewAtPath("example", "example documentation", "example-path", nil)

	if analyzer.URL != repositoryURL+"example-path" {
		t.Errorf("NewAtPath() URL = %q, want %q", analyzer.URL, repositoryURL+"example-path")
	}
}

func TestPreorder(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	noLintIndex := nolint.DirectiveIndex{"example.go": {1: {"example": {}}}}
	generatedFiles := filecheck.GeneratedIndex{"generated.go": {}}

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]any{
			nolint.Analyzer:    noLintIndex,
			filecheck.Analyzer: generatedFiles,
		},
	}

	gotNoLint, gotGenerated, err := Indexes(pass)
	if err != nil {
		t.Fatalf("Indexes() error = %v", err)
	}
	if !reflect.DeepEqual(gotNoLint, noLintIndex) {
		t.Errorf("Indexes() nolint index = %v, want %v", gotNoLint, noLintIndex)
	}
	if !reflect.DeepEqual(gotGenerated, generatedFiles) {
		t.Errorf("Indexes() generated index = %v, want %v", gotGenerated, generatedFiles)
	}
}

func TestIndexesError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		resultOf map[*analysis.Analyzer]any
	}{
		{
			name: "malformed nolint result",
			resultOf: map[*analysis.Analyzer]any{
				nolint.Analyzer:    "not an index",
				filecheck.Analyzer: filecheck.GeneratedIndex{},
			},
		},
		{
			name: "malformed filecheck result",
			resultOf: map[*analysis.Analyzer]any{
				nolint.Analyzer:    nolint.DirectiveIndex{},
				filecheck.Analyzer: "not an index",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := &analysis.Pass{ResultOf: tt.resultOf}
			if _, _, err := Indexes(pass); err == nil {
				t.Error("Indexes() error = nil, want error")
			}
		})
	}
}
