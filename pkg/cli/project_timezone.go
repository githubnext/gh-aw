package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var projectTimezoneLog = logger.New("cli:project_timezone")

var findGitRootForProjectTimezone = gitutil.FindGitRoot
var loadRepoConfigForProjectTimezone = workflow.LoadRepoConfig

// ConfigureProjectTimezone applies the configured project timezone to CLI time rendering.
// Repo-level aw.json takes precedence over the enterprise default env var.
func ConfigureProjectTimezone() {
	utcOffset := strings.TrimSpace(compilerenv.ResolveDefaultUTC(""))

	gitRoot, err := findGitRootForProjectTimezone()
	if err == nil {
		if repoConfig, loadErr := loadRepoConfigForProjectTimezone(gitRoot); loadErr == nil && repoConfig != nil && strings.TrimSpace(repoConfig.UTC) != "" {
			utcOffset = strings.TrimSpace(repoConfig.UTC)
		} else if loadErr != nil {
			projectTimezoneLog.Printf("Failed to load repo config for UTC offset resolution: %v", loadErr)
		}
	} else {
		projectTimezoneLog.Printf("Failed to find git root for UTC offset resolution: %v", err)
	}

	if utcOffset == "" {
		console.ResetTimeLocation()
		return
	}

	location, err := workflow.ParseUTCOffsetLocation(utcOffset)
	if err != nil {
		projectTimezoneLog.Printf("Invalid configured UTC offset %q: %v", utcOffset, err)
		console.ResetTimeLocation()
		return
	}

	projectTimezoneLog.Printf("Configuring CLI rendered times to use UTC offset %q", utcOffset)
	console.SetTimeLocation(location)
}
