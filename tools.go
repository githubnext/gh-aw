//go:build tools

// Package tools tracks build-time tool dependencies.
// Import tools here with blank imports to include them in go.mod.
// This ensures consistent tool versions across development and CI.
package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/rhysd/actionlint/cmd/actionlint"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "golang.org/x/tools/gopls"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
