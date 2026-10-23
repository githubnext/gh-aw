//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPCACertFileParsing(t *testing.T) {
	for _, file := range []string{
		".github/certs/private-ca.pem",
		"/tmp/private-ca.pem",
		"/opt/company certs/ca.pem",
		"${{ github.workspace }}/certs/ca.pem",
		"${{ runner.temp }}/ca.pem",
	} {
		t.Run(file, func(t *testing.T) {
			raw := map[string]any{"url": "https://internal.example/mcp", "ca-cert-file": file}
			config, err := ParseMCPConfig("private", raw, raw)
			require.NoError(t, err)
			require.Equal(t, "http", config.Type)
			require.Equal(t, file, config.CACertFile)
			require.NoError(t, ValidateMCPConfigWithSchema(map[string]any{
				"type": "http", "url": config.URL, "ca-cert-file": file,
			}))
		})
	}
}

func TestMCPCACertFileInvalid(t *testing.T) {
	for _, value := range []any{
		"", " ", 42, false, nil, []any{"ca.pem"},
		"../ca.pem", "/tmp/../ca.pem", "certs/", "ca\n.pem",
		"$(touch injected)", "`touch injected`", "${HOME}/ca.pem",
		"${{ secrets.CA }}", "${{ github.event.issue.title }}", "a\"b.pem", "a\\b.pem", "a,b.pem",
		"${{ github.workspace }}/${{ runner.temp }}/ca.pem",
	} {
		t.Run("invalid-path", func(t *testing.T) {
			raw := map[string]any{"type": "http", "url": "https://internal.example/mcp", "ca-cert-file": value}
			_, err := ParseMCPConfig("private", raw, raw)
			require.ErrorContains(t, err, "ca-cert-file")
		})
	}
	for _, raw := range []map[string]any{
		{"type": "stdio", "container": "example/server", "ca-cert-file": "ca.pem"},
		{"type": "local", "command": "docker", "ca-cert-file": "ca.pem"},
		{"type": "http", "url": "http://internal.example", "ca-cert-file": "ca.pem"},
	} {
		_, err := ParseMCPConfig("private", raw, raw)
		require.ErrorContains(t, err, "ca-cert-file")
		require.Error(t, ValidateMCPConfigWithSchema(raw))
	}
}
