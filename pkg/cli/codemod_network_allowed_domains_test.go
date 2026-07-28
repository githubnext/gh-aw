//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNetworkAllowedDomainsCodemod(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	assert.Equal(t, "network-allowed-domains-rename", codemod.ID)
	assert.Equal(t, "Rename network.allowed-domains to network.allowed", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	require.NotNil(t, codemod.Apply)
}

func TestNetworkAllowedDomainsCodemod_NoNetwork(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test Workflow`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": map[string]any{"contents": "read"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestNetworkAllowedDomainsCodemod_NoAllowedDomains(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed:
    - defaults
    - github
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed": []any{"defaults", "github"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestNetworkAllowedDomainsCodemod_RenamesAllowedDomains(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed-domains:
    - defaults
    - github
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed-domains": []any{"defaults", "github"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "allowed-domains:", "Should remove allowed-domains field")
	assert.Contains(t, result, "allowed:", "Should contain allowed field")
	assert.Contains(t, result, "- defaults", "Should contain defaults domain")
	assert.Contains(t, result, "- github", "Should contain github domain")
}

func TestNetworkAllowedDomainsCodemod_MergesWithExistingAllowed(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed:
    - defaults
  allowed-domains:
    - github
    - api.example.com
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed":         []any{"defaults"},
			"allowed-domains": []any{"github", "api.example.com"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "allowed-domains:", "Should remove allowed-domains field")
	assert.Contains(t, result, "allowed:", "Should contain allowed field")
	assert.Contains(t, result, "- defaults", "Should contain defaults")
	assert.Contains(t, result, "- github", "Should contain github")
	assert.Contains(t, result, "- api.example.com", "Should contain api.example.com")
}

func TestNetworkAllowedDomainsCodemod_DeduplicatesEntries(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed:
    - defaults
    - github
  allowed-domains:
    - github
    - python
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed":         []any{"defaults", "github"},
			"allowed-domains": []any{"github", "python"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "allowed-domains:", "Should remove allowed-domains field")
	assert.Contains(t, result, "- defaults")
	assert.Contains(t, result, "- github")
	assert.Contains(t, result, "- python")

	// Count occurrences of "github" in items — should not be duplicated
	itemCount := 0
	for _, line := range splitNetworkLines(result) {
		if line == "    - github" || line == "  - github" {
			itemCount++
		}
	}
	assert.Equal(t, 1, itemCount, "github should not be duplicated after merge")
}

func TestNetworkAllowedDomainsCodemod_WildcardDomainsQuoted(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed-domains:
    - "*.example.com"
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed-domains": []any{"*.example.com"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "allowed-domains:")
	assert.Contains(t, result, "allowed:")
	assert.Contains(t, result, `"*.example.com"`, "Wildcard domain should be quoted")
}

func TestNetworkAllowedDomainsCodemod_NetworkWithoutAllowedGetsNewField(t *testing.T) {
	codemod := getNetworkAllowedDomainsCodemod()

	content := `---
on: workflow_dispatch
network:
  allowed-domains:
    - defaults
    - node
---

# Test Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"network": map[string]any{
			"allowed-domains": []any{"defaults", "node"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "allowed-domains:")
	assert.Contains(t, result, "allowed:")
	assert.Contains(t, result, "- defaults")
	assert.Contains(t, result, "- node")
}

// splitNetworkLines is a test helper to split content into lines.
func splitNetworkLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
