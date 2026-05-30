//go:build !integration

package constants

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestPermissionConstantsValues(t *testing.T) {
	if FilePermSensitive != 0o600 {
		t.Fatalf("FilePermSensitive = %v, want 0o600", FilePermSensitive)
	}
	if FilePermPublic != 0o644 {
		t.Fatalf("FilePermPublic = %v, want 0o644", FilePermPublic)
	}
	if FilePermExecutable != 0o755 {
		t.Fatalf("FilePermExecutable = %v, want 0o755", FilePermExecutable)
	}
	if DirPermSensitive != 0o750 {
		t.Fatalf("DirPermSensitive = %v, want 0o750", DirPermSensitive)
	}
	if DirPermPublic != 0o755 {
		t.Fatalf("DirPermPublic = %v, want 0o755", DirPermPublic)
	}
}

func TestNoRawOctalPermissionLiteralsInOSCalls(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	pkgRoot := filepath.Join(repoRoot, "pkg")
	octalPattern := regexp.MustCompile(`^0o?[0-7]+$`)

	fset := token.NewFileSet()
	err := filepath.Walk(pkgRoot, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || filepath.Base(path) == "permissions_policy_test.go" {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("failed to parse %s: %v", path, parseErr)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			x, ok := sel.X.(*ast.Ident)
			if !ok || x.Name != "os" {
				return true
			}

			var argIndex int
			switch sel.Sel.Name {
			case "MkdirAll", "Mkdir", "Chmod":
				argIndex = 1
			case "WriteFile", "OpenFile":
				argIndex = 2
			default:
				return true
			}

			if argIndex >= len(call.Args) {
				return true
			}

			lit, ok := call.Args[argIndex].(*ast.BasicLit)
			if !ok || lit.Kind != token.INT {
				return true
			}

			if octalPattern.MatchString(lit.Value) {
				t.Errorf("raw octal permission literal %s in %s", lit.Value, path)
			}

			return true
		})

		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk pkg tree: %v", err)
	}
}
