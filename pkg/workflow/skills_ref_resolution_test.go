//go:build !integration

package workflow

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSkillWorkflowData(skills []string) *WorkflowData {
	refs := make([]SkillReference, 0, len(skills))
	for _, s := range skills {
		refs = append(refs, SkillReference{Skill: s})
	}
	return &WorkflowData{
		Skills:          append([]string(nil), skills...),
		SkillReferences: refs,
		Ctx:             context.Background(),
	}
}

func withCapturedStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() {
		os.Stderr = old
		_ = w.Close()
	}()

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestResolveFrontmatterSkillRefs_PinsNonSHARefUsingCache(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skill-ref-cache")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)
	const sha = "1f181b37d3fe5862ab590648f25a292e345b5de6"
	cache.Set("githubnext/skills", "main", sha)

	compiler := NewCompiler(WithVersion("dev"))
	data := newTestSkillWorkflowData([]string{"githubnext/skills@main"})
	data.ActionResolver = resolver

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	assert.Equal(t, "githubnext/skills@"+sha, data.Skills[0])
	assert.Equal(t, "githubnext/skills@"+sha, data.SkillReferences[0].Skill)
	assert.Empty(t, strings.TrimSpace(output), "no warning expected when resolution succeeds")
}

func TestResolveFrontmatterSkillRefs_PinsNonSHARefWithNilContext(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skill-ref-cache-nil-context")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)
	const sha = "1f181b37d3fe5862ab590648f25a292e345b5de6"
	cache.Set("githubnext/skills", "main", sha)

	compiler := NewCompiler(WithVersion("dev"))
	data := newTestSkillWorkflowData([]string{"githubnext/skills@main"})
	data.Ctx = nil
	data.ActionResolver = resolver

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	assert.Equal(t, "githubnext/skills@"+sha, data.Skills[0])
	assert.Equal(t, "githubnext/skills@"+sha, data.SkillReferences[0].Skill)
	assert.Empty(t, strings.TrimSpace(output), "no warning expected when resolution succeeds")
}

func TestResolveFrontmatterSkillRefs_LeavesFullSHAUnchanged(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	const sha = "1f181b37d3fe5862ab590648f25a292e345b5de6"
	data := newTestSkillWorkflowData([]string{"githubnext/skills@" + sha})

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	assert.Equal(t, "githubnext/skills@"+sha, data.Skills[0])
	assert.Empty(t, strings.TrimSpace(output))
}

func TestResolveFrontmatterSkillRefs_WarnsWhenNoRefSpecified(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	data := newTestSkillWorkflowData([]string{"githubnext/skills@"})

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	// Unpinned spec is kept as-is.
	assert.Equal(t, "githubnext/skills@", data.Skills[0])
	assert.Contains(t, output, "has no ref pinned")
	assert.Contains(t, output, "warning")
	assert.Equal(t, 1, compiler.GetWarningCount())
}

func TestResolveFrontmatterSkillRefs_LeavesLocalPathUnchanged(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	data := newTestSkillWorkflowData([]string{"skills/rig"})

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	assert.Equal(t, "skills/rig", data.Skills[0])
	assert.Empty(t, strings.TrimSpace(output))
}

func TestResolveFrontmatterSkillRefs_WarnsAndKeepsUnpinnedRefOnResolutionFailure(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skill-ref-cache-fail")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)
	// Mark this resolution as already failed so ResolveSHA short-circuits without
	// attempting a network call.
	resolver.failedResolutions[formatActionCacheKey("githubnext/does-not-exist", "no-such-ref")] = struct{}{}

	compiler := NewCompiler(WithVersion("dev"))
	data := newTestSkillWorkflowData([]string{"githubnext/does-not-exist@no-such-ref"})
	data.ActionResolver = resolver

	output := withCapturedStderr(t, func() {
		compiler.resolveFrontmatterSkillRefs(data, "workflow.md")
	})

	assert.Equal(t, "githubnext/does-not-exist@no-such-ref", data.Skills[0])
	assert.Contains(t, output, "failed to resolve ref")
	assert.Equal(t, 1, compiler.GetWarningCount())
}
