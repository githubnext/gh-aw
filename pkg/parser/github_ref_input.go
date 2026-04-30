package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// githubRefPattern matches the owner/repo[@ref] or owner/repo/path[@ref] format.
// owner and repo must follow GitHub identifier rules (alphanumeric, hyphens, underscores, dots).
// path is an optional /… suffix of one or more path segments.
// ref is an optional @<ref> suffix where ref may be a tag, branch, or SHA.
var githubRefPattern = regexp.MustCompile(
	`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+(?:/[a-zA-Z0-9._-][a-zA-Z0-9._/\-]*)?(?:@[a-zA-Z0-9._/\-]+)?$`,
)

// ValidateGitHubRefInput checks that value is a valid github_ref string:
// owner/repo, owner/repo/path, owner/repo@ref, or owner/repo/path@ref.
// The ref suffix (after @) may be a tag, branch name, or SHA.
func ValidateGitHubRefInput(value string) error {
	if !githubRefPattern.MatchString(value) {
		return fmt.Errorf("value %q is not a valid github_ref (expected owner/repo, owner/repo/path, owner/repo@ref, or owner/repo/path@ref)", value)
	}
	// Ensure there are at least two slash-separated segments before the @.
	base, _, _ := strings.Cut(value, "@")
	parts := strings.SplitN(base, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("value %q is not a valid github_ref: must have at least owner/repo", value)
	}
	return nil
}

// ParseGitHubRefParts splits a github_ref value into its constituent parts.
// repo is always "owner/repo" (the first two slash-separated segments before the @).
// subpath is any additional path segments after owner/repo (may be empty).
// ref is the portion after @ (may be empty when not specified).
func ParseGitHubRefParts(value string) (repo, subpath, ref string) {
	// Split off the @ref suffix first.
	base, ref, _ := strings.Cut(value, "@")

	parts := strings.SplitN(base, "/", 3)
	if len(parts) < 2 {
		return base, "", ref
	}
	repo = parts[0] + "/" + parts[1]
	if len(parts) == 3 {
		subpath = parts[2]
	}
	return repo, subpath, ref
}

// ReconstructGitHubRefValue rebuilds a github_ref string from its pinned components.
// repo is the owner/repo (possibly followed by a sha comment like "owner/repo@sha # v1").
// subpath is any additional path after owner/repo (may be empty).
// The result is "repo/subpath" when subpath is non-empty, or "repo" otherwise.
// When the repo string already contains a pinned sha (owner/repo@sha # version),
// the subpath is inserted between the repo slug and the @sha annotation.
func ReconstructGitHubRefValue(pinnedRepo, subpath string) string {
	if subpath == "" {
		return pinnedRepo
	}
	// pinnedRepo may be "owner/repo@sha # version" — insert subpath before the @.
	repoBase, rest, hasSHA := strings.Cut(pinnedRepo, "@")
	if !hasSHA {
		return pinnedRepo + "/" + subpath
	}
	return repoBase + "/" + subpath + "@" + rest
}
