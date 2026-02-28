// Package repoutil provides utility functions for working with GitHub repository slugs and URLs.
package repoutil

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("repoutil:repoutil")

// SplitRepoSlug splits a repository slug (owner/repo) into owner and repo parts.
// Returns an error if the slug format is invalid.
func SplitRepoSlug(slug string) (owner, repo string, err error) {
	log.Printf("Splitting repo slug: %s", slug)
	parts := strings.Split(slug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		log.Printf("Invalid repo slug format: %s", slug)
		return "", "", fmt.Errorf("invalid repo format: %s", slug)
	}
	log.Printf("Split result: owner=%s, repo=%s", parts[0], parts[1])
	return parts[0], parts[1], nil
}

// SanitizeForFilename converts a repository slug (owner/repo) to a filename-safe string.
// Replaces "/" with "-". Returns "clone-mode" if the slug is empty.
func SanitizeForFilename(slug string) string {
	if slug == "" {
		return "clone-mode"
	}
	return strings.ReplaceAll(slug, "/", "-")
}
