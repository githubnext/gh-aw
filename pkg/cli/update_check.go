package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/logger"
)

var updateCheckLog = logger.New("cli:update_check")

const (
	// maxReleasesToQuery is the maximum number of releases queried when prereleases are included.
	maxReleasesToQuery = 50
)

// Release represents a GitHub release
type Release struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// isRunningAsMCPServer reports whether the process was spawned by an MCP server,
// detected via the GH_AW_MCP_SERVER environment variable set by the server process.
func isRunningAsMCPServer() bool {
	return os.Getenv("GH_AW_MCP_SERVER") != ""
}

// getLastCheckFilePathFor returns the path to a named timestamp file in the
// shared gh-aw temp directory, creating the directory if necessary.
func getLastCheckFilePathFor(fileName string) string {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		updateCheckLog.Print("Could not determine temp directory")
		return ""
	}

	ghAwTmpDir := filepath.Join(tmpDir, "gh-aw")
	if err := os.MkdirAll(ghAwTmpDir, constants.DirPermPublic); err != nil {
		updateCheckLog.Printf("Error creating gh-aw temp directory: %v", err)
		return ""
	}

	return filepath.Join(ghAwTmpDir, fileName)
}

// getLatestRelease queries the GitHub API for the latest gh-aw release.
// When includePrereleases is true the first non-draft release (including
// pre-releases) is returned; otherwise only the latest stable release is
// returned.
//
// Always targets github.com explicitly so that users running against a GHE
// host still reach the canonical release registry.
func getLatestRelease(includePrereleases bool) (string, error) {
	updateCheckLog.Print("Querying GitHub API for latest release...")

	client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com"})
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub client: %w", err)
	}

	if includePrereleases {
		var releases []Release
		err = client.Get(fmt.Sprintf("repos/github/gh-aw/releases?per_page=%d", maxReleasesToQuery), &releases)
		if err != nil {
			return "", fmt.Errorf("failed to query releases: %w", err)
		}

		tag := findLatestPublishedReleaseTag(releases)
		updateCheckLog.Printf("Latest published release (pre-releases allowed): %s", tag)
		return tag, nil
	}

	var release Release
	err = client.Get("repos/github/gh-aw/releases/latest", &release)
	if err != nil {
		return "", fmt.Errorf("failed to query latest release: %w", err)
	}

	updateCheckLog.Printf("Latest release: %s (prerelease: %v)", release.TagName, release.Prerelease)

	// /releases/latest already excludes prereleases per the GitHub API contract,
	// but guard defensively in case the response ever changes.
	if release.Prerelease {
		return "", nil
	}

	return release.TagName, nil
}

// findLatestPublishedReleaseTag returns the first non-draft release tag from
// the releases API response, skipping entries without tag names.
func findLatestPublishedReleaseTag(releases []Release) string {
	for _, release := range releases {
		if release.Draft || release.TagName == "" {
			continue
		}
		return release.TagName
	}
	return ""
}
