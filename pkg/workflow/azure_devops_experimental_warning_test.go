//go:build !integration

package workflow

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureDevOpsSafeOutputsEmitExperimentalWarnings(t *testing.T) {
	tests := []struct {
		name       string
		safeOutput *SafeOutputsConfig
	}{
		{"create-work-item", &SafeOutputsConfig{CreateWorkItems: &CreateWorkItemConfig{}}},
		{"update-work-item", &SafeOutputsConfig{UpdateWorkItems: &UpdateWorkItemConfig{}}},
		{"comment-on-work-item", &SafeOutputsConfig{CommentOnWorkItems: &CommentOnWorkItemConfig{}}},
		{"assign-work-item", &SafeOutputsConfig{AssignWorkItems: &AssignWorkItemConfig{}}},
		{"link-work-items", &SafeOutputsConfig{LinkWorkItems: &LinkWorkItemsConfig{}}},
		{"upload-workitem-attachment", &SafeOutputsConfig{UploadWorkItemAttachments: &UploadWorkItemAttachmentConfig{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			var output bytes.Buffer

			compiler.emitExperimentalFeatureWarningsTo(&WorkflowData{SafeOutputs: tt.safeOutput}, &output)

			assert.Contains(t, output.String(), "Using experimental feature: "+tt.name)
			assert.Equal(t, 1, compiler.GetWarningCount())
		})
	}
}
