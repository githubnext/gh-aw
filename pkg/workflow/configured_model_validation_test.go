//go:build !integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWarnUnknownConfiguredModels(t *testing.T) {
	// This test redirects the process-wide stderr stream and cannot run in parallel.
	compiler := NewCompiler()
	compiler.SetConfiguredModelValidator(func(data *WorkflowData) []string {
		assert.Equal(t, "test", data.WorkflowID)
		return []string{"Model missing was not found in the active model inventory"}
	})

	oldStderr := os.Stderr
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() {
		os.Stderr = oldStderr
	}()

	compiler.warnUnknownConfiguredModels(&WorkflowData{WorkflowID: "test"}, "test.md")
	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)

	assert.Equal(t, 1, compiler.GetWarningCount())
	assert.Contains(t, string(bytes.TrimSpace(output)), "test.md: warning: Model missing")
}

func TestWarnUnknownConfiguredModelsWithoutInventory(t *testing.T) {
	t.Parallel()

	compiler := NewCompiler()
	compiler.warnUnknownConfiguredModels(&WorkflowData{}, "test.md")
	assert.Zero(t, compiler.GetWarningCount())
}
