package cli

import (
	"strings"
	"time"

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
	timezoneName := strings.TrimSpace(compilerenv.ResolveDefaultUTC(""))

	gitRoot, err := findGitRootForProjectTimezone()
	if err == nil {
		if repoConfig, loadErr := loadRepoConfigForProjectTimezone(gitRoot); loadErr == nil && repoConfig != nil && strings.TrimSpace(repoConfig.UTC) != "" {
			timezoneName = strings.TrimSpace(repoConfig.UTC)
		} else if loadErr != nil {
			projectTimezoneLog.Printf("Failed to load repo config for timezone resolution: %v", loadErr)
		}
	} else {
		projectTimezoneLog.Printf("Failed to find git root for timezone resolution: %v", err)
	}

	if timezoneName == "" {
		console.ResetTimeLocation()
		return
	}

	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		projectTimezoneLog.Printf("Invalid configured timezone %q: %v", timezoneName, err)
		console.ResetTimeLocation()
		return
	}

	projectTimezoneLog.Printf("Configuring CLI rendered times to use timezone %q", timezoneName)
	console.SetTimeLocation(location)
}
