package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
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
// Partial reports known collection or episode evidence gaps. Continuation and
// Continuations preserve the available single-target and per-target cursors.
type MCPLogsGuardrailResponse struct {
	Message         string                 `json:"message"`
	FilePath        string                 `json:"file_path,omitempty"`
	Partial         bool                   `json:"partial,omitempty"`
	Continuation    *ContinuationData      `json:"continuation,omitempty"`
	Continuations   []WorkflowContinuation `json:"continuations,omitempty"`
	CoverageStatus  string                 `json:"coverage_status,omitempty"`
	CoverageReasons []string               `json:"coverage_reasons,omitempty"`
}

func extractLogsCollectionState(outputStr string) MCPLogsGuardrailResponse {
	var parsed struct {
		Continuation     json.RawMessage `json:"continuation"`
		Continuations    json.RawMessage `json:"continuations"`
		EpisodeRetrieval json.RawMessage `json:"episode_retrieval"`
		Episodes         json.RawMessage `json:"episodes"`
	}
	if err := json.Unmarshal([]byte(outputStr), &parsed); err != nil {
		mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse output JSON: %v", err)
		return MCPLogsGuardrailResponse{Partial: true, CoverageStatus: "unknown"}
	}
	response := MCPLogsGuardrailResponse{}
	extractLogsCursors(parsed.Continuation, parsed.Continuations, &response)
	extractLogsEpisodeRetrieval(parsed.EpisodeRetrieval, &response)
	extractLogsEpisodeCoverage(parsed.Episodes, &response)
	if response.CoverageStatus == "partial" {
		response.Partial = true
	}
	slices.Sort(response.CoverageReasons)
	response.CoverageReasons = slices.Compact(response.CoverageReasons)
	return response
}

func extractLogsCursors(continuationRaw, continuationsRaw json.RawMessage, response *MCPLogsGuardrailResponse) {
	if len(continuationRaw) > 0 && string(continuationRaw) != "null" {
		var continuation ContinuationData
		if err := json.Unmarshal(continuationRaw, &continuation); err != nil {
			mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse continuation: %v", err)
			response.Partial = true
			response.CoverageStatus = "unknown"
		} else {
			response.Continuation = &continuation
		}
	}
	if len(continuationsRaw) > 0 && string(continuationsRaw) != "null" {
		if err := json.Unmarshal(continuationsRaw, &response.Continuations); err != nil {
			mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse continuations: %v", err)
			response.Partial = true
			response.CoverageStatus = "unknown"
		}
	}
	response.Partial = response.Partial || response.Continuation != nil || len(response.Continuations) > 0
}

func extractLogsEpisodeRetrieval(retrievalRaw json.RawMessage, response *MCPLogsGuardrailResponse) {
	if len(retrievalRaw) > 0 && string(retrievalRaw) != "null" {
		// Selected-run coverage cannot establish complete episode membership.
		response.CoverageStatus = "unknown"
		var retrieval EpisodeRetrieval
		if err := json.Unmarshal(retrievalRaw, &retrieval); err != nil {
			mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse episode retrieval: %v", err)
			response.Partial = true
		} else {
			for _, reason := range retrieval.StopReasons {
				response.CoverageReasons = append(response.CoverageReasons, retrievalCoverageReason(reason))
			}
			for _, issue := range retrieval.Issues {
				response.CoverageReasons = append(response.CoverageReasons, retrievalCoverageReason(issue.Reason))
			}
			if len(retrieval.StopReasons) > 0 || len(retrieval.Issues) > 0 {
				response.CoverageStatus = "partial"
			}
		}
	}
}

func extractLogsEpisodeCoverage(episodesRaw json.RawMessage, response *MCPLogsGuardrailResponse) {
	if len(episodesRaw) > 0 && string(episodesRaw) != "null" {
		var episodes []json.RawMessage
		if err := json.Unmarshal(episodesRaw, &episodes); err != nil {
			mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse episodes: %v", err)
			response.Partial = true
			if response.CoverageStatus == "" {
				response.CoverageStatus = "unknown"
			}
		} else if len(episodes) > 0 {
			// Selected-run coverage cannot establish complete episode membership.
			if response.CoverageStatus == "" {
				response.CoverageStatus = "unknown"
			}
			for _, rawEpisode := range episodes {
				var episode struct {
					Coverage EpisodeCoverage `json:"coverage"`
				}
				if err := json.Unmarshal(rawEpisode, &episode); err != nil {
					mcpLogsGuardrailLog.Printf("extractLogsCollectionState: failed to parse episode coverage: %v", err)
					response.Partial = true
					continue
				}
				if episode.Coverage.Status == "partial" || episode.Coverage.CollectionTruncated {
					response.CoverageStatus = "partial"
				}
				response.CoverageReasons = append(response.CoverageReasons, episode.Coverage.Reasons...)
			}
		}
	}
}

// extractLogsStaleWarning returns the top-level "stale_warning" field embedded
// in the logs JSON output, or "" when absent or unparseable. This is a
// dedicated field (distinct from the generic "message" field, which is also
// used for non-warning hints such as the usage-only artifact hint) so that
// only genuine stale-data warnings are surfaced as "WARNING" in the MCP
// response.
func extractLogsStaleWarning(outputStr string) string {
	var parsed struct {
		StaleWarning string `json:"stale_warning"`
	}
	if err := json.Unmarshal([]byte(outputStr), &parsed); err != nil {
		mcpLogsGuardrailLog.Printf("extractLogsStaleWarning: failed to parse output JSON: %v", err)
		return ""
	}
	return parsed.StaleWarning
}

// buildLogsFileResponse writes the logs JSON output to a content-addressed cache
// file and returns a JSON response containing the file path.
// The file is named by the SHA256 hash of its content so that identical results
// are deduplicated — if the file already exists it is not rewritten.
// The cache directory is kept separate from the artifact download directory so
// these summary files are never included in artifact uploads.
func buildLogsFileResponse(outputStr string) string { //nolint:largefunc // Keep the existing symlink-safe cache write and response assembly together.
	// Verify or create the cache directory. Use Lstat to detect symlinks and
	// refuse to follow them, hardening against symlink-based directory attacks.
	if info, err := os.Lstat(mcpLogsCacheDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return buildLogsFileErrorResponse(fmt.Sprintf("logs cache path %q is a symlink; refusing to use it", mcpLogsCacheDir))
		}
		if !info.IsDir() {
			return buildLogsFileErrorResponse(fmt.Sprintf("logs cache path %q is not a directory", mcpLogsCacheDir))
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		if mkErr := os.MkdirAll(mcpLogsCacheDir, constants.DirPermPublic); mkErr != nil && !errors.Is(mkErr, fs.ErrExist) {
			mcpLogsGuardrailLog.Printf("Failed to create logs cache directory: %v", mkErr)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to create logs cache directory: %v", mkErr))
		}
	} else {
		mcpLogsGuardrailLog.Printf("Failed to stat logs cache directory: %v", err)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to access logs cache directory: %v", err))
	}
	if chmodErr := os.Chmod(mcpLogsCacheDir, constants.DirPermPublic); chmodErr != nil {
		mcpLogsGuardrailLog.Printf("Failed to set logs cache directory permissions: %v", chmodErr)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to set logs cache directory permissions: %v", chmodErr))
	}

	// Use SHA256 of content as filename for content-addressed deduplication.
	sum := sha256.Sum256([]byte(outputStr))
	fileName := hex.EncodeToString(sum[:]) + ".json"
	filePath := filepath.Join(mcpLogsCacheDir, fileName)

	// Skip writing if a file with identical content already exists.
	if fileInfo, err := os.Lstat(filePath); err == nil {
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return buildLogsFileErrorResponse(fmt.Sprintf("logs cache file path %q is a symlink; refusing to use it", filePath))
		}
		if !fileInfo.Mode().IsRegular() {
			return buildLogsFileErrorResponse(fmt.Sprintf("logs cache file path %q is not a regular file", filePath))
		}
		if chmodErr := os.Chmod(filePath, constants.FilePermPublic); chmodErr != nil {
			mcpLogsGuardrailLog.Printf("Failed to update logs cache file permissions: %v", chmodErr)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to set logs cache file permissions: %v", chmodErr))
		}
		mcpLogsGuardrailLog.Printf("Logs data already cached at: %s", filePath)
	} else if errors.Is(err, fs.ErrNotExist) {
		// Write with O_EXCL to avoid following symlinks or races.
		writeErr := func() (err error) {
			f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, constants.FilePermPublic)
			if err != nil {
				return fmt.Errorf("failed to create logs cache file: %w", err)
			}
			defer func() {
				if closeErr := f.Close(); closeErr != nil && err == nil {
					err = fmt.Errorf("failed to write logs data to file: %w", closeErr)
				}
			}()

			if _, err = f.WriteString(outputStr); err != nil {
				return fmt.Errorf("failed to write logs data to file: %w", err)
			}
			return nil
		}()
		if writeErr != nil {
			mcpLogsGuardrailLog.Printf("Failed to populate logs cache file %s: %v", filePath, writeErr)
			_ = os.Remove(filePath)
			return buildLogsFileErrorResponse(writeErr.Error())
		}
		if chmodErr := os.Chmod(filePath, constants.FilePermPublic); chmodErr != nil {
			_ = os.Remove(filePath)
			mcpLogsGuardrailLog.Printf("Failed to set logs cache file permissions: %v", chmodErr)
			return buildLogsFileErrorResponse(fmt.Sprintf("failed to set logs cache file permissions: %v", chmodErr))
		}
		mcpLogsGuardrailLog.Printf("Logs data written to file: %s (%d bytes)", filePath, len(outputStr))
	} else {
		mcpLogsGuardrailLog.Printf("Failed to stat logs cache file: %v", err)
		return buildLogsFileErrorResponse(fmt.Sprintf("failed to access logs cache file: %v", err))
	}

	response := extractLogsCollectionState(outputStr)
	response.FilePath = filePath

	var msgs []string
	if response.Continuation != nil || len(response.Continuations) > 0 {
		msgs = append(msgs, fmt.Sprintf("PARTIAL RESULTS: the download stopped before all matching runs were collected. Collected logs data has been written to '%s'. Read file_path and resume using continuation or every per-target entry in continuations, preserving its repository and all cursor parameters.", filePath))
		if response.Continuation != nil {
			msgs = append(msgs, response.Continuation.Message)
		}
	} else if response.CoverageStatus == "unknown" {
		msgs = append(msgs, fmt.Sprintf("PARTIAL RESULTS: collection state could not be fully interpreted. Collected logs data has been written to '%s'. Read file_path for the collected report.", filePath))
	} else if response.Partial {
		msgs = append(msgs, fmt.Sprintf("PARTIAL RESULTS: episode collection or evidence has known gaps. Collected logs data has been written to '%s'. Read file_path for episode coverage and episode_retrieval bounds, counters, stop reasons and issues; no resumable cursor is available.", filePath))
	} else {
		msgs = append(msgs, fmt.Sprintf("Logs data has been written to '%s'. Use file_path to read the collected report.", filePath))
	}
	if response.CoverageStatus != "" {
		msgs = append(msgs, "Episode membership coverage is "+response.CoverageStatus+"; the selected runs do not establish complete episode membership.")
	}
	// Surface the stale-data warning (when no date range was requested and the
	// newest run returned is unexpectedly old) directly in the tool response so
	// callers see it without having to open the file.
	if warning := extractLogsStaleWarning(outputStr); warning != "" {
		msgs = append(msgs, "WARNING: "+warning)
	}
	response.Message = strings.Join(msgs, " ")

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
