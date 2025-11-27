package workflow

import (
	"strings"
	"testing"
)

func TestNoOpScriptBundling(t *testing.T) {
	script := getNoOpScript()

	// Check that load_agent_output.cjs is not being required
	if strings.Contains(script, `require("./load_agent_output.cjs")`) {
		t.Errorf("noop script contains require(\"./load_agent_output.cjs\") - should be bundled")
	}
	if strings.Contains(script, `require('./load_agent_output.cjs')`) {
		t.Errorf("noop script contains require('./load_agent_output.cjs') - should be bundled")
	}

	// Check that loadAgentOutput function exists (was inlined)
	if !strings.Contains(script, "loadAgentOutput") {
		t.Errorf("noop script does not contain loadAgentOutput function - bundling may have failed")
	}

	// Check that the main noop logic is present (from the inlined safe_output_runner module)
	if !strings.Contains(script, "itemTypePlural") {
		t.Errorf("noop script does not contain expected noop logic (missing itemTypePlural from inlined runner)")
	}

	// Check noop-specific function is present
	if !strings.Contains(script, "processNoopItems") {
		t.Errorf("noop script does not contain processNoopItems function")
	}

	// Verify staged mode logic is present
	if !strings.Contains(script, "GH_AW_SAFE_OUTPUTS_STAGED") {
		t.Errorf("noop script does not contain staged mode check")
	}
}
