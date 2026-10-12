package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Records copied from run 38173405739 (Smoke Copilot Routed) with request IDs and
// timestamps redacted (request IDs stay distinct because token usage dedupes on them): the agent's routed classifier/agent traffic and the
// detection job's own token usage.
const (
	routedAgentTokenUsage = `{"_schema":"token-usage/v0.28.51","timestamp":"<redacted>","event":"token_usage","request_id":"<redacted>","provider":"copilot","model":"gpt-5.6-luna","path":"/responses","status":200,"streaming":false,"input_tokens":941,"output_tokens":283,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":3847,"response_bytes":14005,"purpose":"routing_classification","x_initiator":"agent"}
`
	detectionTokenUsage = `{"_schema":"token-usage/v0.28.51","timestamp":"<redacted>","event":"token_usage","request_id":"<redacted-1>","provider":"copilot","model":"claude-haiku-5.5","path":"/chat/completions","status":200,"streaming":true,"input_tokens":2271,"output_tokens":191,"cache_read_tokens":9290,"cache_write_tokens":7932,"duration_ms":2938,"response_bytes":10983,"x_initiator":"user"}
{"_schema":"token-usage/v0.28.51","timestamp":"<redacted>","event":"token_usage","request_id":"<redacted-2>","provider":"copilot","model":"claude-haiku-5.5","path":"/chat/completions","status":200,"streaming":true,"input_tokens":2,"output_tokens":130,"cache_read_tokens":17222,"cache_write_tokens":2955,"duration_ms":3081,"response_bytes":7895,"x_initiator":"agent"}
`
)

func TestReadDetectionModelUsage(t *testing.T) {
	t.Parallel()

	t.Run("reports the detection model separately from the routed agent model", func(t *testing.T) {
		t.Parallel()
		runDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(runDir, "usage", "agent"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(runDir, "usage", "detection"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "usage", "agent", "token_usage.jsonl"), []byte(routedAgentTokenUsage), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "usage", "detection", "token_usage.jsonl"), []byte(detectionTokenUsage), 0o600))

		assert.Equal(t, []DetectionModelUsage{{Model: "claude-haiku-5.5", Requests: 2}}, readDetectionModelUsage(runDir))

		agentUsage, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, agentUsage)
		assert.Contains(t, agentUsage.ByModel, "gpt-5.6-luna")
		assert.NotContains(t, agentUsage.ByModel, "claude-haiku-5.5", "detection usage must not be folded into the agent's token usage")
	})

	t.Run("returns nil without detection token usage", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, readDetectionModelUsage(t.TempDir()))
		assert.Nil(t, readDetectionModelUsage(""))
	})
}
