//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestGetVersionDelegatesToWorkflowVersion(t *testing.T) {
	originalVersion := workflow.GetVersion()
	defer workflow.SetVersion(originalVersion)

	workflow.SetVersion("workflow-direct")
	assert.Equal(t, "workflow-direct", GetVersion(), "cli.GetVersion should read workflow version")
}

func TestSetVersionInfoUpdatesWorkflowVersion(t *testing.T) {
	originalVersion := workflow.GetVersion()
	defer workflow.SetVersion(originalVersion)

	SetVersionInfo("set-version-info")
	assert.Equal(t, "set-version-info", workflow.GetVersion(), "SetVersionInfo should set workflow version")
	assert.Equal(t, "set-version-info", GetVersion(), "cli.GetVersion should return workflow version")
}
