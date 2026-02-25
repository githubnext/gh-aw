//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlaywrightDomainsCodemod(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	assert.Equal(t, "playwright-allowed-domains-migration", codemod.ID, "Codemod ID should match")
	assert.Equal(t, "Migrate playwright allowed_domains to network.allowed", codemod.Name, "Codemod name should match")
	assert.NotEmpty(t, codemod.Description, "Codemod should have a description")
	assert.Equal(t, "0.9.0", codemod.IntroducedIn, "Codemod version should match")
	require.NotNil(t, codemod.Apply, "Codemod should have an Apply function")
}

func TestPlaywrightDomainsCodemod_NoTools(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Should not apply when no tools block")
	assert.Equal(t, content, result, "Content should be unchanged")
}

func TestPlaywrightDomainsCodemod_NoPlaywright(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  github:
    mode: remote
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"github": map[string]any{"mode": "remote"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Should not apply when no playwright tool")
	assert.Equal(t, content, result, "Content should be unchanged")
}

func TestPlaywrightDomainsCodemod_NoAllowedDomains(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    version: v1.41.0
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{"version": "v1.41.0"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Should not apply when no allowed_domains")
	assert.Equal(t, content, result, "Content should be unchanged")
}

func TestPlaywrightDomainsCodemod_BasicMigration(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    allowed_domains:
      - github.com
      - api.github.com
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"allowed_domains": []any{"github.com", "api.github.com"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.NotContains(t, result, "allowed_domains", "Result should not contain allowed_domains")
	assert.Contains(t, result, "network:", "Result should contain top-level network")
	assert.Contains(t, result, "allowed:", "Result should contain network.allowed")
	assert.Contains(t, result, "github.com", "Result should contain github.com domain")
	assert.Contains(t, result, "api.github.com", "Result should contain api.github.com domain")
}

func TestPlaywrightDomainsCodemod_PreservesVersion(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    version: v1.41.0
    allowed_domains:
      - example.com
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"version":         "v1.41.0",
				"allowed_domains": []any{"example.com"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.NotContains(t, result, "allowed_domains", "Result should not contain allowed_domains")
	assert.Contains(t, result, "playwright:", "Result should preserve playwright block")
	assert.Contains(t, result, "version: v1.41.0", "Result should preserve version field")
	assert.Contains(t, result, "network:", "Result should contain top-level network")
	assert.Contains(t, result, "example.com", "Result should contain example.com domain")
}

func TestPlaywrightDomainsCodemod_MergesWithExistingNetwork(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    allowed_domains:
      - example.com
network:
  allowed:
    - python
    - existing.com
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"allowed_domains": []any{"example.com"},
			},
		},
		"network": map[string]any{
			"allowed": []any{"python", "existing.com"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.NotContains(t, result, "allowed_domains", "Result should not contain allowed_domains")
	assert.Contains(t, result, "python", "Result should preserve existing network domain")
	assert.Contains(t, result, "existing.com", "Result should preserve existing.com domain")
	assert.Contains(t, result, "example.com", "Result should add example.com domain")
}

func TestPlaywrightDomainsCodemod_DeduplicatesDomains(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    allowed_domains:
      - github.com
      - github.com
network:
  allowed:
    - github.com
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"allowed_domains": []any{"github.com", "github.com"},
			},
		},
		"network": map[string]any{
			"allowed": []any{"github.com"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.NotContains(t, result, "allowed_domains", "Result should not contain allowed_domains")
	// Count occurrences of github.com in the allowed block
	count := 0
	inAllowed := false
	for _, line := range splitLines(result) {
		if line == "  allowed:" {
			inAllowed = true
			continue
		}
		if inAllowed && len(line) > 0 && line[0] != ' ' {
			inAllowed = false
		}
		if inAllowed && strings.Contains(line, "github.com") {
			count++
		}
	}
	assert.Equal(t, 1, count, "github.com should appear exactly once in network.allowed")
}

func TestPlaywrightDomainsCodemod_SingleDomainString(t *testing.T) {
	codemod := getPlaywrightDomainsCodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    allowed_domains: example.com
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"allowed_domains": "example.com",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.NotContains(t, result, "allowed_domains", "Result should not contain allowed_domains")
	assert.Contains(t, result, "network:", "Result should contain network section")
	assert.Contains(t, result, "example.com", "Result should contain example.com domain")
}
