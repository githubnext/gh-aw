//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
)

type awInfoBackwardCompatibilityTestCase struct {
	name                    string
	jsonData                string
	expectedFirewallVersion string
	expectedCLIVersion      string
	description             string
}

type awInfoMarshalingTestCase struct {
	name             string
	info             AwInfo
	shouldContainNew bool
	shouldContainOld bool
	description      string
}

func awInfoBackwardCompatibilityCases() []awInfoBackwardCompatibilityTestCase {
	return []awInfoBackwardCompatibilityTestCase{
		{name: "new field name awf_version", jsonData: `{"engine_id":"copilot","engine_name":"GitHub Copilot","cli_version":"1.0.0","awf_version":"v0.7.0","workflow_name":"test"}`,
			expectedFirewallVersion: "v0.7.0", expectedCLIVersion: "1.0.0", description: "Should parse new awf_version field"},
		{name: "old field name firewall_version", jsonData: `{"engine_id":"copilot","engine_name":"GitHub Copilot","firewall_version":"v0.6.0","workflow_name":"test"}`,
			expectedFirewallVersion: "v0.6.0", expectedCLIVersion: "", description: "Should parse old firewall_version field for backward compatibility"},
		{name: "both field names present - prefer new", jsonData: `{"engine_id":"copilot","engine_name":"GitHub Copilot","cli_version":"1.0.0","awf_version":"v0.7.0","firewall_version":"v0.6.0","workflow_name":"test"}`,
			expectedFirewallVersion: "v0.7.0", expectedCLIVersion: "1.0.0", description: "Should prefer awf_version when both are present"},
		{name: "no firewall version fields", jsonData: `{"engine_id":"copilot","engine_name":"GitHub Copilot","cli_version":"1.0.0","workflow_name":"test"}`,
			expectedFirewallVersion: "", expectedCLIVersion: "1.0.0", description: "Should handle missing firewall version gracefully"},
		{name: "empty firewall version", jsonData: `{"engine_id":"copilot","engine_name":"GitHub Copilot","cli_version":"1.0.0","awf_version":"","workflow_name":"test"}`,
			expectedFirewallVersion: "", expectedCLIVersion: "1.0.0", description: "Should handle empty awf_version"},
	}
}

func awInfoMarshalingCases() []awInfoMarshalingTestCase {
	return []awInfoMarshalingTestCase{
		{name: "with new field", info: AwInfo{EngineID: "copilot", EngineName: "GitHub Copilot", CLIVersion: "1.0.0", AwfVersion: "v0.7.0"},
			shouldContainNew: true, shouldContainOld: false, description: "Should marshal with awf_version when set"},
		{name: "with old field", info: AwInfo{EngineID: "copilot", EngineName: "GitHub Copilot", FirewallVersion: "v0.6.0"},
			shouldContainNew: false, shouldContainOld: true, description: "Should marshal with firewall_version when set"},
		{name: "with both fields", info: AwInfo{EngineID: "copilot", EngineName: "GitHub Copilot", CLIVersion: "1.0.0", AwfVersion: "v0.7.0", FirewallVersion: "v0.6.0"},
			shouldContainNew: true, shouldContainOld: true, description: "Should marshal both fields when both are set"},
	}
}

func decodeAwInfoJSONMap(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var fields map[string]any
	err := json.Unmarshal(data, &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON map: %v", err)
	}

	return fields
}

func assertAwInfoBackwardCompatibility(t *testing.T, tt awInfoBackwardCompatibilityTestCase) {
	t.Helper()

	var info AwInfo
	err := json.Unmarshal([]byte(tt.jsonData), &info)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if actualFirewallVersion := info.GetFirewallVersion(); actualFirewallVersion != tt.expectedFirewallVersion {
		t.Errorf("%s: GetFirewallVersion() = %q, want %q", tt.description, actualFirewallVersion, tt.expectedFirewallVersion)
	}
	if info.CLIVersion != tt.expectedCLIVersion {
		t.Errorf("%s: CLIVersion = %q, want %q", tt.description, info.CLIVersion, tt.expectedCLIVersion)
	}
	if tt.jsonData != "" && info.EngineID != "copilot" {
		t.Errorf("EngineID = %q, want %q", info.EngineID, "copilot")
	}
}

func assertAwInfoMarshaling(t *testing.T, tt awInfoMarshalingTestCase) {
	t.Helper()

	data, err := json.Marshal(tt.info)
	if err != nil {
		t.Fatalf("Failed to marshal AwInfo: %v", err)
	}

	fields := decodeAwInfoJSONMap(t, data)
	jsonStr := string(data)
	if tt.shouldContainNew {
		if _, exists := fields["awf_version"]; !exists {
			t.Errorf("%s: JSON should contain awf_version field, got: %s", tt.description, jsonStr)
		}
	}
	if tt.shouldContainOld {
		if _, exists := fields["firewall_version"]; !exists {
			t.Errorf("%s: JSON should contain firewall_version field, got: %s", tt.description, jsonStr)
		}
	}
}

// TestAwInfoBackwardCompatibility verifies that AwInfo can parse both old and new field names
func TestAwInfoBackwardCompatibility(t *testing.T) {
	for _, tt := range awInfoBackwardCompatibilityCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertAwInfoBackwardCompatibility(t, tt)
		})
	}
}

// TestAwInfoMarshaling verifies that AwInfo can be marshaled correctly
func TestAwInfoMarshaling(t *testing.T) {
	for _, tt := range awInfoMarshalingCases() {
		t.Run(tt.name, func(t *testing.T) {
			assertAwInfoMarshaling(t, tt)
		})
	}
}
