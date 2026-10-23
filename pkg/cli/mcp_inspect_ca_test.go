//go:build !integration

package cli

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestMCPInspectRejectsGatewayOnlyCACertFile(t *testing.T) {
	info, err := connectHTTPMCPServer(context.Background(), parser.RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Type: "http", URL: "https://private.example", CACertFile: ".github/ca.pem",
		},
	}, false)
	require.Nil(t, info)
	require.ErrorContains(t, err, "mcp inspect does not support ca-cert-file")
}
