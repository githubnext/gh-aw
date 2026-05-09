//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
)

// TestSetupRepositoryContext_ValidScheduleSeedLocksSlug verifies that when
// --schedule-seed contains a valid "owner/repo" slug, setupRepositoryContext
// sets it on the compiler AND locks it so per-file git-remote detection
// cannot overwrite it.
func TestSetupRepositoryContext_ValidScheduleSeedLocksSlug(t *testing.T) {
	compiler := workflow.NewCompiler()
	config := CompileConfig{
		ScheduleSeed: "github/gh-aw",
	}

	setupRepositoryContext(compiler, config)

	if got := compiler.GetRepositorySlug(); got != "github/gh-aw" {
		t.Fatalf("expected slug github/gh-aw, got %q", got)
	}
	if !compiler.IsRepositorySlugLocked() {
		t.Fatal("slug should be locked after a valid --schedule-seed is applied")
	}

	// Simulate what compileWorkflowFile does: per-file remote slug should not win.
	compiler.SetRepositorySlugIfUnlocked("trask/gh-aw")
	if got := compiler.GetRepositorySlug(); got != "github/gh-aw" {
		t.Fatalf("per-file override should have been blocked; expected github/gh-aw, got %q", got)
	}
}

// TestSetupRepositoryContext_InvalidScheduleSeedDoesNotLock verifies that an
// invalid --schedule-seed value triggers a warning and falls back to git remote
// detection; the slug is NOT locked so per-file detection can still set it.
func TestSetupRepositoryContext_InvalidScheduleSeedDoesNotLock(t *testing.T) {
	compiler := workflow.NewCompiler()
	config := CompileConfig{
		ScheduleSeed: "not-valid", // missing slash
	}

	setupRepositoryContext(compiler, config)

	if compiler.IsRepositorySlugLocked() {
		t.Fatal("slug should not be locked when --schedule-seed value is invalid")
	}
}

// TestSetupRepositoryContext_EmptyScheduleSeedDoesNotLock verifies that omitting
// --schedule-seed leaves the slug unlocked so per-file git-remote detection applies.
func TestSetupRepositoryContext_EmptyScheduleSeedDoesNotLock(t *testing.T) {
	compiler := workflow.NewCompiler()
	config := CompileConfig{
		ScheduleSeed: "",
	}

	setupRepositoryContext(compiler, config)

	if compiler.IsRepositorySlugLocked() {
		t.Fatal("slug should not be locked when --schedule-seed is not provided")
	}
}

// TestSetupRepositoryContext_ScheduleSeedTakesPrecedenceOverPerFileRemote is an
// end-to-end regression guard: even after compileWorkflowFile calls
// SetRepositorySlugIfUnlocked, the slug remains the one from --schedule-seed.
func TestSetupRepositoryContext_ScheduleSeedTakesPrecedenceOverPerFileRemote(t *testing.T) {
	compiler := workflow.NewCompiler()

	// Simulate setupRepositoryContext with a valid --schedule-seed.
	config := CompileConfig{ScheduleSeed: "upstream/repo"}
	setupRepositoryContext(compiler, config)

	// Simulate the per-file detection path in compileWorkflowFile.
	compiler.SetRepositorySlugIfUnlocked("fork/repo")

	if got := compiler.GetRepositorySlug(); got != "upstream/repo" {
		t.Fatalf("--schedule-seed should take precedence over per-file remote; expected upstream/repo, got %q", got)
	}
}
