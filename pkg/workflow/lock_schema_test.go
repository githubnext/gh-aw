//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMetadataFromLockFile(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectMetadata *LockMetadata
		expectLegacy   bool
		expectError    bool
	}{
		{
			name: "valid v1 metadata",
			content: `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"abc123"}
name: test
`,
			expectMetadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "abc123",
			},
			expectLegacy: false,
			expectError:  false,
		},
		{
			name: "valid v1 metadata with spaces",
			content: `#   gh-aw-metadata:   {"schema_version":"v1","frontmatter_hash":"def456"}
name: test
`,
			expectMetadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "def456",
			},
			expectLegacy: false,
			expectError:  false,
		},
		{
			name: "legacy format with frontmatter-hash only",
			content: `# frontmatter-hash: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
name: test
`,
			expectMetadata: nil,
			expectLegacy:   true,
			expectError:    false,
		},
		{
			name: "no metadata",
			content: `name: test
on: push
`,
			expectMetadata: nil,
			expectLegacy:   false,
			expectError:    false,
		},
		{
			name: "malformed JSON metadata",
			content: `# gh-aw-metadata: {invalid json}
name: test
`,
			expectMetadata: nil,
			expectLegacy:   false,
			expectError:    true,
		},
		{
			name: "future version",
			content: `# gh-aw-metadata: {"schema_version":"v2","frontmatter_hash":"future"}
name: test
`,
			expectMetadata: &LockMetadata{
				SchemaVersion:   "v2",
				FrontmatterHash: "future",
			},
			expectLegacy: false,
			expectError:  false,
		},
		{
			name: "metadata with compiler version",
			content: `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"abc123","compiler_version":"v0.1.2"}
name: test
`,
			expectMetadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "abc123",
				CompilerVersion: "v0.1.2",
			},
			expectLegacy: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, isLegacy, err := ExtractMetadataFromLockFile(tt.content)

			if tt.expectError {
				require.Error(t, err, "Expected error for malformed metadata")
			} else {
				require.NoError(t, err, "Should not error on valid or missing metadata")
			}

			assert.Equal(t, tt.expectLegacy, isLegacy, "Legacy format detection mismatch")

			if tt.expectMetadata != nil {
				require.NotNil(t, metadata, "Expected metadata to be parsed")
				assert.Equal(t, tt.expectMetadata.SchemaVersion, metadata.SchemaVersion, "Schema version mismatch")
				assert.Equal(t, tt.expectMetadata.FrontmatterHash, metadata.FrontmatterHash, "Frontmatter hash mismatch")
				assert.Equal(t, tt.expectMetadata.CompilerVersion, metadata.CompilerVersion, "Compiler version mismatch")
			} else if !tt.expectError {
				assert.Nil(t, metadata, "Expected nil metadata")
			}
		})
	}
}

func TestValidateLockSchemaCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		lockPath    string
		expectError bool
		errorText   string
	}{
		{
			name: "valid v1 schema",
			content: `# gh-aw-metadata: {"schema_version":"v1"}
name: test
`,
			lockPath:    "test.lock.yml",
			expectError: false,
		},
		{
			name: "legacy format is accepted",
			content: `# frontmatter-hash: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
name: test
`,
			lockPath:    "legacy.lock.yml",
			expectError: false,
		},
		{
			name: "unsupported future version fails",
			content: `# gh-aw-metadata: {"schema_version":"v2"}
name: test
`,
			lockPath:    "future.lock.yml",
			expectError: true,
			errorText:   "unsupported schema version 'v2'",
		},
		{
			name: "missing metadata fails",
			content: `name: test
on: push
`,
			lockPath:    "no-metadata.lock.yml",
			expectError: true,
			errorText:   "missing required metadata",
		},
		{
			name: "malformed metadata fails",
			content: `# gh-aw-metadata: {bad json}
name: test
`,
			lockPath:    "malformed.lock.yml",
			expectError: true,
			errorText:   "failed to parse lock metadata JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLockSchemaCompatibility(tt.content, tt.lockPath)

			if tt.expectError {
				require.Error(t, err, "Expected validation error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Should not error on compatible schema")
			}
		})
	}
}

func TestIsSchemaVersionSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   LockSchemaVersion
		supported bool
	}{
		{
			name:      "v1 is supported",
			version:   LockSchemaV1,
			supported: true,
		},
		{
			name:      "v2 is not supported",
			version:   "v2",
			supported: false,
		},
		{
			name:      "v0 is not supported",
			version:   "v0",
			supported: false,
		},
		{
			name:      "empty version is not supported",
			version:   "",
			supported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSchemaVersionSupported(tt.version)
			assert.Equal(t, tt.supported, result, "Schema version support mismatch")
		})
	}
}

func TestGenerateLockMetadata(t *testing.T) {
	// Save and restore original values
	originalIsRelease := isReleaseBuild
	originalVersion := compilerVersion
	defer func() {
		isReleaseBuild = originalIsRelease
		compilerVersion = originalVersion
	}()

	// Test dev build (default)
	SetIsRelease(false)
	SetVersion("dev")
	hash := "abcd1234"
	stopTime := "2026-02-17 20:00:00"
	metadata := GenerateLockMetadata(hash, stopTime)

	assert.NotNil(t, metadata, "Metadata should be created")
	assert.Equal(t, LockSchemaV1, metadata.SchemaVersion, "Should use current schema version")
	assert.Equal(t, hash, metadata.FrontmatterHash, "Should preserve frontmatter hash")
	assert.Equal(t, stopTime, metadata.StopTime, "Should preserve stop time")
	assert.Empty(t, metadata.CompilerVersion, "Dev builds should not include version")
}

func TestGenerateLockMetadataReleaseBuild(t *testing.T) {
	// Save and restore original values
	originalIsRelease := isReleaseBuild
	originalVersion := compilerVersion
	defer func() {
		isReleaseBuild = originalIsRelease
		compilerVersion = originalVersion
	}()

	// Test release build
	SetIsRelease(true)
	SetVersion("v0.1.2")
	hash := "abcd1234"
	stopTime := "2026-02-17 20:00:00"
	metadata := GenerateLockMetadata(hash, stopTime)

	assert.NotNil(t, metadata, "Metadata should be created")
	assert.Equal(t, LockSchemaV1, metadata.SchemaVersion, "Should use current schema version")
	assert.Equal(t, hash, metadata.FrontmatterHash, "Should preserve frontmatter hash")
	assert.Equal(t, stopTime, metadata.StopTime, "Should preserve stop time")
	assert.Equal(t, "v0.1.2", metadata.CompilerVersion, "Release builds should include version")
}

func TestGenerateLockMetadataWithoutStopTime(t *testing.T) {
	hash := "abcd1234"
	metadata := GenerateLockMetadata(hash, "")

	assert.NotNil(t, metadata, "Metadata should be created")
	assert.Equal(t, LockSchemaV1, metadata.SchemaVersion, "Should use current schema version")
	assert.Equal(t, hash, metadata.FrontmatterHash, "Should preserve frontmatter hash")
	assert.Empty(t, metadata.StopTime, "Stop time should be empty")
}

func TestLockMetadataToJSON(t *testing.T) {
	tests := []struct {
		name     string
		metadata *LockMetadata
		contains []string
	}{
		{
			name: "basic metadata",
			metadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "test123",
			},
			contains: []string{
				`"schema_version":"v1"`,
				`"frontmatter_hash":"test123"`,
			},
		},
		{
			name: "metadata with empty hash",
			metadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "",
			},
			contains: []string{
				`"schema_version":"v1"`,
			},
		},
		{
			name: "metadata with compiler version",
			metadata: &LockMetadata{
				SchemaVersion:   LockSchemaV1,
				FrontmatterHash: "test123",
				CompilerVersion: "v0.1.2",
			},
			contains: []string{
				`"schema_version":"v1"`,
				`"frontmatter_hash":"test123"`,
				`"compiler_version":"v0.1.2"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, err := tt.metadata.ToJSON()
			require.NoError(t, err, "Should serialize to JSON without error")

			for _, expected := range tt.contains {
				assert.Contains(t, json, expected, "JSON should contain expected field")
			}
		})
	}
}

func TestValidationErrorMessages(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		lockPath         string
		expectedMessages []string
	}{
		{
			name: "future version error has remediation",
			content: `# gh-aw-metadata: {"schema_version":"v99"}
name: test
`,
			lockPath: "future.lock.yml",
			expectedMessages: []string{
				"unsupported schema version 'v99'",
				"Upgrade gh-aw",
				"gh extension upgrade gh-aw",
				"gh aw compile future.md",
			},
		},
		{
			name: "missing metadata error has remediation",
			content: `name: test
on: push
`,
			lockPath: "missing.lock.yml",
			expectedMessages: []string{
				"missing required metadata",
				"recompile the workflow",
				"gh aw compile missing.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLockSchemaCompatibility(tt.content, tt.lockPath)
			require.Error(t, err, "Should have validation error")

			errMsg := err.Error()
			for _, expected := range tt.expectedMessages {
				assert.Contains(t, errMsg, expected, "Error message should contain remediation guidance")
			}
		})
	}
}

func TestExtractMetadataRealisticLockFile(t *testing.T) {
	// Realistic lock file content with metadata
	content := `#
#    ___                   _   _      
#   / _ \                 | | (_)     
#  | |_| | __ _  ___ _ __ | |_ _  ___ 
#
# This file was automatically generated by gh-aw. DO NOT EDIT.
#
# Daily status report for gh-aw project
#
# Resolved workflow manifest:
#   Imports:
#     - shared/reporting.md
#
# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"49266e50774d7e6a8b1c50f64b2f790c214dcdcf7b75b6bc8478bb43257b9863"}

name: "Dev"
on:
  schedule:
    - cron: "0 9 * * *"
`

	metadata, isLegacy, err := ExtractMetadataFromLockFile(content)
	require.NoError(t, err, "Should parse realistic lock file")
	assert.False(t, isLegacy, "Should not detect as legacy")
	require.NotNil(t, metadata, "Should extract metadata")
	assert.Equal(t, LockSchemaV1, metadata.SchemaVersion)
	assert.Equal(t, "49266e50774d7e6a8b1c50f64b2f790c214dcdcf7b75b6bc8478bb43257b9863", metadata.FrontmatterHash)
}

func TestExtractMetadataLegacyLockFile(t *testing.T) {
	// Legacy lock file without metadata
	content := `#
# This file was automatically generated by gh-aw. DO NOT EDIT.
#
# frontmatter-hash: 49266e50774d7e6a8b1c50f64b2f790c214dcdcf7b75b6bc8478bb43257b9863

name: "Legacy"
on: push
`

	metadata, isLegacy, err := ExtractMetadataFromLockFile(content)
	require.NoError(t, err, "Should parse legacy lock file")
	assert.True(t, isLegacy, "Should detect as legacy")
	assert.Nil(t, metadata, "Should not extract metadata from legacy")

	// Legacy format should validate successfully
	err = ValidateLockSchemaCompatibility(content, "legacy.lock.yml")
	assert.NoError(t, err, "Legacy lock files should be accepted")
}

func TestMetadataPreservesHash(t *testing.T) {
	// Test that frontmatter hash is preserved in metadata
	originalHash := "abc123def456"
	content := `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"` + originalHash + `"}
name: test
`

	metadata, _, err := ExtractMetadataFromLockFile(content)
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, originalHash, metadata.FrontmatterHash, "Should preserve frontmatter hash exactly")
}

func TestFormatSupportedVersions(t *testing.T) {
	formatted := formatSupportedVersions()
	assert.NotEmpty(t, formatted, "Should format versions")
	assert.Contains(t, formatted, "v1", "Should include v1")
}

func TestLockMetadataJSONCompact(t *testing.T) {
	// Verify JSON is compact (no newlines) for single-line comment embedding
	metadata := &LockMetadata{
		SchemaVersion:   LockSchemaV1,
		FrontmatterHash: "test",
	}

	json, err := metadata.ToJSON()
	require.NoError(t, err)
	assert.NotContains(t, json, "\n", "JSON should be compact without newlines")
	assert.NotContains(t, json, "  ", "JSON should not have extra spaces")
}

func TestSchemaVersionAsString(t *testing.T) {
	// Verify LockSchemaVersion can be used as string
	version := LockSchemaV1
	assert.Equal(t, "v1", string(version))
}

func TestExtractMetadataWithStopTime(t *testing.T) {
	// Test extracting metadata that includes stop time
	stopTime := "2026-02-17 20:00:00"
	content := `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"abc123","stop_time":"` + stopTime + `"}
name: test
`

	metadata, isLegacy, err := ExtractMetadataFromLockFile(content)
	require.NoError(t, err)
	assert.False(t, isLegacy)
	require.NotNil(t, metadata)
	assert.Equal(t, LockSchemaV1, metadata.SchemaVersion)
	assert.Equal(t, "abc123", metadata.FrontmatterHash)
	assert.Equal(t, stopTime, metadata.StopTime, "Should extract stop time from metadata")
}

func TestLockMetadataToJSONWithStopTime(t *testing.T) {
	// Test JSON serialization includes stop time when present
	metadata := &LockMetadata{
		SchemaVersion:   LockSchemaV1,
		FrontmatterHash: "test123",
		StopTime:        "2026-02-17 20:00:00",
	}

	json, err := metadata.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, json, `"schema_version":"v1"`)
	assert.Contains(t, json, `"frontmatter_hash":"test123"`)
	assert.Contains(t, json, `"stop_time":"2026-02-17 20:00:00"`)
}

func TestLockMetadataToJSONWithoutStopTime(t *testing.T) {
	// Test JSON serialization omits stop time when empty (omitempty)
	metadata := &LockMetadata{
		SchemaVersion:   LockSchemaV1,
		FrontmatterHash: "test123",
		StopTime:        "",
	}

	json, err := metadata.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, json, `"schema_version":"v1"`)
	assert.Contains(t, json, `"frontmatter_hash":"test123"`)
	// Should not contain stop_time field when empty due to omitempty
	assert.NotContains(t, json, `"stop_time"`)
}
