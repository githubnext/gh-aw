//go:build integration

package cli

import "os/exec"

// isDockerAvailable checks if Docker is available on the system (for integration tests)
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
