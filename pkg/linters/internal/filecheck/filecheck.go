package filecheck

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GeneratedIndex records generated Go source files by filename.
type GeneratedIndex map[string]struct{}

// Analyzer builds a shared generated-file index once per package so analyzers
// can reuse it via pass.ResultOf.
var Analyzer = &analysis.Analyzer{
	Name:             "generatedfileindex",
	Doc:              "indexes generated Go source files for gh-aw custom linters",
	ResultType:       reflect.TypeFor[GeneratedIndex](),
	RunDespiteErrors: true,
	Run: func(pass *analysis.Pass) (any, error) {
		return BuildGeneratedIndex(pass), nil
	},
}

// Index returns the shared generated-file index for pass.
func Index(pass *analysis.Pass) (GeneratedIndex, error) {
	idx, ok := pass.ResultOf[Analyzer].(GeneratedIndex)
	if !ok {
		return nil, fmt.Errorf("generated file analyzer result has unexpected type %T", pass.ResultOf[Analyzer])
	}
	return idx, nil
}

// BuildGeneratedIndex returns the set of generated Go source files in pass.
func BuildGeneratedIndex(pass *analysis.Pass) GeneratedIndex {
	generated := make(GeneratedIndex)
	for _, file := range pass.Files {
		if !ast.IsGenerated(file) {
			continue
		}
		pos := file.Pos()
		// PositionFor(..., false) preserves the original parsed filename, while
		// Position(...) may return a //line-adjusted logical filename.
		// Record both so callers using either position source can match.
		originalFilename := pass.Fset.PositionFor(pos, false).Filename
		if originalFilename != "" {
			generated[originalFilename] = struct{}{}
		}

		adjustedFilename := pass.Fset.Position(pos).Filename
		if adjustedFilename != "" && adjustedFilename != originalFilename {
			generated[adjustedFilename] = struct{}{}
		}
	}
	return generated
}

// IsTestFile reports whether filename is a Go test file path.
func IsTestFile(filename string) bool {
	return strings.HasSuffix(filepath.Base(filename), "_test.go")
}

// IsGeneratedFile reports whether filename is marked as generated.
func IsGeneratedFile(filename string, generated GeneratedIndex) bool {
	if filename == "" {
		return false
	}
	_, ok := generated[filename]
	return ok
}

// ShouldSkipFilename reports whether filename should be skipped by linters.
func ShouldSkipFilename(filename string, generated GeneratedIndex) bool {
	return IsTestFile(filename) || IsGeneratedFile(filename, generated)
}
