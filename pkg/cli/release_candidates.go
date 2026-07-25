package cli

import (
	"slices"

	"github.com/github/gh-aw/pkg/semverutil"
)

type releaseCandidate struct {
	tag     string
	version *semverutil.SemanticVersion
}

// sortedCompatibleReleaseCandidates filters tags to stable semantic versions,
// applies the major-version compatibility rule, and returns candidates sorted
// newest-first.
func sortedCompatibleReleaseCandidates(releases []string, currentVer *semverutil.SemanticVersion, allowMajor bool) []releaseCandidate {
	var compatibleReleases []releaseCandidate
	for _, release := range releases {
		releaseVer := parseVersion(release)
		if releaseVer == nil || releaseVer.Pre != "" {
			continue
		}
		if !allowMajor && currentVer != nil && releaseVer.Major != currentVer.Major {
			continue
		}
		compatibleReleases = append(compatibleReleases, releaseCandidate{tag: release, version: releaseVer})
	}

	slices.SortFunc(compatibleReleases, func(a, b releaseCandidate) int {
		switch {
		case a.version.IsNewer(b.version):
			return -1
		case b.version.IsNewer(a.version):
			return 1
		default:
			return 0
		}
	})

	return compatibleReleases
}

// newerReleaseCandidates keeps only candidates newer than currentVer. When
// currentVer is nil, all candidates are returned unchanged.
func newerReleaseCandidates(candidates []releaseCandidate, currentVer *semverutil.SemanticVersion) []releaseCandidate {
	if currentVer == nil {
		return candidates
	}

	var newer []releaseCandidate
	for _, c := range candidates {
		if c.version.IsNewer(currentVer) {
			newer = append(newer, c)
		}
	}
	return newer
}
