//go:build !integration

package workflow

import (
	"errors"
	"testing"
)

func TestComputeRepositoryVisibility_CachesSuccessfulLookup(t *testing.T) {
	c := NewCompiler()
	c.SetRepositorySlug("github/gh-aw")

	originalFetchRepositoryVisibility := fetchRepositoryVisibility
	defer func() { fetchRepositoryVisibility = originalFetchRepositoryVisibility }()

	calls := 0
	fetchRepositoryVisibility = func(slug string) (string, error) {
		calls++
		if slug != "github/gh-aw" {
			t.Fatalf("fetchRepositoryVisibility() slug = %q, want %q", slug, "github/gh-aw")
		}
		return "public", nil
	}

	if got := c.computeRepositoryVisibility(); got != "public" {
		t.Fatalf("first computeRepositoryVisibility() = %q, want %q", got, "public")
	}
	if got := c.computeRepositoryVisibility(); got != "public" {
		t.Fatalf("second computeRepositoryVisibility() = %q, want %q", got, "public")
	}
	if calls != 1 {
		t.Fatalf("fetchRepositoryVisibility() calls = %d, want %d", calls, 1)
	}
}

func TestComputeRepositoryVisibility_CachesFailedLookup(t *testing.T) {
	c := NewCompiler()
	c.SetRepositorySlug("github/gh-aw")

	originalFetchRepositoryVisibility := fetchRepositoryVisibility
	defer func() { fetchRepositoryVisibility = originalFetchRepositoryVisibility }()

	calls := 0
	fetchRepositoryVisibility = func(slug string) (string, error) {
		calls++
		if slug != "github/gh-aw" {
			t.Fatalf("fetchRepositoryVisibility() slug = %q, want %q", slug, "github/gh-aw")
		}
		return "", errors.New("boom")
	}

	if got := c.computeRepositoryVisibility(); got != "" {
		t.Fatalf("first computeRepositoryVisibility() = %q, want empty string", got)
	}
	if got := c.computeRepositoryVisibility(); got != "" {
		t.Fatalf("second computeRepositoryVisibility() = %q, want empty string", got)
	}
	if calls != 1 {
		t.Fatalf("fetchRepositoryVisibility() calls = %d, want %d", calls, 1)
	}
}

func TestComputeRepositoryVisibility_InitializesCacheForZeroValueCompiler(t *testing.T) {
	c := &Compiler{}
	c.SetRepositorySlug("github/gh-aw")

	originalFetchRepositoryVisibility := fetchRepositoryVisibility
	defer func() { fetchRepositoryVisibility = originalFetchRepositoryVisibility }()

	fetchRepositoryVisibility = func(slug string) (string, error) {
		return "public", nil
	}

	if got := c.computeRepositoryVisibility(); got != "public" {
		t.Fatalf("computeRepositoryVisibility() = %q, want %q", got, "public")
	}
	if c.repositoryVisibility == nil {
		t.Fatal("computeRepositoryVisibility() left repositoryVisibility cache nil")
	}
	if c.repositoryVisibilitySet == nil {
		t.Fatal("computeRepositoryVisibility() left repositoryVisibilitySet nil")
	}
}
