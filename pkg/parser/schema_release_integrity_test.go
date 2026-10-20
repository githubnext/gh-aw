//go:build !integration

package parser

import (
	"strings"
	"testing"
)

func TestMainWorkflowSchemaReleaseChecksumPins(t *testing.T) {
	t.Parallel()
	for _, location := range []string{"engine", "sandbox", "detection"} {
		for _, pin := range []any{strings.Repeat("a", 64), "", strings.Repeat("a", 63), strings.Repeat("A", 64), "${{ inputs.pin }}", 123, true, nil} {
			engine := map[string]any{"id": "copilot", "version": "1.0.99", "checksums-sha256": pin}
			data := map[string]any{"on": "push", "engine": "copilot"}
			switch location {
			case "engine":
				data["engine"] = engine
			case "sandbox":
				data["sandbox"] = map[string]any{"agent": map[string]any{
					"version": "v0.28.63", "checksums-sha256": pin,
				}}
			case "detection":
				data["safe-outputs"] = map[string]any{
					"create-issue": nil, "threat-detection": map[string]any{"engine": engine},
				}
			}
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(data, "release-pins.md")
			if (err == nil) != (pin == strings.Repeat("a", 64)) {
				t.Errorf("%s pin=%v: unexpected validation result: %v", location, pin, err)
			}
		}
	}
}
