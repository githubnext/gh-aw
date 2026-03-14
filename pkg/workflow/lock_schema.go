package workflow

import (
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
	// LockSchemaV2 is the current lock file schema version (adds strict field)
	LockSchemaV2 LockSchemaVersion = "v2"
)

// LockMetadata represents the structured metadata embedded in lock files
type LockMetadata struct {
	SchemaVersion   LockSchemaVersion `json:"schema_version"`
	FrontmatterHash string            `json:"frontmatter_hash,omitempty"`
	StopTime        string            `json:"stop_time,omitempty"`
	CompilerVersion string            `json:"compiler_version,omitempty"`
	Strict          bool              `json:"strict,omitempty"`
}

// SupportedSchemaVersions lists all schema versions this build can consume
var SupportedSchemaVersions = []LockSchemaVersion{
	LockSchemaV1,
	LockSchemaV2,
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
func GenerateLockMetadata(frontmatterHash string, stopTime string, strict bool) *LockMetadata {
	metadata := &LockMetadata{
		SchemaVersion:   LockSchemaV2,
		FrontmatterHash: frontmatterHash,
		StopTime:        stopTime,
		Strict:          strict,
	}

	// Include compiler version only for release builds
	if IsRelease() {
		metadata.CompilerVersion = GetVersion()
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
