// Package nolint provides shared helpers for nolint-directive detection
// used by linters within pkg/linters.
package nolint

import (
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// HasDirective reports whether the given source position is covered by a
// suppression directive for linterName (or "nolint:all").  Both same-line and
// previous-line directives are recognised, matching golangci-lint behaviour.
func HasDirective(position token.Position, idx map[string]map[int]struct{}) bool {
	if position.Filename == "" {
		return false
	}

	noLintLines := idx[position.Filename]
	if noLintLines == nil {
		return false
	}

	_, sameLine := noLintLines[position.Line]
	_, previousLine := noLintLines[position.Line-1]
	return sameLine || previousLine
}

// BuildLineIndex scans all comments in the analysis pass and returns a map
// from filename → set of line numbers that carry a nolint directive for
// linterName (e.g. "errstringmatch") or "all".
func BuildLineIndex(pass *analysis.Pass, linterName string) map[string]map[int]struct{} {
	prefix := "nolint:" + linterName
	noLintLinesByFile := make(map[string]map[int]struct{}, len(pass.Files))
	for _, file := range pass.Files {
		filename := pass.Fset.PositionFor(file.Pos(), false).Filename
		if filename == "" {
			continue
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				text := strings.TrimPrefix(comment.Text, "//")
				if !strings.HasPrefix(text, prefix) && !strings.HasPrefix(text, "nolint:all") {
					continue
				}
				line := pass.Fset.PositionFor(comment.Slash, false).Line
				if noLintLinesByFile[filename] == nil {
					noLintLinesByFile[filename] = make(map[int]struct{})
				}
				noLintLinesByFile[filename][line] = struct{}{}
			}
		}
	}
	return noLintLinesByFile
}

// ImplementsError reports whether t implements the built-in error interface.
func ImplementsError(t types.Type) bool {
	obj := types.Universe.Lookup("error")
	if obj == nil {
		return false
	}
	errIface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return false
	}

	if types.Implements(t, errIface) {
		return true
	}
	if p, ok := t.(*types.Pointer); ok {
		return types.Implements(p, errIface)
	}
	return types.Implements(types.NewPointer(t), errIface)
}
