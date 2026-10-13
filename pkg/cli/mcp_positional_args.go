package cli

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// validateMCPPositionalArgs prevents client positionals from becoming CLI flags.
// Named tool parameters are the only supported way to select command options.
func validateMCPPositionalArgs(field string, values ...string) error {
	for _, value := range values {
		if strings.HasPrefix(value, "-") {
			return newMCPError(jsonrpc.CodeInvalidParams, field+" must not contain values beginning with '-'; use the tool's named parameters for options", nil)
		}
	}
	return nil
}
