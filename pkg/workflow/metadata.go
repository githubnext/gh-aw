package workflow

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var metadataLog = logger.New("workflow:metadata")

// LockMetadata represents the metadata stored in a lock file
type LockMetadata struct {
	SourceHash       string
	CompiledAt       string
	GhAwVersion      string
	DependenciesHash string
}

// StaleStatus represents the staleness status of a lock file
type StaleStatus string

const (
	StaleStatusUpToDate      StaleStatus = "up-to-date"
	StaleStatusNeverCompiled StaleStatus = "never-compiled"
	StaleStatusHashMismatch  StaleStatus = "stale-hash"
	StaleStatusTimestamp     StaleStatus = "stale-timestamp"
)

// ComputeSourceHash computes a SHA-256 hash of the source file content
// Returns the hash as a hex string prefixed with "sha256:"
func ComputeSourceHash(filepath string) (string, error) {
	metadataLog.Printf("Computing source hash for: %s", filepath)

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	hashString := fmt.Sprintf("sha256:%x", hash.Sum(nil))
	hashPreview := hashString
	if len(hashPreview) > 20 {
		hashPreview = hashPreview[:20] + "..."
	}
	metadataLog.Printf("Computed hash: %s", hashPreview)
	return hashString, nil
}

// ExtractLockFileMetadata parses metadata from a lock file
// Returns nil if no metadata is found (for backward compatibility)
func ExtractLockFileMetadata(lockFilePath string) (*LockMetadata, error) {
	metadataLog.Printf("Extracting metadata from: %s", lockFilePath)

	file, err := os.Open(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	defer file.Close()

	metadata := &LockMetadata{}
	scanner := bufio.NewScanner(file)
	inMetadataBlock := false
	foundAnyMetadata := false

	// Regex patterns for metadata fields
	metadataHeaderRegex := regexp.MustCompile(`^#\s*Metadata:\s*$`)
	sourceHashRegex := regexp.MustCompile(`^#\s*source_hash:\s*(.+)\s*$`)
	compiledAtRegex := regexp.MustCompile(`^#\s*compiled_at:\s*(.+)\s*$`)
	versionRegex := regexp.MustCompile(`^#\s*gh_aw_version:\s*(.+)\s*$`)
	depsHashRegex := regexp.MustCompile(`^#\s*dependencies_hash:\s*(.+)\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if we're entering the metadata block
		if metadataHeaderRegex.MatchString(line) {
			inMetadataBlock = true
			metadataLog.Print("Found metadata block")
			continue
		}

		// If we're in the metadata block, parse fields
		if inMetadataBlock {
			// Exit metadata block if we hit a non-comment line or empty comment
			if !strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "#" {
				break
			}

			if matches := sourceHashRegex.FindStringSubmatch(line); len(matches) > 1 {
				metadata.SourceHash = strings.TrimSpace(matches[1])
				foundAnyMetadata = true
				hashPreview := metadata.SourceHash
				if len(hashPreview) > 20 {
					hashPreview = hashPreview[:20] + "..."
				}
				metadataLog.Printf("Found source_hash: %s", hashPreview)
			} else if matches := compiledAtRegex.FindStringSubmatch(line); len(matches) > 1 {
				metadata.CompiledAt = strings.TrimSpace(matches[1])
				foundAnyMetadata = true
				metadataLog.Printf("Found compiled_at: %s", metadata.CompiledAt)
			} else if matches := versionRegex.FindStringSubmatch(line); len(matches) > 1 {
				metadata.GhAwVersion = strings.TrimSpace(matches[1])
				foundAnyMetadata = true
				metadataLog.Printf("Found gh_aw_version: %s", metadata.GhAwVersion)
			} else if matches := depsHashRegex.FindStringSubmatch(line); len(matches) > 1 {
				metadata.DependenciesHash = strings.TrimSpace(matches[1])
				foundAnyMetadata = true
				depsHashPreview := metadata.DependenciesHash
				if len(depsHashPreview) > 20 {
					depsHashPreview = depsHashPreview[:20] + "..."
				}
				metadataLog.Printf("Found dependencies_hash: %s", depsHashPreview)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lock file: %w", err)
	}

	// Return nil if no metadata was found (backward compatibility)
	if !foundAnyMetadata {
		metadataLog.Print("No metadata found in lock file")
		return nil, nil
	}

	return metadata, nil
}

// CompareLockFileToSource compares a lock file to its source file
// Returns the staleness status and an error if any
func CompareLockFileToSource(lockFile, sourceFile string) (StaleStatus, error) {
	metadataLog.Printf("Comparing lock file to source: %s -> %s", lockFile, sourceFile)

	// Check if lock file exists
	lockStat, err := os.Stat(lockFile)
	if os.IsNotExist(err) {
		metadataLog.Print("Lock file does not exist")
		return StaleStatusNeverCompiled, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat lock file: %w", err)
	}

	// Check if source file exists
	sourceStat, err := os.Stat(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to stat source file: %w", err)
	}

	// Try to extract metadata from lock file
	metadata, err := ExtractLockFileMetadata(lockFile)
	if err != nil {
		return "", fmt.Errorf("failed to extract metadata: %w", err)
	}

	// If metadata exists, use hash-based comparison
	if metadata != nil && metadata.SourceHash != "" {
		metadataLog.Print("Using hash-based staleness detection")

		// Compute current source hash
		currentHash, err := ComputeSourceHash(sourceFile)
		if err != nil {
			return "", fmt.Errorf("failed to compute source hash: %w", err)
		}

		// Compare hashes
		if currentHash == metadata.SourceHash {
			metadataLog.Print("Hashes match - file is up-to-date")
			return StaleStatusUpToDate, nil
		}

		metadataLog.Print("Hashes differ - file is stale")
		return StaleStatusHashMismatch, nil
	}

	// Fall back to timestamp-based comparison (backward compatibility)
	metadataLog.Print("Using timestamp-based staleness detection (no metadata found)")
	if sourceStat.ModTime().After(lockStat.ModTime()) {
		metadataLog.Print("Source is newer than lock file - file is stale")
		return StaleStatusTimestamp, nil
	}

	metadataLog.Print("Source is not newer than lock file - file is up-to-date")
	return StaleStatusUpToDate, nil
}

// GenerateMetadataComment generates a metadata comment block for a lock file
func GenerateMetadataComment(sourceFile, version string, dependencies []string) (string, error) {
	metadataLog.Printf("Generating metadata comment for: %s", sourceFile)

	// Compute source hash
	sourceHash, err := ComputeSourceHash(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to compute source hash: %w", err)
	}

	// Generate compilation timestamp in ISO 8601 format
	compiledAt := time.Now().UTC().Format(time.RFC3339)

	// Compute dependencies hash
	depsHash := computeDependenciesHash(dependencies)

	var builder strings.Builder
	builder.WriteString("# Metadata:\n")
	builder.WriteString(fmt.Sprintf("#   source_hash: %s\n", sourceHash))
	builder.WriteString(fmt.Sprintf("#   compiled_at: %s\n", compiledAt))
	builder.WriteString(fmt.Sprintf("#   gh_aw_version: %s\n", version))
	builder.WriteString(fmt.Sprintf("#   dependencies_hash: %s\n", depsHash))

	metadataLog.Print("Metadata comment generated successfully")
	return builder.String(), nil
}

// computeDependenciesHash computes a hash of the dependency list
func computeDependenciesHash(dependencies []string) string {
	if len(dependencies) == 0 {
		return "sha256:none"
	}

	hash := sha256.New()
	for _, dep := range dependencies {
		hash.Write([]byte(dep))
		hash.Write([]byte("\n"))
	}

	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}
