package workflow

import (
	"fmt"
	"path/filepath"
	"strings"
)

// StepType represents the type of step being generated
type StepType int

const (
	StepTypeSecretRedaction StepType = iota
	StepTypeArtifactUpload
	StepTypeOther
)

// StepRecord tracks a step that was generated during compilation
type StepRecord struct {
	Type        StepType
	Name        string
	Order       int      // Order in which this step was added
	UploadPaths []string // For artifact upload steps, the paths being uploaded
}

// StepOrderTracker tracks the order of steps generated during compilation
type StepOrderTracker struct {
	steps                []StepRecord
	nextOrder            int
	secretRedactionAdded bool
	secretRedactionOrder int
	afterAgentExecution  bool // Track whether we're after agent execution step
}

// NewStepOrderTracker creates a new step order tracker
func NewStepOrderTracker() *StepOrderTracker {
	return &StepOrderTracker{
		steps:     make([]StepRecord, 0),
		nextOrder: 0,
	}
}

// MarkAgentExecutionComplete marks that we've passed the agent execution step
// Validation only applies to steps after this point
func (t *StepOrderTracker) MarkAgentExecutionComplete() {
	t.afterAgentExecution = true
}

// RecordSecretRedaction records that a secret redaction step was added
func (t *StepOrderTracker) RecordSecretRedaction(stepName string) {
	if !t.afterAgentExecution {
		// Only track steps after agent execution
		return
	}

	t.steps = append(t.steps, StepRecord{
		Type:  StepTypeSecretRedaction,
		Name:  stepName,
		Order: t.nextOrder,
	})
	t.secretRedactionAdded = true
	t.secretRedactionOrder = t.nextOrder
	t.nextOrder++
}

// RecordArtifactUpload records that an artifact upload step was added
func (t *StepOrderTracker) RecordArtifactUpload(stepName string, uploadPaths []string) {
	if !t.afterAgentExecution {
		// Only track steps after agent execution
		return
	}

	t.steps = append(t.steps, StepRecord{
		Type:        StepTypeArtifactUpload,
		Name:        stepName,
		Order:       t.nextOrder,
		UploadPaths: uploadPaths,
	})
	t.nextOrder++
}

// ValidateStepOrdering validates that secret redaction happens before artifact uploads
// and that all uploaded paths are covered by secret redaction
func (t *StepOrderTracker) ValidateStepOrdering() error {
	// If we haven't reached agent execution yet, no validation needed
	if !t.afterAgentExecution {
		return nil
	}

	// Find all artifact uploads
	var artifactUploads []StepRecord
	for _, step := range t.steps {
		if step.Type == StepTypeArtifactUpload {
			artifactUploads = append(artifactUploads, step)
		}
	}

	// If no artifact uploads, no validation needed
	if len(artifactUploads) == 0 {
		return nil
	}

	// If there are artifact uploads but no secret redaction, that's a bug
	if !t.secretRedactionAdded {
		return fmt.Errorf("compiler bug: artifact uploads found but no secret redaction step was added (this is a critical security issue)")
	}

	// Check that secret redaction comes before all artifact uploads
	var uploadsBeforeRedaction []string
	for _, upload := range artifactUploads {
		if upload.Order < t.secretRedactionOrder {
			uploadsBeforeRedaction = append(uploadsBeforeRedaction, upload.Name)
		}
	}

	if len(uploadsBeforeRedaction) > 0 {
		return fmt.Errorf("compiler bug: secret redaction must happen before artifact uploads, but found %d upload(s) before redaction: %s",
			len(uploadsBeforeRedaction), strings.Join(uploadsBeforeRedaction, ", "))
	}

	// Check that all uploaded paths are covered by secret redaction
	// Secret redaction scans all files in /tmp/gh-aw/ with extensions .txt, .json, .log
	unscannable := t.findUnscannablePaths(artifactUploads)
	if len(unscannable) > 0 {
		return fmt.Errorf("compiler bug: the following artifact upload paths are not covered by secret redaction: %s",
			strings.Join(unscannable, ", "))
	}

	return nil
}

// findUnscannablePaths finds paths that would be uploaded but not scanned by secret redaction
func (t *StepOrderTracker) findUnscannablePaths(artifactUploads []StepRecord) []string {
	var unscannable []string

	for _, upload := range artifactUploads {
		for _, path := range upload.UploadPaths {
			// Check if this path would be scanned by secret redaction
			// Secret redaction only scans:
			// 1. Files under /tmp/gh-aw/
			// 2. With extensions .txt, .json, .log
			if !isPathScannedBySecretRedaction(path) {
				unscannable = append(unscannable, path)
			}
		}
	}

	return unscannable
}

// isPathScannedBySecretRedaction checks if a path would be scanned by the secret redaction step
func isPathScannedBySecretRedaction(path string) bool {
	// Paths must be under /tmp/gh-aw/ to be scanned
	// Accept both literal paths and environment variable references
	if !strings.HasPrefix(path, "/tmp/gh-aw/") {
		// Check if it's an environment variable that might resolve to /tmp/gh-aw/
		// For now, we'll allow ${{ env.* }} patterns through as we can't resolve them at compile time
		if strings.Contains(path, "${{ env.") {
			// Assume environment variables that might contain /tmp/gh-aw paths are safe
			// This is a conservative assumption - in practice these are controlled by the compiler
			return true
		}
		return false
	}

	// Path must have one of the scanned extensions: .txt, .json, .log
	ext := filepath.Ext(path)
	scannedExtensions := []string{".txt", ".json", ".log", ".jsonl"}
	for _, scannedExt := range scannedExtensions {
		if ext == scannedExt {
			return true
		}
	}

	// If path is a directory (ends with /), we assume it contains scannable files
	if strings.HasSuffix(path, "/") {
		return true
	}

	// Path doesn't have a scannable extension
	return false
}
