//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildOrgWorkflowSearchQuery(t *testing.T) {
	assert.Equal(
		t,
		`org:octo path:.github/workflows filename:.lock.yml`,
		buildOrgWorkflowSearchQuery("octo", nil),
		"nil workflow filters should keep the base org search query",
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows filename:.lock.yml (filename:repo-assist.lock.yml OR filename:triage.lock.yml)`,
		buildOrgWorkflowSearchQuery("octo", []string{"triage.md", "repo-assist"}),
		"workflow filters should be normalized, sorted, and joined with OR",
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows filename:.lock.yml (filename:repo-assist.lock.yml)`,
		buildOrgWorkflowSearchQuery("octo", []string{"repo-assist", ".github/workflows/repo-assist.md"}),
		"duplicate workflow filters should collapse to a single filename predicate",
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows filename:.lock.yml`,
		buildOrgWorkflowSearchQuery("octo", []string{}),
		"an empty workflow filter slice should behave like nil",
	)

	assert.Equal(
		t,
		`org:octo path:.github/workflows filename:.lock.yml`,
		buildOrgWorkflowSearchQuery("octo", []string{""}),
		"all-empty workflow filters should fall back to the base org search query",
	)
}
