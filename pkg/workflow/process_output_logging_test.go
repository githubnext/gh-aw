package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapProcessOutputForActionLog(t *testing.T) {
	command := wrapProcessOutputForActionLog("agent --run", "/tmp/gh-aw/agent-stdio.log", "Agent output")

	assert.Equal(t, `agent --run 2>&1 | tee -a /tmp/gh-aw/agent-stdio.log | "${GH_AW_NODE_BIN:-node}" "${RUNNER_TEMP}/gh-aw/actions/action_log.cjs" 'Agent output'`, command)
}
