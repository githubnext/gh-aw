package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestFirewallFiltering tests that the --firewall flag correctly filters workflow runs
func TestFirewallFiltering(t *testing.T) {
	// Create a temporary directory for test logs
	tmpDir := t.TempDir()

	// Create mock run directories with aw_info.json files
	runWithFirewall := filepath.Join(tmpDir, "run-1")
	runWithoutFirewall := filepath.Join(tmpDir, "run-2")

	if err := os.MkdirAll(runWithFirewall, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(runWithoutFirewall, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create aw_info.json with firewall enabled
	awInfoWithFirewall := AwInfo{
		EngineID:     "copilot",
		EngineName:   "GitHub Copilot CLI",
		WorkflowName: "test-workflow",
		Firewall:     true,
	}
	data, _ := json.MarshalIndent(awInfoWithFirewall, "", "  ")
	if err := os.WriteFile(filepath.Join(runWithFirewall, "aw_info.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write aw_info.json: %v", err)
	}

	// Create aw_info.json without firewall
	awInfoWithoutFirewall := AwInfo{
		EngineID:     "copilot",
		EngineName:   "GitHub Copilot CLI",
		WorkflowName: "test-workflow",
		Firewall:     false,
	}
	data, _ = json.MarshalIndent(awInfoWithoutFirewall, "", "  ")
	if err := os.WriteFile(filepath.Join(runWithoutFirewall, "aw_info.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write aw_info.json: %v", err)
	}

	// Test parsing with firewall enabled
	info, err := parseAwInfo(filepath.Join(runWithFirewall, "aw_info.json"), false)
	if err != nil {
		t.Fatalf("Failed to parse aw_info.json: %v", err)
	}
	if !info.Firewall {
		t.Errorf("Expected firewall=true, got firewall=false")
	}

	// Test parsing without firewall
	info, err = parseAwInfo(filepath.Join(runWithoutFirewall, "aw_info.json"), false)
	if err != nil {
		t.Fatalf("Failed to parse aw_info.json: %v", err)
	}
	if info.Firewall {
		t.Errorf("Expected firewall=false, got firewall=true")
	}
}
