package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var agentDownloadLog = logger.New("cli:agent_download")

// patchSkillFileURLs patches URLs in the skill file to use the correct ref.
func patchSkillFileURLs(content, ref string) string {
	agentDownloadLog.Printf("Patching skill file URLs to ref=%s (content length=%d)", ref, len(content))
	// Pattern 1: Convert local paths to GitHub URLs
	// `.github/aw/file.md` -> `https://github.com/github/gh-aw/blob/{ref}/.github/aw/file.md`
	content = strings.ReplaceAll(content, "`.github/aw/", fmt.Sprintf("`https://github.com/github/gh-aw/blob/%s/.github/aw/", ref))

	// Pattern 2: Update existing GitHub URLs to use the correct ref
	// https://github.com/github/gh-aw/blob/main/ -> https://github.com/github/gh-aw/blob/{ref}/
	if ref != "main" {
		content = strings.ReplaceAll(content, "/blob/main/", fmt.Sprintf("/blob/%s/", ref))
	}

	return content
}

func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	available := cmd.Run() == nil
	agentDownloadLog.Printf("Checked gh CLI availability: available=%v", available)
	return available
}
