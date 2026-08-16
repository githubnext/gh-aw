//go:build !integration

package librarycall

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

func TestIsLibraryPackage(t *testing.T) {
	tests := []struct {
		name    string
		pkgPath string
		pkgName string
		want    bool
	}{
		{name: "library package", pkgPath: "github.com/github/gh-aw/pkg/workflow", pkgName: "workflow", want: true},
		{name: "main package", pkgPath: "github.com/github/gh-aw/tools/gen", pkgName: "main", want: false},
		{name: "main path suffix", pkgPath: "github.com/github/gh-aw/main", pkgName: "app", want: false},
		{name: "cmd package", pkgPath: "github.com/github/gh-aw/cmd/gh-aw/internal", pkgName: "internal", want: false},
		{name: "test binary", pkgPath: "github.com/github/gh-aw/pkg/workflow.test", pkgName: "workflow", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := &analysis.Pass{Pkg: types.NewPackage(tt.pkgPath, tt.pkgName)}
			if got := IsLibraryPackage(pass); got != tt.want {
				t.Errorf("IsLibraryPackage(%q) = %v, want %v", tt.pkgPath, got, tt.want)
			}
		})
	}
}

func TestIsLibraryPackageNilPass(t *testing.T) {
	if IsLibraryPackage(nil) {
		t.Error("IsLibraryPackage(nil) = true, want false")
	}
	if IsLibraryPackage(&analysis.Pass{}) {
		t.Error("IsLibraryPackage(pass without Pkg) = true, want false")
	}
}

func TestRestrictionAnalyzer(t *testing.T) {
	restriction := Restriction{
		Linter:  "examplelinter",
		PkgPath: "os",
		Funcs:   []string{"Exit"},
		Message: func(funcName, pkgPath string) string { return funcName + " in " + pkgPath },
	}

	analyzer := restriction.Analyzer("example documentation")

	if analyzer.Name != "examplelinter" || analyzer.Doc != "example documentation" {
		t.Errorf("Analyzer() metadata = (%q, %q), want (%q, %q)", analyzer.Name, analyzer.Doc, "examplelinter", "example documentation")
	}
	if len(analyzer.Requires) != 3 || analyzer.Requires[0] != inspect.Analyzer || analyzer.Requires[1] != nolint.Analyzer || analyzer.Requires[2] != filecheck.Analyzer {
		t.Errorf("Analyzer() Requires = %v, want standard dependencies", analyzer.Requires)
	}
}

func TestRestrictionRunSkipsNonLibraryPackage(t *testing.T) {
	restriction := Restriction{
		Linter:  "examplelinter",
		PkgPath: "os",
		Funcs:   []string{"Exit"},
		Message: func(funcName, pkgPath string) string { return funcName + " in " + pkgPath },
	}

	// A pass without ResultOf entries would fail if the restriction analyzed
	// the package, so a nil error proves the cmd/ package was skipped early.
	pass := &analysis.Pass{Pkg: types.NewPackage("github.com/github/gh-aw/cmd/gh-aw", "main")}
	result, err := restriction.Run(pass)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("Run() result = %v, want nil", result)
	}
}
