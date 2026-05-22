//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveWorkflowExtension(t *testing.T) {
	tests := []struct {
		name       string
		fileResult *findWorkflowFileResult
		expected   string
		found      bool
	}{
		{
			name:       "prefers lock file",
			fileResult: &findWorkflowFileResult{lockExists: true, ymlExists: true, mdExists: true},
			expected:   ".lock.yml",
			found:      true,
		},
		{
			name:       "uses yml file",
			fileResult: &findWorkflowFileResult{ymlExists: true, mdExists: true},
			expected:   ".yml",
			found:      true,
		},
		{
			name:       "falls back to md as lock",
			fileResult: &findWorkflowFileResult{mdExists: true},
			expected:   ".lock.yml",
			found:      true,
		},
		{
			name:       "no file found",
			fileResult: &findWorkflowFileResult{},
			expected:   "",
			found:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension, found := resolveWorkflowExtension(tt.fileResult)
			assert.Equal(t, tt.expected, extension)
			assert.Equal(t, tt.found, found)
		})
	}
}
