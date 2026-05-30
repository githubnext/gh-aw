//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPServerCommand_PortExampleMentionsSSE(t *testing.T) {
	cmd := NewMCPServerCommand()
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Long, "Run HTTP server on port 8080 with SSE transport", "Port example should match SSE transport behavior")
}
