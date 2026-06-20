//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMentionsAllowTeamMembersCodemod(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	assert.Equal(t, "mentions-allow-team-members-to-allowed-collaborators", codemod.ID)
	assert.Equal(t, "Rename allow-team-members to allowed-collaborators in mentions", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestMentionsAllowTeamMembersCodemod_HappyPath(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-team-members: false
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": false,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "allowed-collaborators: false")
	assert.NotContains(t, result, "allow-team-members:")
}

func TestMentionsAllowTeamMembersCodemod_NoOp_NoSafeOutputs(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
name: test
---

# Test workflow`

	frontmatter := map[string]any{
		"name": "test",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMentionsAllowTeamMembersCodemod_NoOp_NoMentions(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  add-comment: {}
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"add-comment": map[string]any{},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMentionsAllowTeamMembersCodemod_NoOp_MentionsBoolean(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions: false
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": false,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMentionsAllowTeamMembersCodemod_NoOp_AlreadyMigrated(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allowed-collaborators: false
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allowed-collaborators": false,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMentionsAllowTeamMembersCodemod_Idempotent(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-team-members: true
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": true,
			},
		},
	}

	result1, applied1, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied1)

	// Apply again with the updated frontmatter — should be a no-op
	frontmatter2 := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allowed-collaborators": true,
			},
		},
	}
	result2, applied2, err := codemod.Apply(result1, frontmatter2)
	require.NoError(t, err)
	assert.False(t, applied2)
	assert.Equal(t, result1, result2)
}

func TestMentionsAllowTeamMembersCodemod_PreservesInlineComment(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-team-members: false # disable team member mentions
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": false,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "allowed-collaborators: false # disable team member mentions")
}

func TestMentionsAllowTeamMembersCodemod_PreservesIndentation(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-team-members: false
    allow-context: true
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": false,
				"allow-context":      true,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "    allowed-collaborators: false")
	assert.Contains(t, result, "    allow-context: true")
	assert.NotContains(t, result, "allow-team-members:")
}

func TestMentionsAllowTeamMembersCodemod_PreservesMarkdownBody(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-team-members: false
---

# Workflow title

Some markdown content here.
`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": false,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Workflow title")
	assert.Contains(t, result, "Some markdown content here.")
}

func TestMentionsAllowTeamMembersCodemod_NoOp_OldKeyAbsent(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions:
    allow-context: false
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-context": false,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMentionsAllowTeamMembersCodemod_FlowStyleMentions(t *testing.T) {
	codemod := getMentionsAllowTeamMembersCodemod()

	content := `---
safe-outputs:
  mentions: { allow-team-members: false, allow-context: true }
---

# Test workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"mentions": map[string]any{
				"allow-team-members": false,
				"allow-context":      true,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "mentions: { allowed-collaborators: false, allow-context: true }")
	assert.NotContains(t, result, "allow-team-members:")
}
