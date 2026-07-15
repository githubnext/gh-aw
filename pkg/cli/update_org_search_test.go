//go:build !integration

package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsValidOrgSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slug string
		want bool
	}{
		{name: "simple", slug: "github", want: true},
		{name: "hyphenated", slug: "my-org", want: true},
		{name: "singleCharacter", slug: "a", want: true},
		{name: "maxLength", slug: strings.Repeat("a", 39), want: true},
		{name: "empty", slug: "", want: false},
		{name: "tooLong", slug: strings.Repeat("a", 40), want: false},
		{name: "leadingHyphen", slug: "-org", want: false},
		{name: "trailingHyphen", slug: "org-", want: false},
		{name: "space", slug: "my org", want: false},
		{name: "colon", slug: "my:org", want: false},
		{name: "doubleHyphen", slug: "my--org", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isValidOrgSlug(tt.slug))
		})
	}
}

func TestSearchOrgWorkflowReposRejectsInvalidOrgBeforeSearch(t *testing.T) {
	origWait := waitForOrgRateLimitFn
	waitForOrgRateLimitFn = func(context.Context, string, bool) error {
		t.Fatal("waitForOrgRateLimitFn should not be called for invalid org")
		return nil
	}
	defer func() {
		waitForOrgRateLimitFn = origWait
	}()

	repos, err := searchOrgWorkflowRepos(context.Background(), "bad org", nil, false)
	require.Error(t, err)
	assert.Nil(t, repos)
	assert.EqualError(t, err, `invalid organization name "bad org": `+orgSlugConstraintDescription)
}
