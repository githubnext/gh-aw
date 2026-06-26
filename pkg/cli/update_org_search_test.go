//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildOrgWorkflowSearchQuery(t *testing.T) {
	assert.Equal(
		t,
		`org:octo path:.github/workflows extension:md "source:"`,
		buildOrgWorkflowSearchQuery("octo", nil),
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows extension:md "source:" (filename:repo-assist.md OR filename:triage.md)`,
		buildOrgWorkflowSearchQuery("octo", []string{"triage.md", "repo-assist"}),
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows extension:md "source:" (filename:repo-assist.md)`,
		buildOrgWorkflowSearchQuery("octo", []string{"repo-assist", ".github/workflows/repo-assist.md"}),
	)
}
