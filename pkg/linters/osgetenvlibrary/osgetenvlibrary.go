// Package osgetenvlibrary implements a Go analysis linter that flags
// os.Getenv and os.LookupEnv calls in non-main, non-test packages.
package osgetenvlibrary

import (
	"fmt"

	"github.com/github/gh-aw/pkg/linters/internal/librarycall"
)

// restriction bans environment reads outside of main and cmd/ packages.
var restriction = librarycall.Restriction{
	Linter:  "osgetenvlibrary",
	PkgPath: "os",
	Funcs:   []string{"Getenv", "LookupEnv"},
	Message: func(funcName, _ string) string {
		return fmt.Sprintf("os.%s couples the library to the process environment; pass configuration explicitly instead", funcName)
	},
}

// Analyzer is the os-getenv-in-library analysis pass.
var Analyzer = restriction.Analyzer("reports calls to os.Getenv or os.LookupEnv in non-main, non-test packages")
