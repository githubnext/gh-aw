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
	// This lives under /tmp/gh-aw/ so that agents can read the files, but
	// is separate from the artifact download directory (/tmp/gh-aw/aw-mcp/logs)
	// so that these JSON summary files are not included in artifact uploads.
	mcpLogsCacheDir = "/tmp/gh-aw/logs-cache"
)

// MCPLogsGuardrailResponse represents the response returned by the logs tool.
// The full data is always written to a file; this response provides the file
// path so the caller can read the data.
type MCPLogsGuardrailResponse struct {
	Message  string `json:"message"`
	FilePath string `json:"file_path,omitempty"`
}

// buildLogsFileResponse writes the logs JSON output to a content-addressed cache
// file and returns a JSON response containing the file path.
// The file is named by the SHA256 hash of its content so that identical results
// are deduplicated — if the file already exists it is not rewritten.
// The cache directory is kept separate from the artifact download directory so
// these summary files are never included in artifact uploads.
func buildLogsFileResponse(outputStr string) string {
	// Verify or create the cache directory. Use Lstat to detect symlinks and
	// refuse to follow them, hardening against symlink-based directory attacks.
	if info, err := os.Lstat(mcpLogsCacheDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return buildLogsFileErrorResponse(fmt.Sprintf("logs cache path %q is a symlink; refusing to use it", mcpLogsCacheDir))
		}
	} else if os.IsNotExist(err) {
		if mkErr := os.MkdirAll(mcpLogsCacheDir, 0700); mkErr != nil && !os.IsExist(mkErr) {
			mcpLogsGuardrailLog.Printf("Failed to create logs cache directory: %v", mkErr)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to create logs cache directory: %v", mkErr))
		}
	} else {
		mcpLogsGuardrailLog.Printf("Failed to stat logs cache directory: %v", err)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to access logs cache directory: %v", err))
	}

	// Use SHA256 of content as filename for content-addressed deduplication.
	sum := sha256.Sum256([]byte(outputStr))
	fileName := hex.EncodeToString(sum[:]) + ".json"
	filePath := filepath.Join(mcpLogsCacheDir, fileName)

	// Skip writing if a file with identical content already exists.
	if _, err := os.Lstat(filePath); err == nil {
		mcpLogsGuardrailLog.Printf("Logs data already cached at: %s", filePath)
	} else if os.IsNotExist(err) {
		// Write with O_EXCL to avoid following symlinks or races.
		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			mcpLogsGuardrailLog.Printf("Failed to create logs cache file: %v", err)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to create logs cache file: %v", err))
		}
		_, writeErr := f.WriteString(outputStr)
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(filePath)
			errMsg := writeErr
			if errMsg == nil {
				errMsg = closeErr
			}
			mcpLogsGuardrailLog.Printf("Failed to write logs data to file: %v", errMsg)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to write logs data to file: %v", errMsg))
		}
		mcpLogsGuardrailLog.Printf("Logs data written to file: %s (%d bytes)", filePath, len(outputStr))
	} else {
		mcpLogsGuardrailLog.Printf("Failed to stat logs cache file: %v", err)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to access logs cache file: %v", err))
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
