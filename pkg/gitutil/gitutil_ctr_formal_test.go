//go:build !integration

package gitutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGitRef_SafeRefAccepted(t *testing.T) {
	refs := []string{
		"main",
		"v1.2.3",
		"abcdef0123456789abcdef0123456789abcdef01",
		"feature/my-branch",
	}

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			require.NoError(t, ValidateGitRef(ref))
		})
	}
}

func TestValidateGitRef_HyphenPrefixRejected(t *testing.T) {
	ref := "-evil"
	err := ValidateGitRef(ref)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not start with '-'")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", ref))
}

func TestValidateGitRef_NulByteRejected(t *testing.T) {
	ref := "main\x00evil"
	err := ValidateGitRef(ref)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not contain NUL bytes")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", ref))
}

func TestValidateGitRef_TraversalRejected(t *testing.T) {
	ref := "main..evil"
	err := ValidateGitRef(ref)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not contain '..'")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", ref))
}

func TestValidateGitRef_EmptyRejected(t *testing.T) {
	err := ValidateGitRef("")
	require.Error(t, err)
	assert.ErrorContains(t, err, "must not be empty")
}

func TestValidateGitPath_SafePathAccepted(t *testing.T) {
	paths := []string{
		".github/workflows/workflow.md",
		"file.md",
		"docs/spec.md",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			require.NoError(t, ValidateGitPath(p))
		})
	}
}

func TestValidateGitPath_HyphenPrefixRejected(t *testing.T) {
	p := "-evil"
	err := ValidateGitPath(p)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not start with '-'")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", p))
}

func TestValidateGitPath_AbsolutePathRejected(t *testing.T) {
	p := "/etc/passwd"
	err := ValidateGitPath(p)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not be absolute")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", p))
}

func TestValidateGitPath_TraversalRejected(t *testing.T) {
	p := "dir/../../etc/passwd"
	err := ValidateGitPath(p)
	require.Error(t, err)
	require.ErrorContains(t, err, "must not contain '..' path traversal")
	assert.ErrorContains(t, err, fmt.Sprintf("%q", p))
}

func TestValidateGitPath_EmptyRejected(t *testing.T) {
	err := ValidateGitPath("")
	require.Error(t, err)
	assert.ErrorContains(t, err, "must not be empty")
}

func TestValidateGitRef_ErrorMessagesAreActionable(t *testing.T) {
	invalidRefs := []string{
		"-evil",
		"main\x00evil",
		"main..evil",
	}

	for _, ref := range invalidRefs {
		t.Run(ref, func(t *testing.T) {
			err := ValidateGitRef(ref)
			require.Error(t, err)
			assert.ErrorContains(t, err, fmt.Sprintf("%q", ref))
		})
	}
}
