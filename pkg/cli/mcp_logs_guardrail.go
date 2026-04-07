package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpLogsGuardrailLog = logger.New("cli:mcp_logs_guardrail")

const (
	// CharsPerToken is the approximate number of characters per token
	// Using OpenAI's rule of thumb: ~4 characters per token
	CharsPerToken = 4

	// mcpLogsCacheDir is the directory where MCP logs data files are cached.
	// This is separate from the artifact download directory so that these
	// JSON summary files are not included in artifact uploads.
	mcpLogsCacheDir = "/tmp/gh-aw-logs-cache"
)

// MCPLogsGuardrailResponse represents the response returned by the logs tool.
// The full data is always written to a file; this response provides the file
// path so the caller can read the data.
type MCPLogsGuardrailResponse struct {
	Message  string `json:"message"`
	FilePath string `json:"file_path,omitempty"`
}

// estimateTokens estimates the number of tokens in a string
// Using the approximation: ~4 characters per token
func estimateTokens(text string) int {
	return len(text) / CharsPerToken
}

// buildLogsFileResponse writes the logs JSON output to a content-addressed cache
// file and returns a JSON response containing the file path and schema.
// The file is named by the SHA256 hash of its content so that identical results
// are deduplicated — if the file already exists it is not rewritten.
// The cache directory is kept separate from the artifact download directory so
// these summary files are never included in artifact uploads.
func buildLogsFileResponse(outputStr string) string {
	if err := os.MkdirAll(mcpLogsCacheDir, 0755); err != nil {
		mcpLogsGuardrailLog.Printf("Failed to create logs cache directory: %v", err)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to create logs cache directory: %v", err))
	}

	// Use SHA256 of content as filename for content-addressed deduplication.
	sum := sha256.Sum256([]byte(outputStr))
	fileName := hex.EncodeToString(sum[:]) + ".json"
	filePath := filepath.Join(mcpLogsCacheDir, fileName)

	// Skip writing if a file with identical content already exists.
	if _, err := os.Stat(filePath); err == nil {
		mcpLogsGuardrailLog.Printf("Logs data already cached at: %s", filePath)
	} else {
		if err := os.WriteFile(filePath, []byte(outputStr), 0600); err != nil {
			mcpLogsGuardrailLog.Printf("Failed to write logs data to file: %v", err)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to write logs data to file: %v", err))
		}
		mcpLogsGuardrailLog.Printf("Logs data written to file: %s (%d bytes)", filePath, len(outputStr))
	}

	response := MCPLogsGuardrailResponse{
		Message:  fmt.Sprintf("Logs data has been written to '%s'. Use the file_path to read the full data.", filePath),
		FilePath: filePath,
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		mcpLogsGuardrailLog.Printf("Failed to marshal logs file response: %v", err)
		return fmt.Sprintf(`{"message":"Logs data written to file","file_path":%q}`, filePath)
	}

	return string(responseJSON)
}

// buildLogsFileErrorResponse returns a JSON error response when file writing fails.
func buildLogsFileErrorResponse(errMsg string) string {
	response := MCPLogsGuardrailResponse{
		Message: fmt.Sprintf("⚠️  %s. The logs data could not be saved to a file.", errMsg),
	}
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"message":%q}`, errMsg)
	}
	return string(responseJSON)
}
