package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var lockSchemaLog = logger.New("workflow:lock_schema")

var (
	lockMetadataPattern = regexp.MustCompile(`#\s*gh-aw-metadata:\s*(\{.+\})`)
	lockHashPattern     = regexp.MustCompile(`#\s*frontmatter-hash:\s*([0-9a-f]{64})`)
)

// LockSchemaVersion represents a lock file schema version
type LockSchemaVersion string

const (
	// LockSchemaV1 is the legacy lock file schema version (no strict field)
	LockSchemaV1 LockSchemaVersion = "v1"
	// LockSchemaV2 is the lock file schema version that adds the strict field
	LockSchemaV2 LockSchemaVersion = "v2"
	// LockSchemaV3 is the lock file schema version that adds agent id/model and detection agent id/model fields
	LockSchemaV3 LockSchemaVersion = "v3"
	// LockSchemaV4 is the current lock file schema version (adds lock_file_hash for tamper detection)
	LockSchemaV4 LockSchemaVersion = "v4"
)

// LockMetadata represents the structured metadata embedded in lock files
type LockMetadata struct {
	SchemaVersion       LockSchemaVersion `json:"schema_version"`
	FrontmatterHash     string            `json:"frontmatter_hash,omitempty"`
	LockFileHash        string            `json:"lock_file_hash,omitempty"`
	StopTime            string            `json:"stop_time,omitempty"`
	CompilerVersion     string            `json:"compiler_version,omitempty"`
	Strict              bool              `json:"strict,omitempty"`
	AgentID             string            `json:"agent_id,omitempty"`
	AgentModel          string            `json:"agent_model,omitempty"`
	DetectionAgentID    string            `json:"detection_agent_id,omitempty"`
	DetectionAgentModel string            `json:"detection_agent_model,omitempty"`
}

// AgentMetadataInfo holds agent and detection agent information for embedding in lock file metadata
type AgentMetadataInfo struct {
	AgentID             string
	AgentModel          string
	DetectionAgentID    string
	DetectionAgentModel string
}

// SupportedSchemaVersions lists all schema versions this build can consume
var SupportedSchemaVersions = []LockSchemaVersion{
	LockSchemaV1,
	LockSchemaV2,
	LockSchemaV3,
	LockSchemaV4,
}

// IsSchemaVersionSupported checks if a schema version is supported
func IsSchemaVersionSupported(version LockSchemaVersion) bool {
	return slices.Contains(SupportedSchemaVersions, version)
}

// ExtractMetadataFromLockFile extracts structured metadata from a lock file's comment header
// Returns metadata and whether legacy format (no metadata) was detected
func ExtractMetadataFromLockFile(content string) (*LockMetadata, bool, error) {
	// Look for JSON metadata in comments (format: # gh-aw-metadata: {...})
	// Use .+ to capture to end of line since metadata is single-line JSON
	matches := lockMetadataPattern.FindStringSubmatch(content)

	if len(matches) >= 2 {
		jsonStr := matches[1]
		var metadata LockMetadata
		if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
			return nil, false, fmt.Errorf("failed to parse lock metadata JSON: %w", err)
		}
		lockSchemaLog.Printf("Extracted metadata from lock file: schema=%s", metadata.SchemaVersion)
		return &metadata, false, nil
	}

	// Legacy format: look for frontmatter-hash without JSON metadata
	if matches := lockHashPattern.FindStringSubmatch(content); len(matches) >= 2 {
		lockSchemaLog.Print("Legacy lock file detected (no schema version)")
		// Return a minimal metadata struct with just the hash for legacy files
		return &LockMetadata{FrontmatterHash: matches[1]}, true, nil
	}

	// No metadata found at all
	return nil, false, nil
}

// formatSupportedVersions formats the list of supported versions for error messages
func formatSupportedVersions() string {
	versions := make([]string, len(SupportedSchemaVersions))
	for i, v := range SupportedSchemaVersions {
		versions[i] = string(v)
	}
	return strings.Join(versions, ", ")
}

// GenerateLockMetadata creates a LockMetadata struct for embedding in lock files
// For release builds, the compiler version is included in the metadata
func GenerateLockMetadata(frontmatterHash string, stopTime string, strict bool, agentInfo AgentMetadataInfo) *LockMetadata {
	lockSchemaLog.Printf("Generating lock metadata: schema=%s, strict=%t, hasStopTime=%t", LockSchemaV3, strict, stopTime != "")

	metadata := &LockMetadata{
		SchemaVersion:       LockSchemaV3,
		FrontmatterHash:     frontmatterHash,
		StopTime:            stopTime,
		Strict:              strict,
		AgentID:             agentInfo.AgentID,
		AgentModel:          agentInfo.AgentModel,
		DetectionAgentID:    agentInfo.DetectionAgentID,
		DetectionAgentModel: agentInfo.DetectionAgentModel,
	}

	// Include compiler version only for release builds
	if IsRelease() {
		metadata.CompilerVersion = GetVersion()
		lockSchemaLog.Printf("Including compiler version in lock metadata: %s", metadata.CompilerVersion)
	}

	return metadata
}

// ToJSON converts LockMetadata to a compact JSON string for embedding in comments
func (m *LockMetadata) ToJSON() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to serialize lock metadata: %w", err)
	}
	return string(bytes), nil
}

// metadataLinePrefix is the prefix used for the metadata comment line
const metadataLinePrefix = "# gh-aw-metadata: "

// InjectLockFileHash computes the SHA-256 hash of the YAML content after the metadata
// comment line and injects it into the metadata as "lock_file_hash", upgrading the
// schema version to LockSchemaV4. This hash covers the entire compiled YAML body and
// provides tamper evidence: any modification to the YAML after compilation will cause
// the stored hash to no longer match the recomputed one.
//
// If the content does not start with a metadata comment line, it is returned unchanged.
func InjectLockFileHash(yamlContent string) (string, error) {
	// Split on the first newline to isolate the metadata line from the rest
	firstLine, rest, found := strings.Cut(yamlContent, "\n")
	if !found {
		// Single-line or empty content — nothing to hash
		return yamlContent, nil
	}

	// Only process if the first line is a metadata comment
	matches := lockMetadataPattern.FindStringSubmatch(firstLine)
	if len(matches) < 2 {
		return yamlContent, nil
	}

	// Parse existing metadata
	var metadata LockMetadata
	if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
		return yamlContent, fmt.Errorf("failed to parse lock metadata for hash injection: %w", err)
	}

	// Compute SHA-256 hash of all content after the metadata line.
	// This covers the entire compiled YAML body (excluding the metadata comment itself),
	// so any post-compilation edit to the body will invalidate the stored hash.
	sum := sha256.Sum256([]byte(rest))
	metadata.LockFileHash = hex.EncodeToString(sum[:])
	metadata.SchemaVersion = LockSchemaV4

	// Re-serialize metadata
	updatedJSON, err := json.Marshal(metadata)
	if err != nil {
		return yamlContent, fmt.Errorf("failed to serialize updated lock metadata: %w", err)
	}

	// Reconstruct the first line with the updated metadata
	updatedFirstLine := metadataLinePrefix + string(updatedJSON)
	lockSchemaLog.Printf("Injected lock_file_hash into metadata (schema bumped to %s)", LockSchemaV4)
	return updatedFirstLine + "\n" + rest, nil
}
