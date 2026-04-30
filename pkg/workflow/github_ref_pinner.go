package workflow

import (
	"strings"

	actionpins "github.com/github/gh-aw/pkg/actionpins"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var githubRefPinnerLog = logger.New("workflow:github_ref_pinner")

// latestTagResolver is an optional capability that can be implemented by a
// SHAResolver to resolve the latest release tag for a repository when no
// explicit ref is provided.
type latestTagResolver interface {
	ResolveLatestTag(repo string) (tag, sha string, err error)
}

// pinContextGitHubRefPinner implements parser.GitHubRefPinner using an
// actionpins.PinContext. This allows the pinner to leverage dynamic SHA
// resolution when an ActionResolver is available.
type pinContextGitHubRefPinner struct {
	ctx *actionpins.PinContext
}

// Verify at compile time that pinContextGitHubRefPinner satisfies the interface.
var _ parser.GitHubRefPinner = (*pinContextGitHubRefPinner)(nil)

// PinGitHubRef resolves and pins a single github_ref value.
// The value is expected to be in owner/repo[@ref] or owner/repo/path[@ref] format.
// The function pins the owner/repo part and reconstructs the full path
// (including any subpath). If the reference cannot be pinned, the original
// value is returned unchanged.
//
// Resolution order for values without an explicit ref (e.g. "owner/repo"):
//  1. Embedded action_pins.json (fast, no network)
//  2. Dynamic latest-release resolution via the resolver (if available)
func (p *pinContextGitHubRefPinner) PinGitHubRef(value string) string {
	repo, subpath, ref := parser.ParseGitHubRefParts(value)

	var pinnedRepo string
	var err error

	if ref != "" {
		// Strip any existing SHA comment before resolving.
		rawRef, _, _ := strings.Cut(ref, " ")
		pinnedRepo, err = actionpins.ResolveActionPin(repo, rawRef, p.ctx)
	} else {
		pinnedRepo = actionpins.ResolveLatestActionPin(repo, p.ctx)
		// ResolveLatestActionPin returns empty when the repo has no embedded
		// action pins. Fall back to querying the GitHub API for the latest
		// release tag when a latestTagResolver is available.
		if pinnedRepo == "" {
			pinnedRepo = p.tryResolveLatestTag(repo)
		}
	}

	if err != nil || pinnedRepo == "" {
		githubRefPinnerLog.Printf("No pin available for github_ref %q (repo=%s, ref=%s), leaving unchanged", value, repo, ref)
		return value
	}

	result := parser.ReconstructGitHubRefValue(pinnedRepo, subpath)
	githubRefPinnerLog.Printf("Pinned github_ref: %q -> %q", value, result)
	return result
}

// tryResolveLatestTag attempts to resolve the latest release tag for the given
// repository using the latestTagResolver extension interface. Returns empty if
// the resolver is unavailable or resolution fails.
func (p *pinContextGitHubRefPinner) tryResolveLatestTag(repo string) string {
	if p.ctx == nil {
		return ""
	}
	ltr, ok := p.ctx.Resolver.(latestTagResolver)
	if !ok {
		return ""
	}
	return p.resolveWithLatestTag(repo, ltr)
}

// resolveWithLatestTag queries the GitHub API for the latest release tag of
// the repository and formats the result as a pinned action reference.
func (p *pinContextGitHubRefPinner) resolveWithLatestTag(repo string, ltr latestTagResolver) string {
	tag, sha, err := ltr.ResolveLatestTag(repo)
	if err != nil || sha == "" {
		githubRefPinnerLog.Printf("Failed to resolve latest tag for %q: %v", repo, err)
		return ""
	}
	result := actionpins.FormatPinnedActionReference(repo, sha, tag)
	githubRefPinnerLog.Printf("Resolved latest tag for github_ref %q: %s", repo, result)
	return result
}

// newImportCacheGitHubRefPinner creates a GitHubRefPinner suitable for use
// during import processing. It uses the compiler's shared action resolver to
// resolve SHAs dynamically, with a dedicated warnings map to avoid polluting
// the main workflow's warning cache.
func (c *Compiler) newImportCacheGitHubRefPinner() parser.GitHubRefPinner {
	_, resolver := c.getSharedActionResolver()
	ctx := &actionpins.PinContext{
		AllowActionRefs: true, // never fail compilation for unresolved github_ref pins
		Warnings:        make(map[string]bool),
	}
	if resolver != nil {
		ctx.Resolver = resolver
	}
	return &pinContextGitHubRefPinner{ctx: ctx}
}
