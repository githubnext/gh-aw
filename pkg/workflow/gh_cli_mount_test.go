// All tests in gh_cli_mount_test.go have been disabled - network.firewall field deprecated
package workflow

import "testing"

// TestGhCLIMountInAWFContainer has been disabled - tests deprecated network.firewall behavior
func TestGhCLIMountInAWFContainer(t *testing.T) {
t.Skip("Test disabled - network.firewall field has been deprecated")
}
