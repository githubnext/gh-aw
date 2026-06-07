package nolint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestBuildLineIndex_ParsesDirectiveTokens(t *testing.T) {
	const filename = "nolint_tokens.go"
	const src = `package p

//nolint:gosec,tolowerequalfold // second token should match
var _ = 1

//nolint:tolowerequalfold,gosec
var _ = 2

//nolint:tolowerequalfoldX
var _ = 3

//nolint:all
var _ = 4

//nolint:tolowerequalfold
var _ = 5
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	idx := BuildLineIndex(&analysis.Pass{Fset: fset, Files: []*ast.File{file}}, "tolowerequalfold")
	lines := idx[filename]

	for _, line := range []int{3, 6, 12, 15} {
		if _, ok := lines[line]; !ok {
			t.Fatalf("line %d missing from nolint index", line)
		}
	}
	if _, ok := lines[9]; ok {
		t.Fatalf("line 9 unexpectedly matched prefix-only directive")
	}
}

func TestHasDirective_SameLineAndPreviousLine(t *testing.T) {
	idx := map[string]map[int]struct{}{
		"test.go": {
			3: struct{}{},
		},
	}

	if !HasDirective(token.Position{Filename: "test.go", Line: 3}, idx) {
		t.Fatalf("expected same-line directive match")
	}
	if !HasDirective(token.Position{Filename: "test.go", Line: 4}, idx) {
		t.Fatalf("expected previous-line directive match")
	}
	if HasDirective(token.Position{Filename: "test.go", Line: 5}, idx) {
		t.Fatalf("unexpected directive match for unrelated line")
	}
}
