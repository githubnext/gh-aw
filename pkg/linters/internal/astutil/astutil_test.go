//go:build !integration

package astutil

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func typecheckSnippet(t *testing.T, src string) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	cfg := types.Config{Importer: importer.Default()}
	if _, err := cfg.Check("example.com/p", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("type checking failed: %v", err)
	}

	return &analysis.Pass{TypesInfo: info}, file
}

// parseSnippet parses src and returns the fset, ast.File, and an analysis.Pass
// suitable for passing to AddImportEdit. It does not type-check.
func parseSnippet(t *testing.T, src string) (*token.FileSet, *ast.File, *analysis.Pass) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	pass := &analysis.Pass{Fset: fset}
	return fset, file, pass
}

// applyTextEdits applies edits to src and returns the resulting string.
// Edits must be non-overlapping. They are applied in reverse position order so
// that earlier offsets remain valid after each replacement.
func applyTextEdits(t *testing.T, fset *token.FileSet, src []byte, edits []analysis.TextEdit) string {
	t.Helper()
	// Sort edits by Pos descending so we apply from end to start.
	sorted := make([]analysis.TextEdit, len(edits))
	copy(sorted, edits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Pos > sorted[j].Pos
	})

	result := make([]byte, len(src))
	copy(result, src)

	tokFile := fset.File(sorted[0].Pos)
	if tokFile == nil && len(sorted) > 0 {
		t.Fatalf("applyTextEdits: no token.File for pos %d", sorted[0].Pos)
	}

	for _, e := range sorted {
		tf := fset.File(e.Pos)
		if tf == nil {
			t.Fatalf("applyTextEdits: no token.File for pos %d", e.Pos)
		}
		start := tf.Offset(e.Pos)
		// Handle the case where End == Pos (pure insertion).
		var end int
		if e.End == e.Pos {
			end = start
		} else {
			end = tf.Offset(e.End)
		}
		var replacement []byte
		if e.NewText != nil {
			replacement = e.NewText
		}
		result = append(result[:start], append(replacement, result[end:]...)...)
	}
	return string(result)
}

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

func TestConstIntValue(t *testing.T) {
	t.Parallel()

	intExpr := ast.NewIdent("n")
	strExpr := ast.NewIdent("s")
	unknown := ast.NewIdent("x")

	pass := &analysis.Pass{
		TypesInfo: &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{
				intExpr: {Value: constant.MakeInt64(42)},
				strExpr: {Value: constant.MakeString("hello")},
			},
		},
	}

	v, ok := ConstIntValue(pass, intExpr)
	if !ok || v != 42 {
		t.Fatalf("ConstIntValue(int) = (%d, %v), want (42, true)", v, ok)
	}

	if _, ok := ConstIntValue(pass, strExpr); ok {
		t.Fatal("ConstIntValue(string constant) = ok=true, want false")
	}

	if _, ok := ConstIntValue(pass, unknown); ok {
		t.Fatal("ConstIntValue(unknown expr) = ok=true, want false")
	}
}

func TestAsStringsMethodCall(t *testing.T) {
	t.Parallel()

	stringsPkg := types.NewPackage("strings", "strings")
	otherPkg := types.NewPackage("other", "other")
	stringsIdent := ast.NewIdent("strings")
	otherIdent := ast.NewIdent("other")

	pass := &analysis.Pass{
		TypesInfo: &types.Info{
			Uses: map[*ast.Ident]types.Object{
				stringsIdent: types.NewPkgName(token.NoPos, nil, "strings", stringsPkg),
				otherIdent:   types.NewPkgName(token.NoPos, nil, "other", otherPkg),
			},
		},
	}

	makeCall := func(pkgIdent *ast.Ident, method string) *ast.CallExpr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   pkgIdent,
				Sel: ast.NewIdent(method),
			},
		}
	}

	// strings.Index → matches "Index"
	indexCall := makeCall(stringsIdent, "Index")
	got, ok := AsStringsMethodCall(pass, indexCall, "Index")
	if !ok || got != indexCall {
		t.Fatalf("AsStringsMethodCall(strings.Index, Index) = (%v, %v), want (%v, true)", got, ok, indexCall)
	}

	// strings.Index does not match "Count"
	if _, ok := AsStringsMethodCall(pass, indexCall, "Count"); ok {
		t.Fatal("AsStringsMethodCall(strings.Index, Count) = ok=true, want false")
	}

	// other.Index does not match (wrong package)
	if _, ok := AsStringsMethodCall(pass, makeCall(otherIdent, "Index"), "Index"); ok {
		t.Fatal("AsStringsMethodCall(other.Index, Index) = ok=true, want false")
	}

	// non-call expression
	if _, ok := AsStringsMethodCall(pass, ast.NewIdent("x"), "Index"); ok {
		t.Fatal("AsStringsMethodCall(non-call, Index) = ok=true, want false")
	}
}

func TestCallQualifierText(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("strings"),
			Sel: ast.NewIdent("Index"),
		},
	}
	if got := CallQualifierText(fset, call); got != "strings" {
		t.Fatalf("CallQualifierText = %q, want %q", got, "strings")
	}

	// non-selector call (e.g. dot-import): returns ""
	directCall := &ast.CallExpr{Fun: ast.NewIdent("Index")}
	if got := CallQualifierText(fset, directCall); got != "" {
		t.Fatalf("CallQualifierText(non-selector) = %q, want %q", got, "")
	}
}

func TestBuildContainsFix(t *testing.T) {
	t.Parallel()

	expr := &ast.BinaryExpr{
		X:  ast.NewIdent("a"),
		Op: token.NEQ,
		Y:  ast.NewIdent("b"),
	}

	fixes := BuildContainsFix(expr, "strings", "s", "sub", false, "test message")
	if len(fixes) != 1 {
		t.Fatalf("got %d fixes, want 1", len(fixes))
	}
	if fixes[0].Message != "test message" {
		t.Fatalf("Message = %q, want %q", fixes[0].Message, "test message")
	}
	if got := string(fixes[0].TextEdits[0].NewText); got != "strings.Contains(s, sub)" {
		t.Fatalf("NewText = %q, want %q", got, "strings.Contains(s, sub)")
	}

	// negated
	fixes = BuildContainsFix(expr, "strings", "s", "sub", true, "negated message")
	if got := string(fixes[0].TextEdits[0].NewText); got != "!strings.Contains(s, sub)" {
		t.Fatalf("negated NewText = %q, want %q", got, "!strings.Contains(s, sub)")
	}
	if fixes[0].Message != "negated message" {
		t.Fatalf("negated Message = %q, want %q", fixes[0].Message, "negated message")
	}
}

func TestByteStringTypeHelpers(t *testing.T) {
	t.Parallel()

	const src = `package p

func g(s string) []byte { return nil }

func f(s string, b []byte) {
	type myString string
	var ms myString

	_ = []byte(s)
	_ = g(s)
	_ = b
	_ = s
	_ = ms
}
`

	pass, file := typecheckSnippet(t, src)

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		decl, ok := decl.(*ast.FuncDecl)
		if ok && decl.Name.Name == "f" {
			fn = decl
			break
		}
	}
	if fn == nil || fn.Body == nil {
		t.Fatal("failed to find function f in test snippet")
	}

	var rhsExprs []ast.Expr
	for _, stmt := range fn.Body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			continue
		}
		rhsExprs = append(rhsExprs, assign.Rhs[0])
	}
	if len(rhsExprs) != 5 {
		t.Fatalf("found %d assignment expressions in f, want 5", len(rhsExprs))
	}

	byteConv, ok := rhsExprs[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("rhs[0] type = %T, want *ast.CallExpr", rhsExprs[0])
	}
	gCall, ok := rhsExprs[1].(*ast.CallExpr)
	if !ok {
		t.Fatalf("rhs[1] type = %T, want *ast.CallExpr", rhsExprs[1])
	}
	bIdent, ok := rhsExprs[2].(*ast.Ident)
	if !ok {
		t.Fatalf("rhs[2] type = %T, want *ast.Ident", rhsExprs[2])
	}
	sIdent, ok := rhsExprs[3].(*ast.Ident)
	if !ok {
		t.Fatalf("rhs[3] type = %T, want *ast.Ident", rhsExprs[3])
	}
	msIdent, ok := rhsExprs[4].(*ast.Ident)
	if !ok {
		t.Fatalf("rhs[4] type = %T, want *ast.Ident", rhsExprs[4])
	}

	if !IsByteSlice(pass, bIdent) {
		t.Fatal("IsByteSlice(b) = false, want true")
	}
	if IsByteSlice(pass, sIdent) {
		t.Fatal("IsByteSlice(s) = true, want false")
	}

	if !IsByteSliceConversion(pass, byteConv) {
		t.Fatal("IsByteSliceConversion([]byte(s)) = false, want true")
	}
	if IsByteSliceConversion(pass, gCall) {
		t.Fatal("IsByteSliceConversion(g(s)) = true, want false")
	}

	if !IsStringType(pass, sIdent) {
		t.Fatal("IsStringType(s) = false, want true")
	}
	if !IsStringType(pass, msIdent) {
		t.Fatal("IsStringType(ms) = false, want true for named string")
	}
	if IsStringType(pass, bIdent) {
		t.Fatal("IsStringType(b) = true, want false")
	}
}

func TestFileForPos(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	src := `package p

func f() {}
`
	file, err := parser.ParseFile(fset, "snippet.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// pos inside the file should return the file
	got := FileForPos([]*ast.File{file}, file.Pos())
	if got != file {
		t.Fatalf("FileForPos(file.Pos()) = %v, want the file", got)
	}

	// pos at file end should return the file
	got = FileForPos([]*ast.File{file}, file.End())
	if got != file {
		t.Fatalf("FileForPos(file.End()) = %v, want the file", got)
	}

	// pos before the file should return nil
	got = FileForPos([]*ast.File{file}, file.Pos()-1)
	if got != nil {
		t.Fatalf("FileForPos(before) = %v, want nil", got)
	}

	// empty file list should return nil
	got = FileForPos(nil, file.Pos())
	if got != nil {
		t.Fatalf("FileForPos(nil files) = %v, want nil", got)
	}
}

func TestCountPkgUsesInFile(t *testing.T) {
	t.Parallel()

	const src = `package p

import "fmt"

func f() {
	fmt.Println("hello")
	fmt.Println("world")
}
`
	pass, file := typecheckSnippet(t, src)
	pass.Files = []*ast.File{file}

	count := CountPkgUsesInFile(pass, file, "fmt")
	if count != 2 {
		t.Fatalf("CountPkgUsesInFile(fmt) = %d, want 2", count)
	}

	count = CountPkgUsesInFile(pass, file, "strconv")
	if count != 0 {
		t.Fatalf("CountPkgUsesInFile(strconv) = %d, want 0", count)
	}
}

func TestImportSpecLineRange(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	src := `package p

import (
	"fmt"
	"os"
)
`
	file, err := parser.ParseFile(fset, "snippet.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(file.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(file.Imports))
	}
	spec := file.Imports[0] // "fmt"
	start, end := ImportSpecLineRange(fset, spec)
	if start >= end {
		t.Fatalf("ImportSpecLineRange start=%d >= end=%d, want start < end", start, end)
	}
	// The line range should include the spec itself.
	if spec.Pos() < start || spec.End() > end {
		t.Fatalf("ImportSpecLineRange does not contain spec: range [%d, %d), spec [%d, %d)", start, end, spec.Pos(), spec.End())
	}
}

func TestAddImportEdit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		pkg    string
		want   string
		wantOK bool
	}{
		{
			name: "append to grouped import block",
			src: `package p

import (
	"fmt"
)

func f() { fmt.Println() }
`,
			pkg:    "os",
			wantOK: true,
			want: `package p

import (
	"fmt"
	"os"
)

func f() { fmt.Println() }
`,
		},
		{
			name: "convert single ungrouped import to grouped block",
			src: `package p

import "fmt"

func f() { fmt.Println() }
`,
			pkg:    "os",
			wantOK: true,
			want: `package p

import (
	"fmt"
	"os"
)

func f() { fmt.Println() }
`,
		},
		{
			name: "no existing imports — insert standalone after package name",
			src: `package p

func f() {}
`,
			pkg:    "os",
			wantOK: true,
			want: `package p

import "os"

func f() {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, file, pass := parseSnippet(t, tt.src)
			edit, ok := AddImportEdit(pass, file, tt.pkg)
			if ok != tt.wantOK {
				t.Fatalf("AddImportEdit() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			got := applyTextEdits(t, fset, []byte(tt.src), []analysis.TextEdit{edit})
			if got != tt.want {
				t.Fatalf("AddImportEdit() result:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRemoveImportEdit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		pkg    string
		want   string
		wantOK bool
	}{
		{
			name: "remove spec from grouped multi-import block",
			src: `package p

import (
	"fmt"
	"os"
)

func f() {}
`,
			pkg:    "fmt",
			wantOK: true,
			want: `package p

import (
	"os"
)

func f() {}
`,
		},
		{
			name: "remove sole spec from grouped block removes entire decl",
			src: `package p

import (
	"fmt"
)

func f() {}
`,
			pkg:    "fmt",
			wantOK: true,
			want: `package p



func f() {}
`,
		},
		{
			name: "remove ungrouped single import removes entire decl",
			src: `package p

import "fmt"

func f() {}
`,
			pkg:    "fmt",
			wantOK: true,
			want: `package p



func f() {}
`,
		},
		{
			name: "import not present returns false",
			src: `package p

import "fmt"

func f() {}
`,
			pkg:    "os",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, file, _ := parseSnippet(t, tt.src)
			edit, ok := RemoveImportEdit(fset, file, tt.pkg)
			if ok != tt.wantOK {
				t.Fatalf("RemoveImportEdit() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			got := applyTextEdits(t, fset, []byte(tt.src), []analysis.TextEdit{edit})
			if got != tt.want {
				t.Fatalf("RemoveImportEdit() result:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestSwapImportEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		src       string
		addPkg    string
		removePkg string
		want      string
		wantNil   bool
	}{
		{
			name: "ungrouped single import replaced",
			src: `package p

import "fmt"

func f() {}
`,
			addPkg:    "os",
			removePkg: "fmt",
			want: `package p

import "os"

func f() {}
`,
		},
		{
			name: "grouped block with only removePkg replaced",
			src: `package p

import (
	"fmt"
)

func f() {}
`,
			addPkg:    "os",
			removePkg: "fmt",
			want: `package p

import "os"

func f() {}
`,
		},
		{
			name: "grouped block with multiple imports — delete then insert, edits sorted by position",
			src: `package p

import (
	"fmt"
	"os"
)

func f() {}
`,
			addPkg:    "strconv",
			removePkg: "fmt",
			want: `package p

import (
	"os"
	"strconv"
)

func f() {}
`,
		},
		{
			name: "removePkg not found returns nil",
			src: `package p

import "fmt"

func f() {}
`,
			addPkg:    "os",
			removePkg: "log",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, file, _ := parseSnippet(t, tt.src)
			edits := SwapImportEdits(fset, file, tt.addPkg, tt.removePkg)
			if tt.wantNil {
				if edits != nil {
					t.Fatalf("SwapImportEdits() = %v, want nil", edits)
				}
				return
			}
			if edits == nil {
				t.Fatal("SwapImportEdits() = nil, want edits")
			}
			// Verify edits are sorted by position (required by analysis framework).
			for i := 1; i < len(edits); i++ {
				if edits[i].Pos < edits[i-1].Pos {
					t.Fatalf("SwapImportEdits() edits not sorted by position: edits[%d].Pos=%d < edits[%d].Pos=%d",
						i, edits[i].Pos, i-1, edits[i-1].Pos)
				}
			}
			got := applyTextEdits(t, fset, []byte(tt.src), edits)
			if got != tt.want {
				t.Fatalf("SwapImportEdits() result:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
