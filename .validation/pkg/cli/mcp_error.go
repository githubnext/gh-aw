package cli

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// newMCPError creates a jsonrpc.Error with the given code, message, and optional data.
// The data value is marshaled via mcpErrorData.
func newMCPError(code int64, msg string, data any) error {
	return &jsonrpc.Error{Code: code, Message: msg, Data: mcpErrorData(data)}
}

// mcpErrorData marshals data to JSON for use in jsonrpc.Error.Data field.
// Returns nil if marshaling fails to avoid errors in error handling.
func mcpErrorData(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		// Log the error but return nil to avoid breaking error handling
		mcpLog.Printf("Failed to marshal error data: %v", err)
		return nil
	}
	return data
}
