package cli

import (
	"os/exec"

	"github.com/github/gh-aw/pkg/logger"
)

var agentDownloadLog = logger.New("cli:agent_download")

func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	available := cmd.Run() == nil
	agentDownloadLog.Printf("Checked gh CLI availability: available=%v", available)
	return available
}
