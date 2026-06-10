package astutil

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestRhsExprForIndex(t *testing.T) {
	t.Parallel()

	a := &ast.Ident{Name: "a"}
	b := &ast.Ident{Name: "b"}

	tests := []struct {
		name   string
		rhs    []ast.Expr
		idx    int
		want   ast.Expr
		wantOK bool
	}{
		{name: "empty", rhs: nil, idx: 0, want: nil, wantOK: false},
		{name: "single-first", rhs: []ast.Expr{a}, idx: 0, want: a, wantOK: true},
		{name: "single-nonzero-index", rhs: []ast.Expr{a}, idx: 1, want: nil, wantOK: false},
		{name: "multi-first", rhs: []ast.Expr{a, b}, idx: 0, want: a, wantOK: true},
		{name: "multi-second", rhs: []ast.Expr{a, b}, idx: 1, want: b, wantOK: true},
		{name: "multi-out-of-range", rhs: []ast.Expr{a, b}, idx: 2, want: nil, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := RhsExprForIndex(tt.rhs, tt.idx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestIsStringLiteral(t *testing.T) {
	t.Parallel()

	if !IsStringLiteral(&ast.BasicLit{Kind: token.STRING, Value: `"s"`}) {
		t.Fatal("expected string literal to be detected")
	}
	if IsStringLiteral(&ast.BasicLit{Kind: token.INT, Value: "1"}) {
		t.Fatal("did not expect int literal to be detected as string")
	}
}

func TestNodeText(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	node := &ast.Ident{Name: "myVar"}
	got := NodeText(fset, node)
	if got != "myVar" {
		t.Fatalf("NodeText = %q, want %q", got, "myVar")
	}
}
