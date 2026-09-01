package mapdeletecheck

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// buildPassForSameExpr type-checks the given source snippet and returns an
// *analysis.Pass with a populated TypesInfo, along with the parsed file so
// that expressions can be located for testing sameExpr in isolation.
func buildPassForSameExpr(t *testing.T, src string) (*analysis.Pass, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("src", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("failed to type-check source: %v", err)
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		TypesInfo: info,
	}
	return pass, file
}

// findFuncDecl locates a top-level function declaration by name.
func findFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// exprsInReturn extracts the list of expressions from the single return
// statement of the named function's body.
func exprsInReturn(file *ast.File, funcName string) []ast.Expr {
	fn := findFuncDecl(file, funcName)
	if fn == nil || fn.Body == nil {
		return nil
	}
	for _, stmt := range fn.Body.List {
		if ret, ok := stmt.(*ast.ReturnStmt); ok {
			return ret.Results
		}
	}
	return nil
}

func TestSameExpr(t *testing.T) {
	t.Parallel()
	const src = `package src

type S struct {
	Field int
}

func identSame() (int, int) {
	x := 1
	return x, x
}

func identDifferent() (int, int) {
	x := 1
	y := 2
	return x, y
}

func identVsOther() (int, string) {
	x := 1
	return x, "a"
}

func basicLitSame() (int, int) {
	return 42, 42
}

func basicLitDifferent() (int, int) {
	return 42, 43
}

func basicLitKindMismatch() (int, string) {
	return 1, "1"
}

func parenSame() (int, int) {
	x := 1
	return (x), (x)
}

func parenDifferent() (int, int) {
	x := 1
	y := 2
	return (x), (y)
}

func selectorSame() (int, int) {
	s := S{Field: 1}
	return s.Field, s.Field
}

func selectorDifferentField() (int, string) {
	return 1, "x"
}

func starSame() (int, int) {
	x := 1
	p := &x
	return *p, *p
}

func starDifferent() (int, int) {
	x := 1
	y := 2
	px := &x
	py := &y
	return *px, *py
}

func indexSame() (int, int) {
	m := map[string]int{"a": 1}
	k := "a"
	return m[k], m[k]
}

func indexDifferentKey() (int, int) {
	m := map[string]int{"a": 1, "b": 2}
	return m["a"], m["b"]
}

func indexDifferentMap() (int, int) {
	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"a": 1}
	k := "a"
	return m1[k], m2[k]
}

func unsupportedKind() (func(), func()) {
	return func() {}, func() {}
}
`
	pass, file := buildPassForSameExpr(t, src)

	tests := []struct {
		name string
		fn   string
		want bool
	}{
		{"ident same object", "identSame", true},
		{"ident different object", "identDifferent", false},
		{"ident vs basic lit", "identVsOther", false},
		{"basic lit same", "basicLitSame", true},
		{"basic lit different value", "basicLitDifferent", false},
		{"basic lit kind mismatch", "basicLitKindMismatch", false},
		{"paren same", "parenSame", true},
		{"paren different", "parenDifferent", false},
		{"selector same", "selectorSame", true},
		{"selector different field expr", "selectorDifferentField", false},
		{"star same", "starSame", true},
		{"star different", "starDifferent", false},
		{"index same map and key", "indexSame", true},
		{"index different key", "indexDifferentKey", false},
		{"index different map", "indexDifferentMap", false},
		{"unsupported expr kind falls to default", "unsupportedKind", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprs := exprsInReturn(file, tt.fn)
			if len(exprs) != 2 {
				t.Fatalf("expected 2 return expressions in %s, got %d", tt.fn, len(exprs))
			}
			got := sameExpr(pass, exprs[0], exprs[1])
			if got != tt.want {
				t.Errorf("sameExpr(%s) = %v, want %v", tt.fn, got, tt.want)
			}
		})
	}
}
