package astutil

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
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

func TestIsPkgSelector(t *testing.T) {
	t.Parallel()

	makePass := func(ident *ast.Ident, obj types.Object) *analysis.Pass {
		return &analysis.Pass{
			TypesInfo: &types.Info{
				Uses: map[*ast.Ident]types.Object{
					ident: obj,
				},
			},
		}
	}

	logIdent := ast.NewIdent("log")
	aliasIdent := ast.NewIdent("applog")
	localIdent := ast.NewIdent("log")

	logPkg := types.NewPackage("log", "log")
	customType := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "customLogger", nil),
		types.NewStruct(nil, nil),
		nil,
	)

	tests := []struct {
		name    string
		pass    *analysis.Pass
		sel     *ast.SelectorExpr
		pkgPath string
		want    bool
	}{
		{
			name: "direct import name",
			pass: makePass(logIdent, types.NewPkgName(token.NoPos, nil, "log", logPkg)),
			sel: &ast.SelectorExpr{
				X:   logIdent,
				Sel: ast.NewIdent("Printf"),
			},
			pkgPath: "log",
			want:    true,
		},
		{
			name: "aliased import name",
			pass: makePass(aliasIdent, types.NewPkgName(token.NoPos, nil, "applog", logPkg)),
			sel: &ast.SelectorExpr{
				X:   aliasIdent,
				Sel: ast.NewIdent("Fatal"),
			},
			pkgPath: "log",
			want:    true,
		},
		{
			name: "local shadowed identifier",
			pass: makePass(localIdent, types.NewVar(token.NoPos, nil, "log", types.NewPointer(customType))),
			sel: &ast.SelectorExpr{
				X:   localIdent,
				Sel: ast.NewIdent("Printf"),
			},
			pkgPath: "log",
			want:    false,
		},
		{
			name: "nil pass",
			pass: nil,
			sel: &ast.SelectorExpr{
				X:   logIdent,
				Sel: ast.NewIdent("Printf"),
			},
			pkgPath: "log",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsPkgSelector(tt.pass, tt.sel, tt.pkgPath)
			if got != tt.want {
				t.Fatalf("IsPkgSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
