// Package repoutil provides utility functions for working with GitHub repository slugs and URLs.
package repoutil

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var repoutilLog = logger.New("repoutil:repoutil")

// SplitRepoSlug splits a repository slug (owner/repo) into owner and repo parts.
// Returns an error if the slug format is invalid.
func SplitRepoSlug(slug string) (owner, repo string, err error) {
	repoutilLog.Printf("Splitting repo slug: %s", slug)
	parts := strings.Split(slug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		repoutilLog.Printf("Invalid repo slug format: %s", slug)
		return "", "", fmt.Errorf("invalid repo format: %s", slug)
	}
	repoutilLog.Printf("Split result: owner=%s, repo=%s", parts[0], parts[1])
	return parts[0], parts[1], nil
}

// NormalizeRepoForAPI splits a repo string of the form "[HOST/]owner/repo" into
// the owner/repo portion and an optional host. Most callers pass plain
// "owner/repo", but GHES and Proxima installs may supply "HOST/owner/repo".
func NormalizeRepoForAPI(repo string) (ownerRepo string, host string) {
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) == 3 {
		return parts[1] + "/" + parts[2], parts[0]
	}
	return repo, ""
}
