//go:build !integration

package analyzerutil

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

func TestNew(t *testing.T) {
	run := func(*analysis.Pass) (any, error) { return nil, nil }

	analyzer := New("example", "example documentation", run)

	if analyzer.Name != "example" || analyzer.Doc != "example documentation" {
		t.Errorf("New() metadata = (%q, %q), want (%q, %q)", analyzer.Name, analyzer.Doc, "example", "example documentation")
	}
	if analyzer.URL != repositoryURL+"example" {
		t.Errorf("New() URL = %q, want %q", analyzer.URL, repositoryURL+"example")
	}
	if len(analyzer.Requires) != 3 || analyzer.Requires[0] != inspect.Analyzer || analyzer.Requires[1] != nolint.Analyzer || analyzer.Requires[2] != filecheck.Analyzer {
		t.Errorf("New() Requires = %v, want standard dependencies", analyzer.Requires)
	}
}

func TestNewAtPath(t *testing.T) {
	analyzer := NewAtPath("example", "example documentation", "example-path", nil)

	if analyzer.URL != repositoryURL+"example-path" {
		t.Errorf("NewAtPath() URL = %q, want %q", analyzer.URL, repositoryURL+"example-path")
	}
}
