package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var artifactManagerLog = logger.New("workflow:artifact_manager")

// ArtifactManager simulates the behavior of actions/upload-artifact and actions/download-artifact
// to track artifacts and compute actual file locations during compilation.
//
// This manager implements the v4 behavior of GitHub Actions artifacts:
// - Upload: artifacts are immutable, each upload creates a new artifact
// - Download: files extracted directly to path (not path/artifact-name/)
// - Pattern downloads: separate subdirectories unless merge-multiple is used
type ArtifactManager struct {
	// uploads tracks all artifact uploads by job name
	uploads map[string][]*ArtifactUpload

	// downloads tracks all artifact downloads by job name
	downloads map[string][]*ArtifactDownload

	// currentJob tracks the job currently being processed
	currentJob string
}

// ArtifactUpload represents an artifact upload operation
type ArtifactUpload struct {
	// Name is the artifact name (e.g., "agent-artifacts")
	Name string

	// Paths are the file/directory paths being uploaded
	// These can be absolute paths or glob patterns
	Paths []string

	// NormalizedPaths are the paths after common parent directory removal
	// This simulates GitHub Actions behavior where the common parent is stripped
	NormalizedPaths map[string]string

	// IfNoFilesFound specifies behavior when no files match
	// Values: "warn", "error", "ignore"
	IfNoFilesFound string

	// IncludeHiddenFiles determines if hidden files are included
	IncludeHiddenFiles bool

	// JobName is the name of the job uploading this artifact
	JobName string
}

// ArtifactDownload represents an artifact download operation
type ArtifactDownload struct {
	// Name is the artifact name to download (optional if using Pattern)
	Name string

	// Pattern is a glob pattern to match multiple artifacts (v4 feature)
	Pattern string

	// Path is where the artifact will be downloaded
	Path string

	// MergeMultiple determines if multiple artifacts should be merged
	// into the same directory (only applies when using Pattern)
	MergeMultiple bool

	// JobName is the name of the job downloading this artifact
	JobName string

	// DependsOn lists job names this job depends on (from needs:)
	DependsOn []string
}

// ArtifactFile represents a file within an artifact
type ArtifactFile struct {
	// ArtifactName is the name of the artifact containing this file
	ArtifactName string

	// OriginalPath is the path as uploaded
	OriginalPath string

	// DownloadPath is the computed path after download
	DownloadPath string

	// JobName is the job that uploaded this file
	JobName string
}

// NewArtifactManager creates a new artifact manager
func NewArtifactManager() *ArtifactManager {
	return &ArtifactManager{
		uploads:   make(map[string][]*ArtifactUpload),
		downloads: make(map[string][]*ArtifactDownload),
	}
}

//	/tmp/gh-aw/aw-prompts/prompt.txt
//	/tmp/gh-aw/aw.patch
//	aw-prompts/prompt.txt
//	aw.patch

// Reset clears all tracked uploads and downloads
func (am *ArtifactManager) Reset() {
	am.uploads = make(map[string][]*ArtifactUpload)
	am.downloads = make(map[string][]*ArtifactDownload)
	am.currentJob = ""
	artifactManagerLog.Print("Reset artifact manager")
}
