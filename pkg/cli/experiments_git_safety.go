package cli

import (
	"errors"
	"path"
	"strings"
)

func buildSafeGitShowObjectArg(ref, fileName string) (string, error) {
	if !isSafeExperimentStateRef(ref) {
		return "", errors.New("unsafe git ref")
	}
	if !isSafeGitTreePath(fileName) {
		return "", errors.New("unsafe git tree path")
	}
	return ref + ":" + fileName, nil
}

func isSafeExperimentStateRef(ref string) bool {
	if !isSafeGitRevisionArg(ref) {
		return false
	}

	// Allow direct object IDs (including abbreviated prefixes) for future callers
	// while rejecting revision operators.
	if isHexObjectIDPrefix(ref) {
		return true
	}

	trimmed := strings.TrimPrefix(ref, "origin/")
	if !strings.HasPrefix(trimmed, experimentsBranchPrefix) && !strings.HasPrefix(trimmed, evalsBranchPrefix) {
		return false
	}

	return isSafeGitRefName(trimmed)
}

func isHexObjectIDPrefix(ref string) bool {
	// Require >=7 chars to avoid accepting short hex-like experiment names as SHAs.
	// 64 keeps compatibility with SHA-256 object IDs.
	if len(ref) < 7 || len(ref) > 64 {
		return false
	}
	for _, r := range ref {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

// isSafeGitRefName validates a refname with check-ref-format-equivalent rules.
func isSafeGitRefName(ref string) bool {
	hasInvalidShape := ref == "" ||
		strings.HasPrefix(ref, "/") ||
		strings.HasSuffix(ref, "/") ||
		strings.HasSuffix(ref, ".")
	hasInvalidSequences := strings.Contains(ref, "//") ||
		strings.Contains(ref, "..") ||
		strings.Contains(ref, "@{") ||
		strings.Contains(ref, "\\")
	if hasInvalidShape || hasInvalidSequences {
		return false
	}

	for part := range strings.SplitSeq(ref, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
		for _, r := range part {
			if r <= ' ' || r == '~' || r == '^' || r == ':' || r == '?' || r == '*' || r == '[' || r == '\x7f' {
				return false
			}
		}
	}
	return true
}

// isSafeGitTreePath validates a git tree entry path used in "ref:path" syntax.
// Git tree paths always use forward slashes across platforms, so this intentionally
// uses the slash-based path package (not filepath) for normalization checks.
func isSafeGitTreePath(fileName string) bool {
	if fileName == "" || strings.HasPrefix(fileName, "-") {
		return false
	}
	if path.IsAbs(fileName) || strings.Contains(fileName, "\\") || strings.Contains(fileName, ":") || strings.ContainsRune(fileName, '\x00') {
		return false
	}
	clean := path.Clean(fileName)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return clean == fileName
}

// readRemoteExperimentState fetches experiment state from an experiments/* branch via the GitHub API.
// Returns an empty state on any error (branch missing, file absent, parse failure).
