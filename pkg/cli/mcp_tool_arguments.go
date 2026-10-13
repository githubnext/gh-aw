package cli

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func validateMCPPositionals(parameter string, values ...string) error {
	for _, value := range values {
		if strings.HasPrefix(value, "-") {
			return newMCPError(jsonrpc.CodeInvalidParams, parameter+" must not contain values starting with '-'; use tool parameters for options", nil)
		}
	}
	return nil
}

// Keep all CLI options before this boundary, including server-owned options.
func appendMCPPositionals(args []string, values ...string) []string {
	return append(append(args, "--"), values...)
}
