//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepoSlug = "myorg/my-repo"
const testHeadSHA = "aabbccddeeff00112233445566778899aabbccdd"

// ---------------------------------------------------------------------------
// isLocalSkillRef
// ---------------------------------------------------------------------------

func TestIsLocalSkillRef(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want bool
	}{
		{"empty string", "", false},
		{"qualified ref", "owner/repo@aabbccddeeff00112233445566778899aabbccdd", false},
		{"qualified ref with path", "owner/repo/skills/my-skill@aabbccddeeff00112233445566778899aabbccdd", false},
		{"expression", "${{ secrets.SKILL_REF }}", false},
		{"expression with spaces", "  ${{ secrets.SKILL_REF }}  ", false},
		{"repo-relative path", ".github/skills/my-skill", true},
		{"relative path with dot-slash", "./skills/my-skill", true},
		{"bare name", "my-skill", true},
		{"nested path", "skills/nested/my-skill", true},
		{"non-sha ref still has at", "owner/repo@main", false}, // has @, excluded
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isLocalSkillRef(tt.spec))
		})
	}
}

// ---------------------------------------------------------------------------
// buildQualifiedSkillRef
// ---------------------------------------------------------------------------

func TestBuildQualifiedSkillRef(t *testing.T) {
	tests := []struct {
		name      string
		localPath string
		want      string
	}{
		{
			name:      "repo-relative path",
			localPath: ".github/skills/my-skill",
			want:      testRepoSlug + "/.github/skills/my-skill@" + testHeadSHA,
		},
		{
			name:      "dot-slash prefix stripped",
			localPath: "./skills/my-skill",
			want:      testRepoSlug + "/skills/my-skill@" + testHeadSHA,
		},
		{
			name:      "bare skill name",
			localPath: "my-skill",
			want:      testRepoSlug + "/my-skill@" + testHeadSHA,
		},
		{
			name:      "nested path preserved",
			localPath: ".github/skills/some/nested/skill",
			want:      testRepoSlug + "/.github/skills/some/nested/skill@" + testHeadSHA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQualifiedSkillRef(tt.localPath, testRepoSlug, testHeadSHA)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// trimYAMLQuotesSkill
// ---------------------------------------------------------------------------

func TestTrimYAMLQuotesSkill(t *testing.T) {
	assert.Equal(t, "value", trimYAMLQuotesSkill(`"value"`))
	assert.Equal(t, "value", trimYAMLQuotesSkill(`'value'`))
	assert.Equal(t, "value", trimYAMLQuotesSkill("value"))
	assert.Equal(t, `"mixed'`, trimYAMLQuotesSkill(`"mixed'`)) // mismatched quotes – untouched
	assert.Empty(t, trimYAMLQuotesSkill(""))
	assert.Equal(t, `"`, trimYAMLQuotesSkill(`"`)) // single char – untouched
}

// ---------------------------------------------------------------------------
// rewriteLocalSkillRefsInContent
// ---------------------------------------------------------------------------

func skillWorkflow(skillsBlock string) string {
	return "---\non:\n  workflow_dispatch:\nengine: copilot\n" + skillsBlock + "---\n\nDo some work.\n"
}

func TestRewriteLocalSkillRefsInContent(t *testing.T) {
	t.Run("rewrites string-form local ref", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - .github/skills/my-skill\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "- "+testRepoSlug+"/.github/skills/my-skill@"+testHeadSHA)
		assert.NotContains(t, got, ".github/skills/my-skill\n")
	})

	t.Run("rewrites object-form local ref", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - skill: .github/skills/my-skill\n    github-token: ${{ secrets.TOKEN }}\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "- skill: "+testRepoSlug+"/.github/skills/my-skill@"+testHeadSHA)
		// Auth fields must be preserved
		assert.Contains(t, got, "github-token: ${{ secrets.TOKEN }}")
	})

	t.Run("preserves already-qualified refs unchanged", func(t *testing.T) {
		qualified := "owner/repo/.github/skills/skill@aabbccddeeff00112233445566778899aabbccdd"
		content := skillWorkflow("skills:\n  - " + qualified + "\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, qualified)
	})

	t.Run("rewrites only local refs in mixed list", func(t *testing.T) {
		qualified := "owner/repo/.github/skills/skill@aabbccddeeff00112233445566778899aabbccdd"
		content := skillWorkflow("skills:\n  - .github/skills/local-skill\n  - " + qualified + "\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "- "+testRepoSlug+"/.github/skills/local-skill@"+testHeadSHA)
		assert.Contains(t, got, "- "+qualified)
	})

	t.Run("no-op when no skills key", func(t *testing.T) {
		content := "---\non:\n  workflow_dispatch:\n---\n\nDo work.\n"
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("no-op when skills array is empty", func(t *testing.T) {
		content := skillWorkflow("skills: []\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("no-op when no local refs present", func(t *testing.T) {
		qualified := "owner/repo/skill@aabbccddeeff00112233445566778899aabbccdd"
		content := skillWorkflow("skills:\n  - " + qualified + "\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("rewrites dot-slash prefixed local ref", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - ./skills/my-skill\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		// leading "./" should be stripped in the qualified ref
		assert.Contains(t, got, "- "+testRepoSlug+"/skills/my-skill@"+testHeadSHA)
		assert.NotContains(t, got, "./skills/my-skill\n")
	})

	t.Run("preserves expression refs unchanged", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - ${{ vars.SKILL_REF }}\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "${{ vars.SKILL_REF }}")
	})

	t.Run("rewrites quoted local ref", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - \".github/skills/my-skill\"\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "- "+testRepoSlug+"/.github/skills/my-skill@"+testHeadSHA)
	})

	t.Run("multiple local refs all rewritten", func(t *testing.T) {
		content := skillWorkflow("skills:\n  - .github/skills/skill-a\n  - .github/skills/skill-b\n")
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Contains(t, got, "- "+testRepoSlug+"/.github/skills/skill-a@"+testHeadSHA)
		assert.Contains(t, got, "- "+testRepoSlug+"/.github/skills/skill-b@"+testHeadSHA)
	})

	t.Run("no-op for content without frontmatter", func(t *testing.T) {
		content := "Just some markdown without frontmatter."
		got, err := rewriteLocalSkillRefsInContent(content, testRepoSlug, testHeadSHA)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})
}

// ---------------------------------------------------------------------------
// applyLocalSkillRefRewriting – non-local source is a no-op
// ---------------------------------------------------------------------------

func TestApplyLocalSkillRefRewriting_NonLocal(t *testing.T) {
	content := skillWorkflow("skills:\n  - .github/skills/my-skill\n")
	// sourceInfo.IsLocal = false → rewriting must be skipped
	sourceInfo := &FetchedWorkflow{IsLocal: false}
	got, err := applyLocalSkillRefRewriting(content, sourceInfo, AddOptions{})
	require.NoError(t, err)
	// Content must be unchanged
	assert.Equal(t, content, got)
}

func TestApplyLocalSkillRefRewriting_NilSource(t *testing.T) {
	content := skillWorkflow("skills:\n  - .github/skills/my-skill\n")
	got, err := applyLocalSkillRefRewriting(content, nil, AddOptions{})
	require.NoError(t, err)
	assert.Equal(t, content, got)
}
