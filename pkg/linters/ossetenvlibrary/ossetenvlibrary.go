// Package ossetenvlibrary implements a Go analysis linter that flags
// os.Setenv and os.Unsetenv calls in non-main, non-test packages.
package ossetenvlibrary

import (
	"fmt"

	"github.com/github/gh-aw/pkg/linters/internal/librarycall"
)

// restriction bans environment mutation outside of main and cmd/ packages.
var restriction = librarycall.Restriction{
	Linter:  "ossetenvlibrary",
	PkgPath: "os",
	Funcs:   []string{"Setenv", "Unsetenv"},
	Message: func(funcName, _ string) string {
		return fmt.Sprintf("os.%s mutates the process environment; pass configuration explicitly instead", funcName)
	},
}

// Analyzer is the os-setenv-in-library analysis pass.
var Analyzer = restriction.Analyzer("reports calls to os.Setenv or os.Unsetenv in non-main, non-test packages")
