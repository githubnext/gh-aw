// Package rawloginlib implements a Go analysis linter that flags
// standard log package calls in library (pkg/) packages.
package rawloginlib

import (
	"fmt"

	"github.com/github/gh-aw/pkg/linters/internal/librarycall"
)

// restriction bans standard log output functions in library code.
var restriction = librarycall.Restriction{
	Linter:  "rawloginlib",
	PkgPath: "log",
	Funcs:   []string{"Print", "Printf", "Println", "Panic", "Panicf", "Panicln"},
	Message: func(funcName, pkgPath string) string {
		return fmt.Sprintf("log.%s called in library package %s; use pkg/logger instead", funcName, pkgPath)
	},
}

var Analyzer = restriction.Analyzer("reports use of the standard log package in library packages where pkg/logger should be used instead")
