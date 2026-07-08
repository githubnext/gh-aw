//go:build !integration

// Note: these tests mutate the package-level fetchRepositoryVisibility variable.
// Do not mark any test in this file as t.Parallel() without adding synchronisation
// around that variable, as concurrent mutations would cause a data race.

package workflow

import "testing"

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

func TestComputeRepositoryVisibility_DoesNotCacheFailedLookup(t *testing.T) {
	c := NewCompiler()
	c.SetRepositorySlug("github/gh-aw")

	originalFetchRepositoryVisibility := fetchRepositoryVisibility
	defer func() { fetchRepositoryVisibility = originalFetchRepositoryVisibility }()

	calls := 0
	firstCall := true
	fetchRepositoryVisibility = func(slug string) (string, error) {
		calls++
		if slug != "github/gh-aw" {
			t.Fatalf("fetchRepositoryVisibility() slug = %q, want %q", slug, "github/gh-aw")
		}
		if firstCall {
			firstCall = false
			return "", assertiveTestError("boom")
		}
		return "public", nil
	}

	if got := c.computeRepositoryVisibility(); got != "" {
		t.Fatalf("first computeRepositoryVisibility() = %q, want empty string", got)
	}
	if got := c.computeRepositoryVisibility(); got != "public" {
		t.Fatalf("second computeRepositoryVisibility() = %q, want %q", got, "public")
	}
	if calls != 2 {
		t.Fatalf("fetchRepositoryVisibility() calls = %d, want %d", calls, 2)
	}
}

func TestComputeRepositoryVisibility_DoesNotCacheEmptyVisibility(t *testing.T) {
	c := NewCompiler()
	c.SetRepositorySlug("github/gh-aw")

	originalFetchRepositoryVisibility := fetchRepositoryVisibility
	defer func() { fetchRepositoryVisibility = originalFetchRepositoryVisibility }()

	calls := 0
	fetchRepositoryVisibility = func(slug string) (string, error) {
		calls++
		// Simulate an API response that is valid JSON but has no recognisable
		// visibility field — fetchRepositoryVisibility returns ("", nil).
		return "", nil
	}

	if got := c.computeRepositoryVisibility(); got != "" {
		t.Fatalf("first computeRepositoryVisibility() = %q, want empty string", got)
	}
	if got := c.computeRepositoryVisibility(); got != "" {
		t.Fatalf("second computeRepositoryVisibility() = %q, want empty string (must not be cached)", got)
	}
	if calls != 2 {
		t.Fatalf("fetchRepositoryVisibility() calls = %d, want 2 (empty string must not be cached)", calls)
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
	if got := c.repositoryVisibility["github/gh-aw"]; got != "public" {
		t.Fatalf("repositoryVisibility[%q] = %q, want %q", "github/gh-aw", got, "public")
	}
}

func TestComputeRepositoryVisibility_EmptySlugReturnsEmpty(t *testing.T) {
	c := NewCompiler()
	// Do NOT call SetRepositorySlug — slug is intentionally empty.
	got := c.computeRepositoryVisibility()
	if got != "" {
		t.Fatalf("computeRepositoryVisibility() = %q, want empty string for unset slug", got)
	}
}

type assertiveTestError string

func (e assertiveTestError) Error() string { return string(e) }
