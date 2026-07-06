//go:build !integration

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

func TestEnclosingFuncType(t *testing.T) {
	t.Parallel()

	funcDecl := &ast.FuncDecl{Type: &ast.FuncType{}}
	if got := EnclosingFuncType(funcDecl); got != funcDecl.Type {
		t.Fatalf("EnclosingFuncType(FuncDecl) = %#v, want %#v", got, funcDecl.Type)
	}

	funcLit := &ast.FuncLit{Type: &ast.FuncType{}}
	if got := EnclosingFuncType(funcLit); got != funcLit.Type {
		t.Fatalf("EnclosingFuncType(FuncLit) = %#v, want %#v", got, funcLit.Type)
	}

	if got := EnclosingFuncType(ast.NewIdent("x")); got != nil {
		t.Fatalf("EnclosingFuncType(non-func) = %#v, want nil", got)
	}
}

func TestContextHelpers(t *testing.T) {
	t.Parallel()

	ctxPkg := types.NewPackage("context", "context")
	ctxIface := types.NewInterfaceType(nil, nil)
	ctxIface.Complete()
	ctxType := types.NewTypeName(token.NoPos, ctxPkg, "Context", ctxIface)
	ctxPkg.Scope().Insert(ctxType)

	makePassWithFuncType := func(includeContextImport bool, paramName string) (*analysis.Pass, *ast.FuncType) {
		pkg := types.NewPackage("example.com/p", "p")
		if includeContextImport {
			pkg.SetImports([]*types.Package{ctxPkg})
		}
		ctxIdent := ast.NewIdent("Context")
		fnType := &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{{
					Names: []*ast.Ident{ast.NewIdent(paramName)},
					Type:  ctxIdent,
				}},
			},
		}
		pass := &analysis.Pass{
			Pkg: pkg,
			TypesInfo: &types.Info{
				Types: map[ast.Expr]types.TypeAndValue{
					ctxIdent: {Type: ctxType.Type()},
				},
			},
		}
		return pass, fnType
	}

	passWithContext, fnTypeWithContext := makePassWithFuncType(true, "ctx")
	if got := ContextContextType(passWithContext); got == nil {
		t.Fatal("ContextContextType() = nil, want context.Context type")
	}
	name, ok := ContextParamName(passWithContext, fnTypeWithContext)
	if !ok || name != "ctx" {
		t.Fatalf("ContextParamName() = (%q, %v), want (%q, true)", name, ok, "ctx")
	}

	// blank identifier: a context param named "_" should not be found.
	passWithBlank, fnTypeWithBlank := makePassWithFuncType(true, "_")
	if _, ok := ContextParamName(passWithBlank, fnTypeWithBlank); ok {
		t.Fatal("ContextParamName() = ok=true for blank-identifier param, want false")
	}

	passWithoutContext, fnTypeWithoutContext := makePassWithFuncType(false, "ctx")
	if got := ContextContextType(passWithoutContext); got != nil {
		t.Fatalf("ContextContextType() = %#v, want nil without context import", got)
	}
	if _, ok := ContextParamName(passWithoutContext, fnTypeWithoutContext); ok {
		t.Fatal("ContextParamName() = ok=true, want false without context import")
	}
}

func TestCalledOSFunc(t *testing.T) {
	t.Parallel()

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	osPkg := types.NewPackage("os", "os")
	osFunc := types.NewFunc(token.NoPos, osPkg, "Getenv", sig)
	otherPkg := types.NewPackage("example.com/p", "p")
	otherFunc := types.NewFunc(token.NoPos, otherPkg, "Getenv", sig)

	selIdent := ast.NewIdent("Getenv")
	pass := &analysis.Pass{
		TypesInfo: &types.Info{
			Uses: map[*ast.Ident]types.Object{
				selIdent: osFunc,
			},
		},
	}
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("os"), Sel: selIdent}}

	if fn, ok := CalledOSFunc(pass, call, "Getenv", "LookupEnv"); !ok || fn != osFunc {
		t.Fatalf("CalledOSFunc() = (%#v, %v), want (%#v, true)", fn, ok, osFunc)
	}
	if _, ok := CalledOSFunc(pass, call, "Setenv"); ok {
		t.Fatal("CalledOSFunc() = ok=true for non-allowed name, want false")
	}

	pass.TypesInfo.Uses[selIdent] = otherFunc
	if _, ok := CalledOSFunc(pass, call); ok {
		t.Fatal("CalledOSFunc() = ok=true for non-os package, want false")
	}

	// direct *ast.Ident call (e.g. via dot-import): CalledOSFunc resolves Uses on the Ident.
	directIdent := ast.NewIdent("Getenv")
	pass.TypesInfo.Uses[directIdent] = osFunc
	directCall := &ast.CallExpr{Fun: directIdent}
	if fn, ok := CalledOSFunc(pass, directCall, "Getenv"); !ok || fn != osFunc {
		t.Fatalf("CalledOSFunc() direct ident = (%#v, %v), want (%#v, true)", fn, ok, osFunc)
	}
}

func TestFlipComparisonOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   token.Token
		want token.Token
	}{
		{name: "less", in: token.LSS, want: token.GTR},
		{name: "greater", in: token.GTR, want: token.LSS},
		{name: "leq", in: token.LEQ, want: token.GEQ},
		{name: "geq", in: token.GEQ, want: token.LEQ},
		{name: "equal unchanged", in: token.EQL, want: token.EQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FlipComparisonOp(tt.in); got != tt.want {
				t.Fatalf("FlipComparisonOp(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
