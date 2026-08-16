// Command purity_scan is the PureLock precompute analyzer.
//
// It performs all the deterministic, expensive work for the PureLock agentic
// workflow so the agent itself only has to write tests:
//
//  1. Type-checks the requested packages with golang.org/x/tools/go/packages.
//  2. Classifies every top-level function as pure or impure using a
//     type-aware, fixed-point side-effect analysis.
//  3. Correlates each pure function with per-function coverage produced by
//     `go tool cover -func`.
//  4. Scores candidates by uncovered branch weight so the agent works on the
//     functions where a small test suite buys the most coverage.
//
// It is intentionally located under .github/ so it is invisible to the main
// module's package patterns (`go build ./...`). Run it with:
//
//	go run .github/scripts/purelock/purity_scan.go \
//	  -cover /tmp/purelock/func-coverage.txt \
//	  -out /tmp/purelock/candidates.json ./pkg/...
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// pureStdlibPackages are standard library packages whose exported functions are
// free of observable side effects for our purposes (no I/O, no globals, no
// clock, no randomness). Calls into any other package make a function impure.
var pureStdlibPackages = map[string]bool{
	"errors":          true,
	"fmt":             true, // only Sprintf-style helpers survive the call filter below
	"math":            true,
	"math/big":        true,
	"math/bits":       true,
	"path":            true,
	"path/filepath":   true,
	"regexp":          true,
	"sort":            true,
	"strconv":         true,
	"strings":         true,
	"unicode":         true,
	"unicode/utf8":    true,
	"encoding/base64": true,
	"encoding/hex":    true,
	"encoding/json":   true,
	"net/url":         true,
	"slices":          true,
	"maps":            true,
	"cmp":             true,
	"time":            true, // only pure time helpers survive the call filter below
}

// impureFuncs are individually banned functions inside otherwise pure packages.
var impureFuncs = map[string]bool{
	"fmt.Print":    true,
	"fmt.Printf":   true,
	"fmt.Println":  true,
	"fmt.Fprint":   true,
	"fmt.Fprintf":  true,
	"fmt.Fprintln": true,
	"fmt.Scan":     true,
	"fmt.Scanf":    true,
	"fmt.Scanln":   true,
	"time.Now":     true,
	"time.Since":   true,
	"time.Until":   true,
	"time.Sleep":   true,
	"time.After":   true,
	"time.Tick":    true,
}

// Candidate is one analyzed pure function emitted to the agent.
type Candidate struct {
	Package      string   `json:"package"`
	File         string   `json:"file"`
	Line         int      `json:"line"`
	Name         string   `json:"name"`
	Receiver     string   `json:"receiver,omitempty"`
	Signature    string   `json:"signature"`
	Exported     bool     `json:"exported"`
	Complexity   int      `json:"complexity"`
	Statements   int      `json:"statements"`
	Coverage     float64  `json:"coverage_pct"`
	HasTestFile  bool     `json:"has_test_file"`
	FuzzFriendly bool     `json:"fuzz_friendly"`
	Score        float64  `json:"score"`
	PurityNotes  []string `json:"purity_notes"`
}

// Report is the JSON document handed to the agent.
type Report struct {
	Module            string      `json:"module"`
	Patterns          []string    `json:"patterns"`
	FunctionsAnalyzed int         `json:"functions_analyzed"`
	PureFunctions     int         `json:"pure_functions"`
	CoverageLoaded    bool        `json:"coverage_loaded"`
	Candidates        []Candidate `json:"candidates"`
}

func main() {
	coverPath := flag.String("cover", "", "path to `go tool cover -func` output (optional)")
	outPath := flag.String("out", "purelock-candidates.json", "path of the JSON report to write")
	summaryPath := flag.String("summary", "", "optional path of a compact markdown summary to write")
	limit := flag.Int("limit", 40, "maximum number of candidates to emit")
	maxCoverage := flag.Float64("max-coverage", 100, "skip functions already covered at or above this percentage")
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./pkg/..."}
	}

	coverage, coveredFiles, coverageLoaded, err := loadFuncCoverage(*coverPath)
	if err != nil {
		fatalf("reading coverage: %v", err)
	}

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedModule | packages.NeedDeps | packages.NeedImports,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		fatalf("loading packages: %v", err)
	}

	analyzer := newAnalyzer(pkgs)
	analyzer.run()

	moduleName := ""
	for _, pkg := range pkgs {
		if pkg.Module != nil && pkg.Module.Path != "" {
			moduleName = pkg.Module.Path
			break
		}
	}

	report := Report{
		Module:            moduleName,
		Patterns:          patterns,
		FunctionsAnalyzed: len(analyzer.funcs),
		CoverageLoaded:    coverageLoaded,
	}

	for _, fn := range analyzer.funcs {
		if !analyzer.pure[fn.key] {
			continue
		}
		report.PureFunctions++

		cov, ok := coverage[fn.coverageKey()]
		if !ok {
			// The function's file was instrumented but the function itself has no
			// coverage record: the profile and the source tree disagree, so skip it
			// rather than report a coverage number that cannot be verified.
			if coveredFiles[fn.file] {
				continue
			}
			// The file was never instrumented (for example a package the coverage
			// run did not include). Treat it as uncovered; the agent measures its
			// own baseline before writing tests.
			cov = 0
		}
		if cov >= *maxCoverage {
			continue
		}

		candidate := Candidate{
			Package:      fn.pkgPath,
			File:         fn.file,
			Line:         fn.line,
			Name:         fn.name,
			Receiver:     fn.receiver,
			Signature:    fn.signature,
			Exported:     fn.exported,
			Complexity:   fn.complexity,
			Statements:   fn.statements,
			Coverage:     cov,
			HasTestFile:  hasTestFile(fn.file),
			FuzzFriendly: fn.fuzzFriendly,
			PurityNotes:  fn.notes,
		}
		// Prefer functions where a single well-designed table test closes the
		// largest number of uncovered branches: uncovered fraction x complexity.
		candidate.Score = (100 - cov) / 100 * float64(fn.complexity)
		if fn.exported {
			candidate.Score *= 1.25
		}
		report.Candidates = append(report.Candidates, candidate)
	}

	sort.Slice(report.Candidates, func(i, j int) bool {
		a, b := report.Candidates[i], report.Candidates[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	if *limit > 0 && len(report.Candidates) > *limit {
		report.Candidates = report.Candidates[:*limit]
	}

	if err := writeJSON(*outPath, report); err != nil {
		fatalf("writing report: %v", err)
	}
	if *summaryPath != "" {
		if err := writeSummary(*summaryPath, report); err != nil {
			fatalf("writing summary: %v", err)
		}
	}

	fmt.Printf("analyzed %d functions, %d pure, %d candidates written to %s\n",
		report.FunctionsAnalyzed, report.PureFunctions, len(report.Candidates), *outPath)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "purity_scan: "+format+"\n", args...)
	os.Exit(1)
}

// funcInfo is a single analyzed top-level function.
type funcInfo struct {
	key          string
	pkgPath      string
	file         string
	line         int
	name         string
	receiver     string
	signature    string
	exported     bool
	complexity   int
	statements   int
	fuzzFriendly bool
	notes        []string
	calls        []string
	localImpure  bool
}

// coverageKey matches the `file.go:line:` prefix used by `go tool cover -func`.
func (f *funcInfo) coverageKey() string {
	return f.file + ":" + strconv.Itoa(f.line) + ":" + f.name
}

type analyzer struct {
	pkgs  []*packages.Package
	funcs []*funcInfo
	byKey map[string]*funcInfo
	pure  map[string]bool
}

func newAnalyzer(pkgs []*packages.Package) *analyzer {
	return &analyzer{
		pkgs:  pkgs,
		byKey: map[string]*funcInfo{},
		pure:  map[string]bool{},
	}
}

func (a *analyzer) run() {
	for _, pkg := range a.pkgs {
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				info := a.inspect(pkg, fd)
				if info == nil {
					continue
				}
				a.funcs = append(a.funcs, info)
				a.byKey[info.key] = info
			}
		}
	}
	a.resolve()
}

// inspect performs the single-function part of the purity analysis.
func (a *analyzer) inspect(pkg *packages.Package, fd *ast.FuncDecl) *funcInfo {
	pos := pkg.Fset.Position(fd.Pos())
	if pos.Filename == "" || strings.HasSuffix(pos.Filename, "_test.go") {
		return nil
	}
	rel := relPath(pkg, pos.Filename)

	info := &funcInfo{
		key:        pkg.PkgPath + "." + receiverName(fd) + fd.Name.Name,
		pkgPath:    pkg.PkgPath,
		file:       rel,
		line:       pos.Line,
		name:       fd.Name.Name,
		receiver:   receiverName(fd),
		signature:  signatureOf(pkg, fd),
		exported:   fd.Name.IsExported(),
		complexity: 1,
	}

	if fd.Type.Results == nil || len(fd.Type.Results.List) == 0 {
		// A function returning nothing can only be useful through side effects.
		info.localImpure = true
		info.notes = append(info.notes, "no return values")
		return info
	}

	// A pointer receiver signals mutation of the receiver in most Go code.
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		if _, isPtr := fd.Recv.List[0].Type.(*ast.StarExpr); isPtr {
			info.localImpure = true
			info.notes = append(info.notes, "pointer receiver")
		}
	}

	info.fuzzFriendly = isFuzzFriendly(pkg, fd)

	locals := map[types.Object]bool{}
	collectLocals(pkg, fd, locals)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
			info.complexity++
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				info.complexity++
			}
		case *ast.GoStmt:
			info.localImpure = true
			info.notes = append(info.notes, "starts a goroutine")
		case *ast.SendStmt:
			info.localImpure = true
			info.notes = append(info.notes, "channel send")
		case *ast.SelectStmt:
			info.localImpure = true
			info.notes = append(info.notes, "select statement")
		case *ast.DeferStmt:
			info.localImpure = true
			info.notes = append(info.notes, "defer statement")
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				info.localImpure = true
				info.notes = append(info.notes, "channel receive")
			}
		case *ast.AssignStmt:
			info.statements++
			for _, lhs := range node.Lhs {
				if reason := escapingTarget(pkg, lhs, locals); reason != "" {
					info.localImpure = true
					info.notes = append(info.notes, reason)
				}
			}
		case *ast.IncDecStmt:
			info.statements++
			if reason := escapingTarget(pkg, node.X, locals); reason != "" {
				info.localImpure = true
				info.notes = append(info.notes, reason)
			}
		case *ast.CallExpr:
			info.statements++
			if target, reason := a.classifyCall(pkg, node, locals); reason != "" {
				info.localImpure = true
				info.notes = append(info.notes, reason)
			} else if target != "" {
				info.calls = append(info.calls, target)
			}
		case ast.Stmt:
			info.statements++
		}
		return true
	})

	info.notes = dedupe(info.notes)
	return info
}

// classifyCall returns either the key of an in-repo callee to resolve later, or
// a reason string explaining why the call makes the caller impure.
func (a *analyzer) classifyCall(pkg *packages.Package, call *ast.CallExpr, locals map[types.Object]bool) (string, string) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj := pkg.TypesInfo.Uses[fun]
		if obj == nil {
			obj = pkg.TypesInfo.Defs[fun]
		}
		if obj == nil {
			return "", ""
		}
		if _, isBuiltin := obj.(*types.Builtin); isBuiltin {
			switch fun.Name {
			case "append", "copy", "delete":
				// These write into the backing storage of their first argument,
				// which is only safe when that storage was created locally.
				if len(call.Args) > 0 && !isLocalValue(pkg, call.Args[0], locals) {
					return "", "builtin " + fun.Name + " may mutate shared storage"
				}
			}
			return "", ""
		}
		fn, ok := obj.(*types.Func)
		if !ok {
			// Calling a function value (parameter, field, variable) is opaque.
			return "", "call through function value " + fun.Name
		}
		return funcKey(fn), ""
	case *ast.SelectorExpr:
		sel := pkg.TypesInfo.Selections[fun]
		if sel == nil {
			// Package-qualified call such as strings.TrimSpace.
			if ident, ok := fun.X.(*ast.Ident); ok {
				if pkgName, ok := pkg.TypesInfo.Uses[ident].(*types.PkgName); ok {
					path := pkgName.Imported().Path()
					qualified := pkgName.Imported().Name() + "." + fun.Sel.Name
					if impureFuncs[qualified] {
						return "", "calls " + qualified
					}
					if a.isLocalPackage(path) {
						if obj := pkgName.Imported().Scope().Lookup(fun.Sel.Name); obj != nil {
							if fn, ok := obj.(*types.Func); ok {
								return funcKey(fn), ""
							}
						}
						return "", "unresolved call into " + path
					}
					if pureStdlibPackages[path] {
						return "", ""
					}
					return "", "calls external package " + path
				}
			}
			return "", "unresolved selector call " + fun.Sel.Name
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			return "", "method call through non-function selector"
		}
		if fn.Pkg() == nil {
			return "", ""
		}
		path := fn.Pkg().Path()
		if a.isLocalPackage(path) {
			if sel.Kind() == types.MethodVal && isPointerReceiver(fn) {
				return "", "calls pointer-receiver method " + fn.Name()
			}
			return funcKey(fn), ""
		}
		if pureStdlibPackages[path] {
			qualified := fn.Pkg().Name() + "." + fn.Name()
			if impureFuncs[qualified] {
				return "", "calls " + qualified
			}
			return "", ""
		}
		return "", "calls external package " + path
	}
	return "", ""
}

func (a *analyzer) isLocalPackage(path string) bool {
	for _, pkg := range a.pkgs {
		if pkg.PkgPath == path {
			return true
		}
	}
	return false
}

// resolve runs the fixed point: a function is pure when it has no local side
// effect and every in-repo function it calls is also pure.
func (a *analyzer) resolve() {
	for _, fn := range a.funcs {
		a.pure[fn.key] = !fn.localImpure
	}
	for changed := true; changed; {
		changed = false
		for _, fn := range a.funcs {
			if !a.pure[fn.key] {
				continue
			}
			for _, callee := range fn.calls {
				if callee == fn.key {
					continue // direct recursion is fine
				}
				calleeInfo, known := a.byKey[callee]
				if !known {
					// In-repo callee that was filtered out (for example a
					// generated or build-tagged file): stay conservative.
					a.pure[fn.key] = false
					fn.notes = append(fn.notes, "calls unanalyzed function")
					changed = true
					break
				}
				if !a.pure[calleeInfo.key] {
					a.pure[fn.key] = false
					fn.notes = append(fn.notes, "calls impure function "+calleeInfo.name)
					changed = true
					break
				}
			}
		}
	}
	for _, fn := range a.funcs {
		fn.notes = dedupe(fn.notes)
		if a.pure[fn.key] && len(fn.notes) == 0 {
			fn.notes = []string{"no observable side effects detected"}
		}
	}
}

// isLocalValue reports whether expr denotes storage created inside the function
// body. Composite literals, `make` calls, and locally declared variables qualify;
// parameters, globals, and fields of shared values do not.
func isLocalValue(pkg *packages.Package, expr ast.Expr, locals map[types.Object]bool) bool {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return true
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok && ident.Name == "make" {
			if _, isBuiltin := pkg.TypesInfo.Uses[ident].(*types.Builtin); isBuiltin {
				return true
			}
		}
		return false
	case *ast.Ident:
		if e.Name == "nil" {
			return true
		}
		obj := resolveObj(pkg, e)
		return obj != nil && locals[obj]
	}
	return false
}

func resolveObj(pkg *packages.Package, ident *ast.Ident) types.Object {
	obj := pkg.TypesInfo.Uses[ident]
	if obj == nil {
		obj = pkg.TypesInfo.Defs[ident]
	}
	return obj
}

// escapingTarget reports why writing to expr escapes the function, or "" when
// the write only touches a local variable.
func escapingTarget(pkg *packages.Package, expr ast.Expr, locals map[types.Object]bool) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "_" {
			return ""
		}
		obj := resolveObj(pkg, e)
		if obj == nil {
			return ""
		}
		if locals[obj] {
			return ""
		}
		return "assigns to non-local " + e.Name
	case *ast.StarExpr:
		return "writes through a pointer"
	case *ast.IndexExpr:
		return escapingWriteThroughBase(pkg, e.X, locals)
	case *ast.SelectorExpr:
		return escapingWriteThroughBase(pkg, e.X, locals)
	}
	return ""
}

// escapingWriteThroughBase treats index/field writes as escaping unless the base
// is a locally constructed value: maps and slices share backing storage with
// their caller.
func escapingWriteThroughBase(pkg *packages.Package, base ast.Expr, locals map[types.Object]bool) string {
	ident, ok := base.(*ast.Ident)
	if !ok {
		return "writes into a non-local composite value"
	}
	obj := resolveObj(pkg, ident)
	if obj == nil || !locals[obj] {
		return "writes into shared value " + ident.Name
	}
	return ""
}

// collectLocals records every object declared inside the function body so writes
// to them can be distinguished from writes to globals. Parameters are
// deliberately excluded: mutating a parameter's pointee or backing array is
// observable by the caller.
func collectLocals(pkg *packages.Package, fd *ast.FuncDecl, locals map[types.Object]bool) {
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			if node.Tok != token.DEFINE {
				return true
			}
			for _, lhs := range node.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if obj := pkg.TypesInfo.Defs[ident]; obj != nil {
						locals[obj] = true
					}
				}
			}
		case *ast.ValueSpec:
			for _, name := range node.Names {
				if obj := pkg.TypesInfo.Defs[name]; obj != nil {
					locals[obj] = true
				}
			}
		case *ast.RangeStmt:
			for _, expr := range []ast.Expr{node.Key, node.Value} {
				if ident, ok := expr.(*ast.Ident); ok {
					if obj := pkg.TypesInfo.Defs[ident]; obj != nil {
						locals[obj] = true
					}
				}
			}
		}
		return true
	})
}

// isFuzzFriendly reports whether every parameter is a type that Go's native
// fuzzing engine can generate directly.
func isFuzzFriendly(pkg *packages.Package, fd *ast.FuncDecl) bool {
	if fd.Recv != nil {
		return false
	}
	if fd.Type.Params == nil || len(fd.Type.Params.List) == 0 {
		return false
	}
	for _, field := range fd.Type.Params.List {
		tv, ok := pkg.TypesInfo.Types[field.Type]
		if !ok {
			return false
		}
		if basic, ok := tv.Type.Underlying().(*types.Basic); ok {
			switch basic.Kind() {
			case types.String, types.Bool, types.Int, types.Int8, types.Int16, types.Int32,
				types.Int64, types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
				types.Float32, types.Float64:
				continue
			}
			return false
		}
		if slice, ok := tv.Type.Underlying().(*types.Slice); ok {
			if elem, ok := slice.Elem().Underlying().(*types.Basic); ok && elem.Kind() == types.Uint8 {
				continue
			}
		}
		return false
	}
	return true
}

func isPointerReceiver(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	_, isPtr := sig.Recv().Type().(*types.Pointer)
	return isPtr
}

func funcKey(fn *types.Func) string {
	if fn.Pkg() == nil {
		return ""
	}
	recv := ""
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		recvType := sig.Recv().Type()
		if ptr, ok := recvType.(*types.Pointer); ok {
			recvType = ptr.Elem()
		}
		if named, ok := recvType.(*types.Named); ok {
			recv = named.Obj().Name() + "."
		}
	}
	return fn.Pkg().Path() + "." + recv + fn.Name()
}

func receiverName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	expr := fd.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if index, ok := expr.(*ast.IndexExpr); ok {
		expr = index.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name + "."
	}
	return ""
}

func signatureOf(pkg *packages.Package, fd *ast.FuncDecl) string {
	obj := pkg.TypesInfo.Defs[fd.Name]
	if obj == nil {
		return fd.Name.Name
	}
	return types.ObjectString(obj, types.RelativeTo(pkg.Types))
}

func relPath(pkg *packages.Package, filename string) string {
	if pkg.Module != nil && pkg.Module.Dir != "" {
		if rel, err := filepath.Rel(pkg.Module.Dir, filename); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, filename); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(filename)
}

func hasTestFile(file string) bool {
	candidate := strings.TrimSuffix(file, ".go") + "_test.go"
	_, err := os.Stat(candidate)
	return err == nil
}

// loadFuncCoverage parses `go tool cover -func` output into a map keyed by
// "file:line:FuncName", plus the set of files present in the profile.
func loadFuncCoverage(path string) (map[string]float64, map[string]bool, bool, error) {
	result := map[string]float64{}
	files := map[string]bool{}
	if path == "" {
		return result, files, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, files, false, nil
		}
		return nil, nil, false, err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		location, name, percent := fields[0], fields[1], fields[len(fields)-1]
		if !strings.HasSuffix(percent, "%") {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(percent, "%"), 64)
		if err != nil {
			continue
		}
		parts := strings.Split(location, ":")
		if len(parts) < 2 {
			continue
		}
		file := strings.TrimPrefix(filepath.ToSlash(parts[0]), "./")
		if idx := strings.Index(file, "gh-aw/"); idx >= 0 {
			file = file[idx+len("gh-aw/"):]
		}
		result[file+":"+parts[1]+":"+name] = value
		files[file] = true
	}
	return result, files, len(result) > 0, nil
}

func writeJSON(path string, report Report) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeSummary(path string, report Report) error {
	var b strings.Builder
	b.WriteString("### PureLock candidates\n\n")
	fmt.Fprintf(&b, "Analyzed %d functions, %d classified pure, %d candidates below the coverage ceiling.\n\n",
		report.FunctionsAnalyzed, report.PureFunctions, len(report.Candidates))
	if !report.CoverageLoaded {
		b.WriteString("Coverage data was unavailable; scores assume 0% coverage.\n\n")
	}
	b.WriteString("| # | Function | File:Line | Cov% | Cyclo | Fuzzable | Score |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for i, c := range report.Candidates {
		fmt.Fprintf(&b, "| %d | `%s%s` | `%s:%d` | %.1f | %d | %t | %.1f |\n",
			i+1, c.Receiver, c.Name, c.File, c.Line, c.Coverage, c.Complexity, c.FuzzFriendly, c.Score)
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func ensureDir(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := values[:0]
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
