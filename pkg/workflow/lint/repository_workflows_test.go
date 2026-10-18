package lint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Keep the checker applied to the real compiled workflows, not just synthetic
// fixtures. Coverage warnings remain visible without claiming complete analysis.
func TestRepositoryAgenticWorkflows(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", ".github", "workflows", "*.lock.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	runner := Default()
	warnings := 0
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			workflow, err := actions.Parse(content)
			require.NoError(t, err)
			source, err := os.ReadFile(strings.TrimSuffix(file, ".lock.yml") + ".md")
			require.NoError(t, err)
			document, err := aw.Parse(string(source))
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), Input{Workflow: workflow, Document: document})
			require.NoError(t, err)
			for _, diagnostic := range diagnostics {
				assert.NotEqual(t, Error, diagnostic.Severity, "%s: %s", diagnostic.Path, diagnostic.Message)
				if diagnostic.Severity == Warning {
					warnings++
					t.Logf("%s: %s", diagnostic.Path, diagnostic.Message)
				}
			}
		})
	}
	t.Logf("Applied built-in rules to %d agentic workflows; %d explicit coverage warnings", len(files), warnings)
}
